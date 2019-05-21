package core

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7/bundle"
	"github.com/dtn7/dtn7/cla"
)

// EpidemicRouting is an implementation of a RoutingAlgorithm and behaves in a
// flooding-based epidemic way.
type EpidemicRouting struct {
	c       *Core
	sentMap map[string][]bundle.EndpointID
}

// NewEpidemicRouting creates a new EpidemicRouting RoutingAlgorithm interacting
// with the given Core.
func NewEpidemicRouting(c *Core) EpidemicRouting {
	return EpidemicRouting{
		c:       c,
		sentMap: make(map[string][]bundle.EndpointID),
	}
}

// NotifyIncoming tells the EpidemicRouting about new bundles. In our case, the
// PreviousNodeBlock will be inspected.
func (er EpidemicRouting) NotifyIncoming(bp BundlePack) {
	// Check if we got a PreviousNodeBlock and extract its EndpointID
	var prevNode bundle.EndpointID
	if pnBlock, err := bp.Bundle.ExtensionBlock(bundle.PreviousNodeBlock); err == nil {
		prevNode = pnBlock.Data.(bundle.EndpointID)
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

	er.sentMap[bp.ID()] = append(sentEids, prevNode)
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

	for _, cs := range er.c.convergenceSenders {
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

	er.sentMap[bp.ID()] = sentEids

	log.WithFields(log.Fields{
		"bundle":              bp.ID(),
		"sent":                sentEids,
		"convergence-senders": css,
	}).Debug("EpidemicRouting selected Convergence Senders for an outbounding bundle")

	del = false
	return
}
