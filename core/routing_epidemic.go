package core

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// EpidemicRouting is an implementation of a RoutingAlgorithm and behaves in a
// flooding-based epidemic way.
type EpidemicRouting struct {
	c       *Core
	sentMap map[bundle.BundleID][]bundle.EndpointID
	// Mutex for concurrent modification of data by multiple goroutines
	dataMutex sync.RWMutex
}

// NewEpidemicRouting creates a new EpidemicRouting RoutingAlgorithm interacting
// with the given Core.
func NewEpidemicRouting(c *Core) *EpidemicRouting {
	log.Debug("Initialised epidemic routing")

	er := EpidemicRouting{
		c:       c,
		sentMap: make(map[bundle.BundleID][]bundle.EndpointID),
	}

	err := c.cron.Register("epidemic_gc", er.GarbageCollect, time.Second*60)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Could not register EpidemicRouting gc-cron")
	}

	return &er
}

// GarbageCollect performs periodical cleanup of Bundle metadata
func (er *EpidemicRouting) GarbageCollect() {
	er.dataMutex.Lock()
	defer er.dataMutex.Unlock()

	for bundleId := range er.sentMap {
		if !er.c.store.KnowsBundle(bundleId) {
			delete(er.sentMap, bundleId)
		}
	}
}

// NotifyIncoming tells the EpidemicRouting about new bundles. In our case, the
// PreviousNodeBlock will be inspected.
func (er *EpidemicRouting) NotifyIncoming(bp BundlePack) {
	// Check if we got a PreviousNodeBlock and extract its EndpointID
	var prevNode bundle.EndpointID
	if pnBlock, err := bp.MustBundle().ExtensionBlock(bundle.ExtBlockTypePreviousNodeBlock); err == nil {
		prevNode = pnBlock.Value.(*bundle.PreviousNodeBlock).Endpoint()
	} else {
		return
	}

	er.dataMutex.RLock()
	sentEids, ok := er.sentMap[bp.Id]
	er.dataMutex.RUnlock()
	if !ok {
		sentEids = make([]bundle.EndpointID, 0)
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
	er.sentMap[bp.Id] = append(sentEids, prevNode)
	er.dataMutex.Unlock()
}

// SenderForBundle returns the Core's ConvergenceSenders.
func (er *EpidemicRouting) SenderForBundle(bp BundlePack) (css []cla.ConvergenceSender, del bool) {
	er.dataMutex.RLock()
	sentEids, ok := er.sentMap[bp.Id]
	er.dataMutex.RUnlock()
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
	er.sentMap[bp.Id] = sentEids
	er.dataMutex.Unlock()

	log.WithFields(log.Fields{
		"bundle":              bp.ID(),
		"sent":                sentEids,
		"convergence-senders": css,
	}).Debug("EpidemicRouting selected Convergence Senders for an outbounding bundle")

	del = false
	return
}

func (er *EpidemicRouting) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {
	er.dataMutex.RLock()
	sentEids, ok := er.sentMap[bp.Id]
	er.dataMutex.RUnlock()
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
	er.sentMap[bp.Id] = sentEids
	er.dataMutex.Unlock()
}

func (_ *EpidemicRouting) ReportPeerAppeared(_ cla.Convergence) {}

func (_ *EpidemicRouting) ReportPeerDisappeared(_ cla.Convergence) {}
