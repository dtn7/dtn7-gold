package core

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// L is the number of copies that are sprayed
const sprayl = 10

// SprayAndWait implements the vanilla Spray and Wait routing protocol
// In this case, the bundle originator distributes all L copies themselves
type SprayAndWait struct {
	c *Core
	// bundleData stores the metadata for each bundle
	bundleData map[string]sprayMetaData
}

type sprayMetaData struct {
	// sent is the list of nodes to which we have already relayed this bundle
	sent []bundle.EndpointID
	// remainingCopies is the number of copies we have to distribute before we enter wait-mode
	remainingCopies int32
}

// NewSprayAndWait creates new instance of SprayAndWait
func NewSprayAndWait(c *Core) SprayAndWait {
	return SprayAndWait{
		c:          c,
		bundleData: make(map[string]sprayMetaData),
	}
}

// NotifyIncoming tells the routing algorithm about new bundles.
// In this case, we simply check if we originated this bundle and set L if we did
// If we are not the originator, we don't further distribute the bundle
func (sw SprayAndWait) NotifyIncoming(bp BundlePack) {
	if sw.c.hasEndpoint(bp.Bundle.PrimaryBlock.SourceNode) {
		metadata := sprayMetaData{
			sent:            make([]bundle.EndpointID, 0),
			remainingCopies: sprayl,
		}
		sw.bundleData[bp.ID()] = metadata
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("SprayAndWait initialised new bundle from this host")
	} else {
		metadata := sprayMetaData{
			sent:            make([]bundle.EndpointID, 0),
			remainingCopies: 0,
		}
		sw.bundleData[bp.ID()] = metadata
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("SprayAndWait received bundle from foreign host")
	}
}

// SenderForBundle returns the Core's ConvergenceSenders.
func (sw SprayAndWait) SenderForBundle(bp BundlePack) (css []cla.ConvergenceSender, del bool) {
	metadata, _ := sw.bundleData[bp.ID()]
	// if there are no copies left, we just wait until we meet the recipient
	if !(metadata.remainingCopies > 0) {
		return nil, false
	}

	for _, cs := range sw.c.convergenceSenders {
		// if we ran out of copies, then don't send it to any further peers
		if !(metadata.remainingCopies > 0) {
			break
		}

		var skip = false
		for _, eid := range metadata.sent {
			if cs.GetPeerEndpointID() == eid {
				skip = true
				break
			}
		}

		if !skip {
			css = append(css, cs)
			metadata.sent = append(metadata.sent, cs.GetPeerEndpointID())
			metadata.remainingCopies = metadata.remainingCopies - 1
		}
	}

	sw.bundleData[bp.ID()] = metadata

	log.WithFields(log.Fields{
		"bundle":              bp.ID(),
		"convergence-senders": css,
	}).Debug("SprayAndWait selected Convergence Senders for an outgoing bundle")

	del = false
	return
}

// ReportFailure re-increments remaining copies if delivery was unsuccessful.
func (sw SprayAndWait) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {
	log.WithFields(log.Fields{
		"bundle":  bp.ID(),
		"bad_cla": sender,
	}).Debug("Transmission failure")

	metadata, _ := sw.bundleData[bp.ID()]
	metadata.remainingCopies = metadata.remainingCopies + 1

	for i := 0; i < len(metadata.sent); i++ {
		if metadata.sent[i] == sender.GetPeerEndpointID() {
			metadata.sent = append(metadata.sent[:i], metadata.sent[i+1:]...)
			break
		}
	}

	sw.bundleData[bp.ID()] = metadata
}
