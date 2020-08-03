// SPDX-FileCopyrightText: 2019 Markus Sommer
// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package core

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// EpidemicRouting is an implementation of a RoutingAlgorithm and behaves in a
// flooding-based epidemic way.
type EpidemicRouting struct {
	c *Core
}

// NewEpidemicRouting creates a new EpidemicRouting RoutingAlgorithm interacting
// with the given Core.
func NewEpidemicRouting(c *Core) *EpidemicRouting {
	log.Debug("Initialised epidemic routing")

	return &EpidemicRouting{c: c}
}

// NotifyIncoming tells the EpidemicRouting about new bundles. In our case, the
// PreviousNodeBlock will be inspected.
func (er *EpidemicRouting) NotifyIncoming(bp BundlePack) {
	bi, biErr := er.c.store.QueryId(bp.Id)
	if biErr != nil {
		log.WithFields(log.Fields{
			"error": biErr,
		}).Warn("Failed to proceed a non-stored Bundle")
		return
	}

	bndl := bp.MustBundle()

	if _, ok := bi.Properties["routing/epidemic/destination"]; !ok {
		bi.Properties["routing/epidemic/destination"] = bndl.PrimaryBlock.Destination
		if err := er.c.store.Update(bi); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Updating BundleItem failed")
		}
	}

	// Check if we got a PreviousNodeBlock and extract its EndpointID
	var prevNode bundle.EndpointID
	if pnBlock, err := bndl.ExtensionBlock(bundle.ExtBlockTypePreviousNodeBlock); err == nil {
		prevNode = pnBlock.Value.(*bundle.PreviousNodeBlock).Endpoint()
	} else {
		return
	}

	sentEids, ok := bi.Properties["routing/epidemic/sent"].([]bundle.EndpointID)
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

	bi.Properties["routing/epidemic/sent"] = append(sentEids, prevNode)
	if err := er.c.store.Update(bi); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Updating BundleItem failed")
	}
}

func (er *EpidemicRouting) clasForBundle(bp BundlePack, updateDb bool) (css []cla.ConvergenceSender, del bool) {
	bi, biErr := er.c.store.QueryId(bp.Id)
	if biErr != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"error":  biErr,
		}).Warn("Failed to proceed a non-stored Bundle")
		return nil, false
	}

	css, sentEids := filterCLAs(bi, er.c.claManager.Sender(), "epidemic")

	log.WithFields(log.Fields{
		"bundle": bp.ID(),
		"sent":   sentEids,
	}).Debug("EpidemicRouting is processing an outgoing bundle")

	if updateDb {
		bi.Properties["routing/epidemic/sent"] = sentEids
		if err := er.c.store.Update(bi); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Updating BundleItem failed")
		}
	}

	log.WithFields(log.Fields{
		"bundle":              bp.ID(),
		"sent":                sentEids,
		"convergence-senders": css,
	}).Debug("EpidemicRouting selected Convergence Senders for an outbounding bundle")

	del = false
	return
}

// DispatchingAllowed only allows dispatching, iff the bundle is addressed to
// this Node or if any known CLA without having received this bundle exists.
func (er *EpidemicRouting) DispatchingAllowed(bp BundlePack) bool {
	bi, biErr := er.c.store.QueryId(bp.Id)
	if biErr != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"error":  biErr,
		}).Warn("Failed to proceed a non-stored Bundle")

		return true
	} else if dst, ok := bi.Properties["routing/epidemic/destination"]; ok {
		if er.c.HasEndpoint(dst.(bundle.EndpointID)) {
			return true
		}
	}

	css, _ := er.clasForBundle(bp, false)

	if len(css) == 0 {
		bi.Pending = true
		if err := er.c.store.Update(bi); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Updating BundleItem failed")
		}
	}

	return len(css) > 0
}

// SenderForBundle returns the Core's ConvergenceSenders.
func (er *EpidemicRouting) SenderForBundle(bp BundlePack) (css []cla.ConvergenceSender, del bool) {
	return er.clasForBundle(bp, true)
}

func (er *EpidemicRouting) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {
	bi, biErr := er.c.store.QueryId(bp.Id)
	if biErr != nil {
		log.WithFields(log.Fields{
			"error": biErr,
		}).Warn("Failed to proceed a non-stored Bundle")
		return
	}

	sentEids, ok := bi.Properties["routing/epidemic/sent"].([]bundle.EndpointID)
	if !ok {
		sentEids = make([]bundle.EndpointID, 0)
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

	bi.Properties["routing/epidemic/sent"] = sentEids
	if err := er.c.store.Update(bi); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Updating BundleItem failed")
	}
}

func (_ *EpidemicRouting) ReportPeerAppeared(_ cla.Convergence) {}

func (_ *EpidemicRouting) ReportPeerDisappeared(_ cla.Convergence) {}

func (_ *EpidemicRouting) String() string {
	return "epidemic"
}
