package cla

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

// convergenceElem is a wrapper around a Convergence to assign a status,
// supervised by a Manager.
type convergenceElem struct {
	// conv is the wrapped Convergence
	conv Convergence

	// mutex protects critical parts
	mutex sync.Mutex

	// convChnl is the Manager's inChnl.
	convChnl chan ConvergenceStatus

	// ttl is used both for determining the activity and for counting-off.
	// A negative ttl implies an active convergenceElem.
	ttl int

	// stop{Syn,Ack} are used to supervise closing this convergenceElem, see deactivate()
	stopSyn chan struct{}
	stopAck chan struct{}
}

// newConvergenceElement creates a new convergenceElem for a Convergence with
// an initial ttl value.
func newConvergenceElement(conv Convergence, convChnl chan ConvergenceStatus, ttl int) *convergenceElem {
	return &convergenceElem{
		conv:     conv,
		convChnl: convChnl,
		ttl:      ttl,
	}
}

// isReceiver returns if this convergenceElem is wraped around a ConvergenceReceiver.
func (ce *convergenceElem) isReceiver() bool {
	_, ok := (ce.conv).(ConvergenceReceiver)
	return ok
}

// isSender returns if this convergenceElem is wraped around a ConvergenceSender.
func (ce *convergenceElem) isSender() bool {
	_, ok := (ce.conv).(ConvergenceSender)
	return ok
}

// isActive return if this convergenceElem is wraped around an active Convergence.
func (ce *convergenceElem) isActive() bool {
	return ce.ttl < 0
}

// handler supervises both stopping and ConvergenceStatus forwarding to the Manager.
func (ce *convergenceElem) handler() {
	for {
		select {
		case <-ce.stopSyn:
			log.WithFields(log.Fields{
				"cla": ce.conv,
			}).Debug("Closing CLA's handler")

			close(ce.stopAck)
			return

		case cs := <-ce.conv.Channel():
			log.WithFields(log.Fields{
				"cla":    ce.conv,
				"status": cs.String(),
			}).Debug("Forwarding ConvergenceStatus to Manager")

			ce.convChnl <- cs
		}
	}
}

// activate tries to start this convergenceElem. Both a success message and an
// indicator for a new attempt are returned.
func (ce *convergenceElem) activate() (successful, retry bool) {
	if ce.isActive() {
		return
	}

	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	if ce.ttl == 0 && !ce.conv.IsPermanent() {
		log.WithFields(log.Fields{
			"cla":   ce.conv,
			"error": "TTL expired",
		}).Info("Failed to start CLA")

		return false, false
	}

	claErr, claRetry := ce.conv.Start()
	if claErr == nil {
		log.WithFields(log.Fields{
			"cla": ce.conv,
		}).Info("Started CLA")

		ce.ttl = -1

		ce.stopSyn = make(chan struct{})
		ce.stopAck = make(chan struct{})
		go ce.handler()

		return true, false
	} else {
		log.WithFields(log.Fields{
			"cla":       ce.conv,
			"permanent": ce.conv.IsPermanent(),
			"ttl":       ce.ttl,
			"retry":     claRetry,
			"error":     claErr,
		}).Info("Failed to start CLA")

		if claRetry {
			ce.ttl -= 1
		} else {
			ce.ttl = 0
		}

		return false, claRetry
	}
}

// deactivate marks this convergenceElem as deactivated. Both a new ttl as well
// as whether Stop should be executed can be specified.
func (ce *convergenceElem) deactivate(ttl int, closeCall bool) {
	if !ce.isActive() {
		return
	}

	log.WithFields(log.Fields{
		"cla":   ce.conv,
		"close": closeCall,
	}).Info("Deactivating CLA")

	if closeCall {
		ce.conv.Close()
	}

	close(ce.stopSyn)
	<-ce.stopAck

	ce.ttl = ttl
}
