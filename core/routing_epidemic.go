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
// with the given Core. The second parameter indicates if a bundle should also
// be send back to its origin.
func NewEpidemicRouting(c *Core) EpidemicRouting {
	return EpidemicRouting{
		c:       c,
		sentMap: make(map[string][]bundle.EndpointID),
	}
}

// NotifyIncoming tells the EpidemicRouting new bundles. However,
// EpidemicRouting simply does not listen.
func (er EpidemicRouting) NotifyIncoming(_ BundlePack) {}

// SenderForBundle returns the Core's ConvergenceSenders. The ConvergenceSender
// for this BundlePack's receiver will be removed sendBack is false.
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
