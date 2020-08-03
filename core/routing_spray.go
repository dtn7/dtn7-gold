// SPDX-FileCopyrightText: 2019 Markus Sommer
// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package core

import (
	"io"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/cboring"
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

type SprayConfig struct {
	// Multiplicity is the number of copies of a bundle which are sprayed
	Multiplicity uint64
}

// SprayAndWait implements the vanilla Spray and Wait routing protocol
// In this case, the bundle originator distributes all Multiplicity copies themselves
type SprayAndWait struct {
	c *Core
	// l is the number of copies of a bundle which are sprayed
	l uint64
	// bundleData stores the metadata for each bundle
	bundleData map[bundle.BundleID]sprayMetaData
	// Mutex for concurrent modification of data by multiple goroutines
	dataMutex sync.RWMutex
}

// sprayMetaData stores bundle-specific metadata
type sprayMetaData struct {
	// sent is the list of nodes to which we have already relayed this bundle
	sent []bundle.EndpointID
	// remainingCopies is the number of copies we have to distribute before we enter wait-mode
	remainingCopies uint64
}

// cleanupMetaData goes through stored metadata, determines if the corresponding bundle is still alive
// and deletes metadata for expired bundles
func cleanupMetaData(c *Core, metadata *map[bundle.BundleID]sprayMetaData) {
	for bundleId := range *metadata {
		if !c.store.KnowsBundle(bundleId) {
			delete(*metadata, bundleId)
		}
	}
}

// NewSprayAndWait creates new instance of SprayAndWait
func NewSprayAndWait(c *Core, config SprayConfig) *SprayAndWait {
	log.WithFields(log.Fields{
		"Multiplicity": config.Multiplicity,
	}).Debug("Initialised SprayAndWait")

	sprayAndWait := SprayAndWait{
		c:          c,
		l:          config.Multiplicity,
		bundleData: make(map[bundle.BundleID]sprayMetaData),
	}

	err := c.cron.Register("spray_and_wait_gc", sprayAndWait.GarbageCollect, time.Second*60)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Could not register SprayAndWait gc-cron")
	}

	return &sprayAndWait
}

// GarbageCollect performs periodical cleanup of Bundle metadata
func (sw *SprayAndWait) GarbageCollect() {
	sw.dataMutex.Lock()
	cleanupMetaData(sw.c, &sw.bundleData)
	sw.dataMutex.Unlock()
}

// NotifyIncoming tells the routing algorithm about new bundles.
// In this case, we simply check if we originated this bundle and set Multiplicity if we did
// If we are not the originator, we don't further distribute the bundle
func (sw *SprayAndWait) NotifyIncoming(bp BundlePack) {
	if sw.c.HasEndpoint(bp.MustBundle().PrimaryBlock.SourceNode) {
		metadata := sprayMetaData{
			sent:            make([]bundle.EndpointID, 0),
			remainingCopies: sw.l,
		}

		sw.dataMutex.Lock()
		sw.bundleData[bp.Id] = metadata
		sw.dataMutex.Unlock()

		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("SprayAndWait initialised new bundle from this host")
	} else {
		metadata := sprayMetaData{
			sent:            make([]bundle.EndpointID, 0),
			remainingCopies: 1,
		}

		// if the bundle has a PreviousNodeBlock, add it to the list of nodes which we know to have the bundle
		if pnBlock, err := bp.MustBundle().ExtensionBlock(bundle.ExtBlockTypePreviousNodeBlock); err == nil {
			metadata.sent = append(metadata.sent, pnBlock.Value.(*bundle.PreviousNodeBlock).Endpoint())
		}

		sw.dataMutex.Lock()
		sw.bundleData[bp.Id] = metadata
		sw.dataMutex.Unlock()

		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("SprayAndWait received bundle from foreign host")
	}
}

// DispatchingAllowed allows the processing of all packages.
func (_ *SprayAndWait) DispatchingAllowed(_ BundlePack) bool {
	return true
}

// SenderForBundle returns the Core's ConvergenceSenders.
// The bundle's originator will distribute Multiplicity copies amongst its peers
// Forwarders will only every deliver the bundle to its final destination
func (sw *SprayAndWait) SenderForBundle(bp BundlePack) (css []cla.ConvergenceSender, del bool) {
	sw.dataMutex.RLock()
	metadata, ok := sw.bundleData[bp.Id]
	sw.dataMutex.RUnlock()
	if !ok {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Warn("No metadata")
		return
	}
	// if there are no copies left, we just wait until we meet the recipient
	if metadata.remainingCopies < 2 {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("Not relaying bundle because there are no copies left")
		return nil, false
	}

	for _, cs := range sw.c.claManager.Sender() {
		// if we ran out of copies, then don't send it to any further peers
		if metadata.remainingCopies < 2 {
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

	sw.dataMutex.Lock()
	sw.bundleData[bp.Id] = metadata
	sw.dataMutex.Unlock()

	log.WithFields(log.Fields{
		"bundle":              bp.ID(),
		"convergence-senders": css,
		"remaining copies":    metadata.remainingCopies,
	}).Debug("SprayAndWait selected Convergence Senders for an outgoing bundle")

	del = false
	return
}

// ReportFailure re-increments remaining copies if delivery was unsuccessful.
func (sw *SprayAndWait) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {
	log.WithFields(log.Fields{
		"bundle":  bp.ID(),
		"bad_cla": sender,
	}).Debug("Transmission failure")

	sw.dataMutex.RLock()
	metadata, ok := sw.bundleData[bp.Id]
	sw.dataMutex.RUnlock()
	if !ok {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Warn("No metadata")
		return
	}

	metadata.remainingCopies = metadata.remainingCopies + 1

	for i := 0; i < len(metadata.sent); i++ {
		if metadata.sent[i] == sender.GetPeerEndpointID() {
			metadata.sent = append(metadata.sent[:i], metadata.sent[i+1:]...)
			break
		}
	}

	sw.dataMutex.Lock()
	sw.bundleData[bp.Id] = metadata
	sw.dataMutex.Unlock()
}

func (_ *SprayAndWait) ReportPeerAppeared(_ cla.Convergence) {}

func (_ *SprayAndWait) ReportPeerDisappeared(_ cla.Convergence) {}

// BinarySpray implements the binary Spray and Wait routing protocol
// In this case, each node hands over floor(copies/2) during the spray phase
type BinarySpray struct {
	c *Core
	// l is the number of copies of a bundle which are sprayed
	l uint64
	// bundleData stores the metadata for each bundle
	bundleData map[bundle.BundleID]sprayMetaData
	// Mutex for concurrent modification of data by multiple goroutines
	dataMutex sync.RWMutex
}

// NewBinarySpray creates new instance of BinarySpray
func NewBinarySpray(c *Core, config SprayConfig) *BinarySpray {
	log.WithFields(log.Fields{
		"Multiplicity": config.Multiplicity,
	}).Debug("Initialised BinarySpray")

	// register our custom metadata-block
	extensionBlockManager := bundle.GetExtensionBlockManager()
	if !extensionBlockManager.IsKnown(ExtBlockTypeBinarySprayBlock) {
		// since we already checked if the block type exists, this really shouldn't ever fail...
		_ = extensionBlockManager.Register(NewBinarySprayBlock(0))
	}

	binarySpray := BinarySpray{
		c:          c,
		l:          config.Multiplicity,
		bundleData: make(map[bundle.BundleID]sprayMetaData),
	}

	err := c.cron.Register("binary_spray_gc", binarySpray.GarbageCollect, time.Second*60)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Could not register BinarySpray gc-cron")
	}

	return &binarySpray
}

// GarbageCollect performs periodical cleanup of Bundle metadata
func (bs *BinarySpray) GarbageCollect() {
	bs.dataMutex.Lock()
	cleanupMetaData(bs.c, &bs.bundleData)
	bs.dataMutex.Unlock()
}

// NotifyIncoming tells the routing algorithm about new bundles.
// In this case, we check, whether we are the originator of this bundle
// If yes, then we initialise the remaining Copies to Multiplicity
// If not we attempt to ready the routing-metadata-block end get the remaining copies
func (bs *BinarySpray) NotifyIncoming(bp BundlePack) {
	if metadataBlock, err := bp.MustBundle().ExtensionBlock(ExtBlockTypeBinarySprayBlock); err == nil {
		binarySprayBlock := metadataBlock.Value.(*BinarySprayBlock)
		metadata := sprayMetaData{
			sent:            make([]bundle.EndpointID, 0),
			remainingCopies: binarySprayBlock.RemainingCopies(),
		}

		// if the bundle has a PreviousNodeBlock, add it to the list of nodes which we know to have the bundle
		if pnBlock, err := bp.MustBundle().ExtensionBlock(bundle.ExtBlockTypePreviousNodeBlock); err == nil {
			metadata.sent = append(metadata.sent, pnBlock.Value.(*bundle.PreviousNodeBlock).Endpoint())
		}

		bs.dataMutex.Lock()
		bs.bundleData[bp.Id] = metadata
		bs.dataMutex.Unlock()

		log.WithFields(log.Fields{
			"bundle":           bp.ID(),
			"remaining_copies": metadata.remainingCopies,
		}).Debug("SprayAndWait received bundle from foreign host")
	} else {
		metadata := sprayMetaData{
			sent:            make([]bundle.EndpointID, 0),
			remainingCopies: bs.l,
		}

		bs.dataMutex.Lock()
		bs.bundleData[bp.Id] = metadata
		bs.dataMutex.Unlock()

		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("SprayAndWait initialised new bundle from this host")
	}
}

// DispatchingAllowed allows the processing of all packages.
func (_ *BinarySpray) DispatchingAllowed(_ BundlePack) bool {
	return true
}

// SenderForBundle returns the Core's ConvergenceSenders.
// If a node has more than 1 copy left it will send floor(copies/2) to the peer
// and keep roof(copies/2) for itself
func (bs *BinarySpray) SenderForBundle(bp BundlePack) (css []cla.ConvergenceSender, del bool) {
	bs.dataMutex.RLock()
	metadata, ok := bs.bundleData[bp.Id]
	bs.dataMutex.RUnlock()
	if !ok {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Warn("No metadata")
		return
	}

	// if there are no copies left, we just wait until we meet the recipient
	if metadata.remainingCopies < 2 {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("Not relaying bundle because there are no copies left")
		return nil, false
	}

	for _, cs := range bs.c.claManager.Sender() {
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
			if metadataBlock, err := bp.MustBundle().ExtensionBlock(ExtBlockTypeBinarySprayBlock); err == nil {
				binarySprayBlock := metadataBlock.Value.(*BinarySprayBlock)
				binarySprayBlock.SetCopies(sendCopies)
			} else {
				// if it doesn't, then create one
				metadataBlock := NewBinarySprayBlock(sendCopies)
				bp.MustBundle().AddExtensionBlock(bundle.NewCanonicalBlock(0, 0, metadataBlock))
			}

			// we currently only send a bundle to a single peer at once
			break
		}
	}

	bs.dataMutex.Lock()
	bs.bundleData[bp.Id] = metadata
	bs.dataMutex.Unlock()

	log.WithFields(log.Fields{
		"bundle":              bp.ID(),
		"convergence-senders": css,
		"remaining copies":    metadata.remainingCopies,
	}).Debug("BinarySpray selected Convergence Sender for an outgoing bundle")

	del = false
	return
}

// ReportFailure resets remaining copies if delivery was unsuccessful.
func (bs *BinarySpray) ReportFailure(bp BundlePack, sender cla.ConvergenceSender) {
	log.WithFields(log.Fields{
		"bundle":  bp.ID(),
		"bad_cla": sender,
	}).Debug("Transmission failure")

	metadataBlock, err := bp.MustBundle().ExtensionBlock(ExtBlockTypeBinarySprayBlock)
	if err != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Warn("Bundle has not metadata Block")
		return
	}

	binarySprayBlock := metadataBlock.Value.(*BinarySprayBlock)

	bs.dataMutex.RLock()
	metadata, ok := bs.bundleData[bp.Id]
	bs.dataMutex.RUnlock()
	if !ok {
		log.WithFields(log.Fields{
			"bundle":  bp.ID(),
			"bad_cla": sender,
		}).Warn("No metadata")
		return
	}
	binarySprayBlock.SetCopies(metadata.remainingCopies + binarySprayBlock.RemainingCopies())

	for i := 0; i < len(metadata.sent); i++ {
		if metadata.sent[i] == sender.GetPeerEndpointID() {
			metadata.sent = append(metadata.sent[:i], metadata.sent[i+1:]...)
			break
		}
	}

	bs.dataMutex.Lock()
	bs.bundleData[bp.Id] = metadata
	bs.dataMutex.Unlock()
}

func (_ *BinarySpray) ReportPeerAppeared(_ cla.Convergence) {}

func (_ *BinarySpray) ReportPeerDisappeared(_ cla.Convergence) {}

const ExtBlockTypeBinarySprayBlock uint64 = 192

// BinarySprayBlock contains metadata to let the next forwarder know their remaining copies
type BinarySprayBlock uint64

func NewBinarySprayBlock(copies uint64) *BinarySprayBlock {
	newBlock := BinarySprayBlock(copies)
	return &newBlock
}

func (bsb *BinarySprayBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeBinarySprayBlock
}

func (bsb *BinarySprayBlock) CheckValid() error {
	return nil
}

func (bsb *BinarySprayBlock) MarshalCbor(w io.Writer) error {
	return cboring.WriteUInt(uint64(*bsb), w)
}

func (bsb *BinarySprayBlock) UnmarshalCbor(r io.Reader) error {
	if us, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		*bsb = BinarySprayBlock(us)
		return nil
	}
}

func (bsb *BinarySprayBlock) RemainingCopies() uint64 {
	return uint64(*bsb)
}

func (bsb *BinarySprayBlock) SetCopies(newValue uint64) {
	*bsb = BinarySprayBlock(newValue)
}
