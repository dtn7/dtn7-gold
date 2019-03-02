package core

import (
	log "github.com/sirupsen/logrus"

	"github.com/geistesk/dtn7/cla"
)

// queueTTL is the amount of retries to add a CLA.
const queueTTL uint = 10

// convergenceQueueElement is a tuple for adding new CLAs. The TTL starts with
// queueTTL and will be decremented for each failed start. If the TTL reaches
// zero, this CLA will be removed.
type convergenceQueueElement struct {
	conv cla.Convergence
	ttl  uint
}

// newConvergenceQueueElement creates a new convergenceQueueElement for the
// given Convergence with the default queueTTL.
func newConvergenceQueueElement(conv cla.Convergence) *convergenceQueueElement {
	return &convergenceQueueElement{
		conv: conv,
		ttl:  queueTTL,
	}
}

// isReceiver checks if this stores a ConvergenceReceiver.
func (cqe *convergenceQueueElement) isReceiver() bool {
	_, ok := (cqe.conv).(cla.ConvergenceReceiver)
	return ok
}

// isReceiver checks if this stores a ConvergenceSender.
func (cqe *convergenceQueueElement) isSender() bool {
	_, ok := (cqe.conv).(cla.ConvergenceSender)
	return ok
}

// activate tries to activate the stored Convergence. The return value indicates
// if it should be tried again later.
func (cqe *convergenceQueueElement) activate(c *Core) (retry bool) {
	// always decrement cqe's ttl
	defer func() {
		if cqe.ttl > 0 {
			cqe.ttl--
		}
	}()

	// Check TTL
	if cqe.ttl == 0 {
		log.WithFields(log.Fields{
			"cla": cqe.conv,
		}).Warn("Failed to start CLA, TTL expired")

		retry = false
		return
	}

	// Check registration state (receiver and sender)
	var doReceiver, doSender bool = false, false

	if cqe.isReceiver() {
		doReceiver = true

		c.convergenceMutex.Lock()
		for _, rec := range c.convergenceReceivers {
			if rec.Address() == cqe.conv.Address() {
				log.WithFields(log.Fields{
					"cla":  cqe.conv,
					"type": "receiver",
				}).Debug("CLA's address is already known")

				doReceiver = false
				break
			}
		}
		c.convergenceMutex.Unlock()
	}

	if cqe.isSender() {
		doSender = true

		c.convergenceMutex.Lock()
		for _, sender := range c.convergenceSenders {
			if sender.Address() == cqe.conv.Address() {
				log.WithFields(log.Fields{
					"cla":  cqe.conv,
					"type": "sender",
				}).Debug("CLA's address is already known")

				doSender = false
				break
			}
		}
		c.convergenceMutex.Unlock()
	}

	if !doReceiver && !doSender {
		log.WithFields(log.Fields{
			"cla": cqe.conv,
		}).Debug("Don't continuing on CLA, it is already known")

		retry = false
		return
	}

	// Check if endpoint ID is already registered in this node
	if doSender && c.HasEndpoint((cqe.conv.(cla.ConvergenceSender)).GetPeerEndpointID()) {
		log.WithFields(log.Fields{
			"cla": cqe.conv,
		}).Debug("Node contains ConvergenceSender's endpoint ID")

		retry = false
		return
	}

	if err, claRetry := cqe.conv.Start(); err != nil {
		log.WithFields(log.Fields{
			"cla":             cqe.conv,
			"error":           err,
			"retry_requested": claRetry,
			"ttl":             cqe.ttl,
		}).Info("Failed to start CLA")

		retry = claRetry
		return
	} else {
		log.WithFields(log.Fields{
			"cla":      cqe.conv,
			"receiver": doReceiver,
			"sender":   doSender,
		}).Info("Started and registered CLA")

		c.convergenceMutex.Lock()
		if doReceiver {
			c.convergenceReceivers = append(c.convergenceReceivers, cqe.conv.(cla.ConvergenceReceiver))
		}
		if doSender {
			c.convergenceSenders = append(c.convergenceSenders, cqe.conv.(cla.ConvergenceSender))
		}
		c.convergenceMutex.Unlock()

		if doReceiver {
			c.reloadConvRecs <- struct{}{}
		}

		retry = false
		return
	}
}

// RegisterConvergence registeres a CLA on this Core. This could be a
// ConvergenceReceiver, ConvergenceSender or even both.
func (c *Core) RegisterConvergence(conv cla.Convergence) {
	cqe := newConvergenceQueueElement(conv)

	if retry := cqe.activate(c); !retry {
		c.convergenceMutex.Lock()
		c.convergenceQueue = append(c.convergenceQueue, cqe)
		c.convergenceMutex.Unlock()

		log.WithFields(log.Fields{
			"cla": conv,
		}).Debug("Failed to start CLA, it will be enqueued")
	}
}

// removeConvergenceSender removes a (known) ConvergenceSender. It should have
// been `Close()`ed before.
func (c *Core) removeConvergenceSender(sender cla.ConvergenceSender) {
	c.convergenceMutex.Lock()
	for i := len(c.convergenceSenders) - 1; i >= 0; i-- {
		if c.convergenceSenders[i] == sender {
			log.WithFields(log.Fields{
				"cla": sender,
			}).Info("Removing ConvergenceSender")

			c.convergenceSenders = append(
				c.convergenceSenders[:i], c.convergenceSenders[i+1:]...)
		}
	}
	c.convergenceMutex.Unlock()
}

// removeConvergenceReceiver removes a (known) ConvergenceSender. It should have
// been `Close()`ed before.
func (c *Core) removeConvergenceReceiver(rec cla.ConvergenceReceiver) {
	c.convergenceMutex.Lock()
	for i := len(c.convergenceReceivers) - 1; i >= 0; i-- {
		if c.convergenceReceivers[i] == rec {
			log.WithFields(log.Fields{
				"cla": rec,
			}).Info("Removing ConvergenceReceiver")

			c.convergenceReceivers = append(
				c.convergenceReceivers[:i], c.convergenceReceivers[i+1:]...)
		}
	}
	c.convergenceMutex.Unlock()
}

// RemoveConvergence removes a Convergence. It should have been
// `Close()`ed before.
func (c *Core) RemoveConvergence(conv cla.Convergence) {
	if _, ok := conv.(cla.ConvergenceReceiver); ok {
		c.removeConvergenceReceiver(conv.(cla.ConvergenceReceiver))
	}

	if _, ok := conv.(cla.ConvergenceSender); ok {
		c.removeConvergenceSender(conv.(cla.ConvergenceSender))
	}
}

// RestartConvergence stops and restarts a Convergence.
func (c *Core) RestartConvergence(conv cla.Convergence) {
	log.WithFields(log.Fields{
		"cla": conv,
	}).Info("Restarting Convergence")

	c.RemoveConvergence(conv)
	c.RegisterConvergence(conv)
}
