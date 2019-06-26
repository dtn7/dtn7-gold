package core

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// sprayl is the number of copies that are sprayed
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
	remainingCopies uint64
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
			remainingCopies: 1,
		}

		// if the bundle has a PreviousNodeBlock, add it to the list of nodes which we know to have the bundle
		if pnBlock, err := bp.Bundle.ExtensionBlock(bundle.PreviousNodeBlock); err == nil {
			metadata.sent = append(metadata.sent, pnBlock.Data.(bundle.EndpointID))
		}

		sw.bundleData[bp.ID()] = metadata
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("SprayAndWait received bundle from foreign host")
	}
}

// SenderForBundle returns the Core's ConvergenceSenders.
// The bundle's originator will distribute L copies amongst its peers
// Forwarders will only every deliver the bundle to its final destination
func (sw SprayAndWait) SenderForBundle(bp BundlePack) (css []cla.ConvergenceSender, del bool) {
	metadata, _ := sw.bundleData[bp.ID()]
	// if there are no copies left, we just wait until we meet the recipient
	if !(metadata.remainingCopies > 1) {
		return nil, false
	}

	for _, cs := range sw.c.convergenceSenders {
		// if we ran out of copies, then don't send it to any further peers
		if !(metadata.remainingCopies > 1) {
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

// BinarySpray implements the binary Spray and Wait routing protocol
// In this case, each node hands over floor(copies/2) during the spray phase
type BinarySpray struct {
	c *Core
	// bundleData stores the metadata for each bundle
	bundleData map[string]sprayMetaData
}

// NewBinarySpray creates new instance of BinarySpray
func NewBinarySpray(c *Core) BinarySpray {
	return BinarySpray{
		c:          c,
		bundleData: make(map[string]sprayMetaData),
	}
}

// NotifyIncoming tells the routing algorithm about new bundles.
// In this case, we check, whether we are the originator of this bundle
// If yes, then we initialise the remaining Copies to L
// If not we attempt to ready the routing-metadata-block end get the remaining copies
func (bs BinarySpray) NotifyIncoming(bp BundlePack) {
	if metadatBlock, err := bp.Bundle.ExtensionBlock(bundle.BinarySprayBlock); err == nil {
		metadata := sprayMetaData{
			sent:            make([]bundle.EndpointID, 0),
			remainingCopies: metadatBlock.Data.(uint64),
		}

		// if the bundle has a PreviousNodeBlock, add it to the list of nodes which we know to have the bundle
		if pnBlock, err := bp.Bundle.ExtensionBlock(bundle.PreviousNodeBlock); err == nil {
			metadata.sent = append(metadata.sent, pnBlock.Data.(bundle.EndpointID))
		}

		bs.bundleData[bp.ID()] = metadata
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("SprayAndWait received bundle from foreign host")
	} else {
		metadata := sprayMetaData{
			sent:            make([]bundle.EndpointID, 0),
			remainingCopies: sprayl,
		}
		bs.bundleData[bp.ID()] = metadata
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("SprayAndWait initialised new bundle from this host")
	}
}

// SenderForBundle returns the Core's ConvergenceSenders.
// If a node has more than 1 copy left it will send floor(copies/2) to the peer
// and keep roof(copies/2) for itself
func (bs BinarySpray) SenderForBundle(bp BundlePack) (css []cla.ConvergenceSender, del bool) {
	metadata, _ := bs.bundleData[bp.ID()]
	// if there are no copies left, we just wait until we meet the recipient
	if !(metadata.remainingCopies > 1) {
		return nil, false
	}

	for _, cs := range bs.c.convergenceSenders {
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

			// we send half our remaining copies
			sendCopies := metadata.remainingCopies / 2
			metadata.remainingCopies = metadata.remainingCopies - sendCopies

			// if the bundle already has a metadata-block
			if metadataBlock, err := bp.Bundle.ExtensionBlock(bundle.BinarySprayBlock); err == nil {
				metadataBlock.Data = sendCopies
			} else {
				// if it doesn't, then create one
				NewMetadataBlock := bundle.NewCanonicalBlock(bundle.BinarySprayBlock, 0, 0, sendCopies)
				bp.Bundle.AddExtensionBlock(NewMetadataBlock)
			}

			// we currently only send a bundle to a single peer at once
			break
		}
	}

	bs.bundleData[bp.ID()] = metadata

	log.WithFields(log.Fields{
		"bundle":              bp.ID(),
		"convergence-senders": css,
	}).Debug("BinarySpray selected Convergence Sender for an outgoing bundle")

	del = false
	return
}

// ReportFailure resets remaining copies if delivery was unsuccessful.
func (bs BinarySpray) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {
	log.WithFields(log.Fields{
		"bundle":  bp.ID(),
		"bad_cla": sender,
	}).Debug("Transmission failure")

	metadataBlock, _ := bp.Bundle.ExtensionBlock(bundle.BinarySprayBlock)

	metadata, _ := bs.bundleData[bp.ID()]
	metadata.remainingCopies = metadata.remainingCopies + metadataBlock.Data.(uint64)

	for i := 0; i < len(metadata.sent); i++ {
		if metadata.sent[i] == sender.GetPeerEndpointID() {
			metadata.sent = append(metadata.sent[:i], metadata.sent[i+1:]...)
			break
		}
	}

	bs.bundleData[bp.ID()] = metadata
}
