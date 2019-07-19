package cla

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

// Manager monitors and manages the various CLAs, restarts them if necessary,
// and forwards the ConvergenceStatus messages. The recipient can perform
// further actions based on these, but does not have to take care of the
// CLA administration themselves.
type Manager struct {
	// queueTtl is the amount of retries for a CLA.
	queueTtl int

	// retryTime is the duration between two activation attempts.
	retryTime time.Duration

	// convs maps each CLA's address to a wrapped convergenceElem struct.
	// convs: Map[string]*convergenceElem
	convs *sync.Map

	// inChnl receives ConvergenceStatus while outChnl passes it on. Both channels
	// are not buffered. While this is not a problem for inChnl, outChnl must
	// always be read, otherwise the Manager will block.
	inChnl  chan ConvergenceStatus
	outChnl chan ConvergenceStatus

	// stop{Syn,Ack} are used to supervise closing this Manager, see Close()
	stopSyn chan struct{}
	stopAck chan struct{}
}

// NewManager creates a new Manager to supervise different CLAs.
func NewManager() *Manager {
	manager := &Manager{
		queueTtl:  10,
		retryTime: 10 * time.Second,

		convs: new(sync.Map),

		inChnl:  make(chan ConvergenceStatus),
		outChnl: make(chan ConvergenceStatus),

		stopSyn: make(chan struct{}),
		stopAck: make(chan struct{}),
	}

	go manager.handler()

	return manager
}

// handler is the internal goroutine for management.
func (manager *Manager) handler() {
	activateTicker := time.NewTicker(manager.retryTime)
	defer activateTicker.Stop()

	for {
		select {
		case <-manager.stopSyn:
			log.Debug("CLA Manager received closing-signal")

			manager.convs.Range(func(_, convElem interface{}) bool {
				_ = manager.Unregister(convElem.(*convergenceElem).conv, true)
				return true
			})

			close(manager.inChnl)
			close(manager.outChnl)

			close(manager.stopAck)
			return

		case cs := <-manager.inChnl:
			log.WithFields(log.Fields{
				"type":   cs.MessageType,
				"status": cs.String(),
			}).Debug("CLA Manager received ConvergenceStatus")

			switch cs.MessageType {
			case PeerDisappeared:
				log.WithFields(log.Fields{
					"cla":      cs.Sender,
					"endpoint": cs.Message.(bundle.EndpointID),
				}).Info("CLA Manager received Peer Disappeared, restarting CLA")

				if err := manager.Restart(cs.Sender); err != nil {
					log.WithFields(log.Fields{
						"cla":      cs.Sender,
						"endpoint": cs.Message.(bundle.EndpointID),
						"error":    err,
					}).Warn("CLA Manager failed to restart CLA")
				}

				manager.outChnl <- cs

			default:
				manager.outChnl <- cs
			}

		case <-activateTicker.C:
			manager.convs.Range(func(key, convElem interface{}) bool {
				ce := convElem.(*convergenceElem)
				if ce.isActive() {
					return true
				}

				if successful, retry := ce.activate(); !successful && !retry {
					log.WithFields(log.Fields{
						"cla": ce.conv,
					}).Warn("Startup of CLA failed, a retry should not be made")

					manager.convs.Delete(key)
				}
				return true
			})
		}
	}
}

// Channel references the outgoing channel for ConvergenceStatus messages.
func (manager *Manager) Channel() chan ConvergenceStatus {
	return manager.outChnl
}

// Close the Manager and all supervised CLAs.
func (manager *Manager) Close() {
	close(manager.stopSyn)
	<-manager.stopAck
}

// Register a new CLA.
func (manager *Manager) Register(conv Convergence) error {
	if _, exists := manager.convs.Load(conv.Address()); exists {
		return fmt.Errorf("CLA for address %v does already exists", conv.Address())
	}

	ce := newConvergenceElement(conv, manager.inChnl, manager.queueTtl)

	if successful, retry := ce.activate(); !successful && !retry {
		return fmt.Errorf("Startup of CLA %v failed, a retry should not be made", conv.Address())
	} else {
		manager.convs.Store(conv.Address(), ce)
		return nil
	}
}

// Unregister an already known CLA.
func (manager *Manager) Unregister(conv Convergence, closeCall bool) error {
	convElem, exists := manager.convs.Load(conv.Address())
	if !exists {
		return fmt.Errorf("No CLA for address %v is available", conv.Address())
	}

	convElem.(*convergenceElem).deactivate(manager.queueTtl, closeCall)
	manager.convs.Delete(conv.Address())

	return nil
}

// Restart an already known CLA.
func (manager *Manager) Restart(conv Convergence) error {
	if err := manager.Unregister(conv, true); err != nil {
		return err
	}
	if err := manager.Register(conv); err != nil {
		return err
	}
	return nil
}

// Sender returns an array of all active ConvergenceSenders.
func (manager *Manager) Sender() (css []ConvergenceSender) {
	manager.convs.Range(func(_, convElem interface{}) bool {
		ce := convElem.(*convergenceElem)
		if !ce.isActive() {
			return true
		}

		if cs, ok := ce.asSender(); ok {
			css = append(css, cs)
		}
		return true
	})
	return
}

// Receiver returns an array of all active ConvergenceReceivers.
func (manager *Manager) Receiver() (crs []ConvergenceReceiver) {
	manager.convs.Range(func(_, convElem interface{}) bool {
		ce := convElem.(*convergenceElem)
		if !ce.isActive() {
			return true
		}

		if cr, ok := ce.asReceiver(); ok {
			crs = append(crs, cr)
		}
		return true
	})
	return
}
