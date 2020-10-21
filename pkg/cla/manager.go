// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
// SPDX-FileCopyrightText: 2020 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cla

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
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

	listenerIDs map[CLAType][]bpv7.EndpointID

	// providers is an array of ConvergenceProvider. Those will report their
	// created Convergence objects to this Manager, which also supervises it.
	providers      []ConvergenceProvider
	providersMutex sync.Mutex

	// inChnl receives ConvergenceStatus while outChnl passes it on. Both channels
	// are not buffered. While this is not a problem for inChnl, outChnl must
	// always be read, otherwise the Manager will block.
	inChnl  chan ConvergenceStatus
	outChnl chan ConvergenceStatus

	// stop{Syn,Ack} are used to supervise closing this Manager, see Close()
	stopSyn chan struct{}
	stopAck chan struct{}

	// stopFlag and its mutex protect the Manager against acting on new CLAs
	// after the Close method was called once.
	stopFlag      bool
	stopFlagMutex sync.Mutex
}

// NewManager creates a new Manager to supervise different CLAs.
func NewManager() *Manager {
	manager := &Manager{
		queueTtl:  10,
		retryTime: 10 * time.Second,

		convs: new(sync.Map),

		listenerIDs: make(map[CLAType][]bpv7.EndpointID),

		inChnl:  make(chan ConvergenceStatus, 100),
		outChnl: make(chan ConvergenceStatus),

		stopSyn: make(chan struct{}),
		stopAck: make(chan struct{}),

		stopFlag: false,
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
			log.Debug("CLA Manager received closing signal")

			manager.convs.Range(func(_, convElem interface{}) bool {
				manager.Unregister(convElem.(*convergenceElem).conv)
				return true
			})

			manager.providersMutex.Lock()
			for _, provider := range manager.providers {
				_ = provider.Close()
			}
			manager.providersMutex.Unlock()

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
					"endpoint": cs.Message.(bpv7.EndpointID),
				}).Info("CLA Manager received Peer Disappeared, restarting CLA")

				manager.Restart(cs.Sender)
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

// isStopped signals if the Manager should be stopped.
func (manager *Manager) isStopped() bool {
	manager.stopFlagMutex.Lock()
	defer manager.stopFlagMutex.Unlock()

	return manager.stopFlag
}

// Close the Manager and all supervised CLAs.
func (manager *Manager) Close() error {
	manager.stopFlagMutex.Lock()
	manager.stopFlag = true
	manager.stopFlagMutex.Unlock()

	close(manager.stopSyn)
	<-manager.stopAck

	return nil
}

// Register any kind of Convergable.
func (manager *Manager) Register(conv Convergable) {
	if manager.isStopped() {
		return
	}

	if c, ok := conv.(Convergence); ok {
		manager.registerConvergence(c)
	} else if c, ok := conv.(ConvergenceProvider); ok {
		manager.registerProvider(c)
	} else {
		log.WithField("convergence", conv).Warn("Unknown kind of Convergable")
	}
}

func (manager *Manager) registerConvergence(conv Convergence) {
	// Check if this CLA is already known. Re-activate a deactivated CLA or abort.
	var ce *convergenceElem
	if convElem, exists := manager.convs.Load(conv.Address()); exists {
		ce = convElem.(*convergenceElem)
		if ce.isActive() {
			log.WithFields(log.Fields{
				"cla":     conv,
				"address": conv.Address(),
			}).Debug("CLA registration failed, because this address does already exists")

			return
		}
	} else {
		ce = newConvergenceElement(conv, manager.inChnl, manager.queueTtl)
	}

	// Check if this CLA is a sender to a registered receiver.
	if cs, ok := ce.asSender(); ok {
		for _, cr := range manager.Receiver() {
			if cr.GetEndpointID() == cs.GetPeerEndpointID() {
				log.WithFields(log.Fields{
					"cla":     conv,
					"address": conv.Address(),
				}).Debug("CLA registration aborted, because of a known Endpoint ID")

				return
			}
		}
	}

	if successful, retry := ce.activate(); !successful && !retry {
		log.WithFields(log.Fields{
			"cla":     conv,
			"address": conv.Address(),
		}).Warn("Startup of CLA  failed, a retry should not be made")
	} else {
		manager.convs.Store(conv.Address(), ce)
	}
}

func (manager *Manager) registerProvider(conv ConvergenceProvider) {
	manager.providersMutex.Lock()
	defer manager.providersMutex.Unlock()

	for _, provider := range manager.providers {
		if conv == provider {
			log.WithField("provider", conv).Debug("Provider registration aborted, already known")
			return
		}
	}

	manager.providers = append(manager.providers, conv)

	conv.RegisterManager(manager)

	if err := conv.Start(); err != nil {
		log.WithError(err).WithField("provider", conv).Warn("Starting Provider errored")
	}
}

// Unregister any kind of Convergable.
func (manager *Manager) Unregister(conv Convergable) {
	if c, ok := conv.(Convergence); ok {
		manager.unregisterConvergence(c)
	} else if c, ok := conv.(ConvergenceProvider); ok {
		manager.unregisterProvider(c)
	} else {
		log.WithField("convergence", conv).Warn("Unknown kind of Convergable")
	}
}

func (manager *Manager) unregisterConvergence(conv Convergence) {
	convElem, exists := manager.convs.Load(conv.Address())
	if !exists {
		log.WithFields(log.Fields{
			"cla":     conv,
			"address": conv.Address(),
		}).Info("CLA unregistration failed, this address does not exists")

		return
	}

	convElem.(*convergenceElem).deactivate(manager.queueTtl)
	manager.convs.Delete(conv.Address())
}

func (manager *Manager) unregisterProvider(conv ConvergenceProvider) {
	manager.providersMutex.Lock()
	defer manager.providersMutex.Unlock()

	for i, provider := range manager.providers {
		if conv == provider {
			manager.providers = append(manager.providers[:i], manager.providers[i+1:]...)
			return
		}
	}
}

// Restart a known Convergable.
func (manager *Manager) Restart(conv Convergable) {
	manager.Unregister(conv)
	manager.Register(conv)
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

func (manager *Manager) RegisterEndpointID(claType CLAType, eid bpv7.EndpointID) {
	clas, ok := manager.listenerIDs[claType]

	if ok {
		clas = append(clas, eid)
	} else {
		clas = []bpv7.EndpointID{eid}
	}

	manager.listenerIDs[claType] = clas
}

// EndpointIDs returns the EndpointIDs of all registered CLAs of the specified type.
// Returns an empty slice if no CLAs of the tye exist.
func (manager *Manager) EndpointIDs(claType CLAType) []bpv7.EndpointID {
	if clas, ok := manager.listenerIDs[claType]; ok {
		return clas
	} else {
		return make([]bpv7.EndpointID, 0)
	}
}

func (manager *Manager) HasEndpoint(endpoint bpv7.EndpointID) bool {
	for _, clas := range manager.listenerIDs {
		for _, adapter := range clas {
			if adapter.Authority() == endpoint.Authority() {
				return true
			}
		}
	}

	return false
}
