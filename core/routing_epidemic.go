package core

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// EpidemicRouting is an implementation of a RoutingAlgorithm and behaves in a
// flooding-based epidemic way.
type EpidemicRouting struct {
	c       *Core
	sentMap map[string][]bundle.EndpointID
	// Mutex for concurrent modification of data by multiple goroutines
	dataMutex *sync.Mutex
}

// NewEpidemicRouting creates a new EpidemicRouting RoutingAlgorithm interacting
// with the given Core.
func NewEpidemicRouting(c *Core) EpidemicRouting {
	log.Debug("Initialised epidemic routing")

	er := EpidemicRouting{
		c:         c,
		sentMap:   make(map[string][]bundle.EndpointID),
		dataMutex: &sync.Mutex{},
	}

	err := c.cron.Register("epidemic_gc", er.GarbageCollect, time.Second*60)
	if err != nil {
		log.WithFields(log.Fields{
			"reason": err,
		}).Warn("Could not register EpidemicRouting gc-cron")
	}

	return er
}

// GarbageCollect performs periodical cleanup of Bundle metadata
func (er EpidemicRouting) GarbageCollect() {
	cleanedData := make(map[string][]bundle.EndpointID)
	for bundleId, data := range er.sentMap {
		if _, err := er.c.store.QueryId(bundleId); err == nil {
			cleanedData[bundleId] = data
		}
	}

	er.dataMutex.Lock()
	er.sentMap = cleanedData
	er.dataMutex.Unlock()
}

// NotifyIncoming tells the EpidemicRouting about new bundles. In our case, the
// PreviousNodeBlock will be inspected.
func (er EpidemicRouting) NotifyIncoming(bp BundlePack) {
	// Check if we got a PreviousNodeBlock and extract its EndpointID
	var prevNode bundle.EndpointID
	if pnBlock, err := bp.Bundle.ExtensionBlock(bundle.ExtBlockTypePreviousNodeBlock); err == nil {
		prevNode = pnBlock.Value.(*bundle.PreviousNodeBlock).Endpoint()
	} else {
		return
	}

	sentEids, ok := er.sentMap[bp.ID()]
	if !ok {
		sentEids = make([]bundle.EndpointID, 0, 0)
	}

	// Check if PreviousNodeBlock is already known
	for _, eids := range sentEids {
		if eids == prevNode {
			return
		}
	}

	log.WithFields(log.Fields{
		"bundle": bp.ID(),
		"eid":    prevNode,
	}).Debug("EpidemicRouting received an incomming bundle and checked its PreviousNodeBlock")

	er.dataMutex.Lock()
	er.sentMap[bp.ID()] = append(sentEids, prevNode)
	er.dataMutex.Unlock()
}

// SenderForBundle returns the Core's ConvergenceSenders.
func (er EpidemicRouting) SenderForBundle(bp BundlePack) (css []cla.ConvergenceSender, del bool) {
	sentEids, ok := er.sentMap[bp.ID()]
	if !ok {
		sentEids = make([]bundle.EndpointID, 0, 0)
	}

	log.WithFields(log.Fields{
		"bundle": bp.ID(),
		"sent":   sentEids,
	}).Debug("EpidemicRouting is processing outbounding bundle")

	for _, cs := range er.c.claManager.Sender() {
		var skip bool = false
		for _, eid := range sentEids {
			if cs.GetPeerEndpointID() == eid {
				skip = true
				break
			}
		}

		if !skip {
			css = append(css, cs)
			sentEids = append(sentEids, cs.GetPeerEndpointID())
		}
	}

	er.dataMutex.Lock()
	er.sentMap[bp.ID()] = sentEids
	er.dataMutex.Unlock()

	log.WithFields(log.Fields{
		"bundle":              bp.ID(),
		"sent":                sentEids,
		"convergence-senders": css,
	}).Debug("EpidemicRouting selected Convergence Senders for an outbounding bundle")

	del = false
	return
}

func (er EpidemicRouting) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {
	sentEids, ok := er.sentMap[bp.ID()]
	if !ok {
		return
	}

	log.WithFields(log.Fields{
		"bundle":  bp.ID(),
		"bad_cla": sender,
		"sent":    sentEids,
	}).Debug("EpidemicRouting failed to transmit to CLA")

	for i := 0; i < len(sentEids); i++ {
		if sentEids[i] == sender.GetPeerEndpointID() {
			sentEids = append(sentEids[:i], sentEids[i+1:]...)
			break
		}
	}

	er.dataMutex.Lock()
	er.sentMap[bp.ID()] = sentEids
	er.dataMutex.Unlock()
}

func (_ EpidemicRouting) ReportPeerAppeared(_ cla.Convergence) {}

func (_ EpidemicRouting) ReportPeerDisappeared(_ cla.Convergence) {}
