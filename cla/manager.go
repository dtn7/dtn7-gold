package cla

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

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

			manager.outChnl <- cs
			// TODO: inspect cs, act on it

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

func (manager *Manager) Channel() chan ConvergenceStatus {
	return manager.outChnl
}

func (manager *Manager) Close() {
	close(manager.stopSyn)
	<-manager.stopAck
}

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

func (manager *Manager) Unregister(conv Convergence, closeCall bool) error {
	convElem, exists := manager.convs.Load(conv.Address())
	if !exists {
		return fmt.Errorf("No CLA for address %v is available", conv.Address())
	}

	convElem.(*convergenceElem).deactivate(manager.queueTtl, closeCall)
	manager.convs.Delete(conv.Address())

	return nil
}
