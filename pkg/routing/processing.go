// SPDX-FileCopyrightText: 2019, 2020, 2021 Alvar Penning
// SPDX-FileCopyrightText: 2019, 2022 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
)

// SendBundle transmits an outbounding bundle.
func (c *Core) SendBundle(bndl *bpv7.Bundle) {
	if c.signPriv != nil && bndl.IsAdministrativeRecord() {
		c.sendBundleAttachSignature(bndl)
	}
	bp := NewBundleDescriptorFromBundle(*bndl, c.store)

	c.routing.NotifyNewBundle(bp)
	c.transmit(bp)
}

// sendBundleAttachSignature attaches a SignatureBlock to outgoing Administrative Records, if configured.
func (c *Core) sendBundleAttachSignature(bndl *bpv7.Bundle) {
	if c.signPriv == nil || !bndl.IsAdministrativeRecord() {
		return
	}

	sb, sbErr := bpv7.NewSignatureBlock(*bndl, c.signPriv)
	if sbErr != nil {
		log.WithField("bundle", bndl.ID()).WithError(sbErr).Error("Creating signature erred, proceeding without")
		return
	}

	cb := bpv7.NewCanonicalBlock(0, bpv7.ReplicateBlock|bpv7.DeleteBundle, sb)
	cb.SetCRCType(bpv7.CRC32)

	if err := bndl.AddExtensionBlock(cb); err != nil {
		log.WithFields(log.Fields{
			"bundle": bndl.ID().String(),
			"error":  err,
		}).Error("Error attaching signature block")
	}

	log.WithField("bundle", bndl.ID()).Info("Attached signature to outgoing bundle")
}

// transmit starts the transmission of an outgoing bundle pack.
// Therefore, the source's endpoint ID must be dtn:none or a member of this node.
func (c *Core) transmit(bp BundleDescriptor) {
	log.WithField("bundle", bp.ID().String()).Info("Transmission of bundle requested")

	c.idKeeper.update(&bp)

	bp.AddConstraint(DispatchPending)
	_ = bp.Sync()

	src := bp.MustBundle().PrimaryBlock.SourceNode
	if src != bpv7.DtnNone() && !c.HasEndpoint(src) {
		log.WithFields(log.Fields{
			"bundle": bp.ID().String(),
			"source": src,
		}).Info("Bundle's source is neither dtn:none nor an endpoint of this node")

		c.bundleDeletion(bp, bpv7.NoInformation)
		return
	}

	c.dispatching(bp)
}

// receive handles received/incoming bundles.
func (c *Core) receive(bp BundleDescriptor) {
	log.WithField("bundle", bp.ID().String()).Debug("Received new bundle")

	if len(bp.Constraints) > 0 {
		log.WithField("bundle", bp.ID().String()).Debug("Received bundle's ID is already known.")

		// bundleDeletion is _not_ called because this would delete the already
		// stored BundleDescriptor.
		return
	}

	log.WithField("bundle", bp.ID().String()).Info("Processing newly received bundle")

	bp.AddConstraint(DispatchPending)
	_ = bp.Sync()

	if bp.MustBundle().PrimaryBlock.BundleControlFlags.Has(bpv7.StatusRequestReception) {
		c.SendStatusReport(bp, bpv7.ReceivedBundle, bpv7.NoInformation)
	}

	for i := len(bp.MustBundle().CanonicalBlocks) - 1; i >= 0; i-- {
		var cb = &bp.MustBundle().CanonicalBlocks[i]

		if bpv7.GetExtensionBlockManager().IsKnown(cb.TypeCode()) {
			continue
		}

		log.WithFields(log.Fields{
			"bundle": bp.ID().String(),
			"number": i,
			"type":   cb.TypeCode(),
		}).Warn("Bundle's canonical block is unknown")

		if cb.BlockControlFlags.Has(bpv7.StatusReportBlock) {
			log.WithFields(log.Fields{
				"bundle": bp.ID().String(),
				"number": i,
				"type":   cb.TypeCode(),
			}).Info("Bundle's unknown canonical block requested reporting")

			c.SendStatusReport(bp, bpv7.ReceivedBundle, bpv7.BlockUnsupported)
		}

		if cb.BlockControlFlags.Has(bpv7.DeleteBundle) {
			log.WithFields(log.Fields{
				"bundle": bp.ID().String(),
				"number": i,
				"type":   cb.TypeCode(),
			}).Info("Bundle's unknown canonical block requested bundle deletion")

			c.bundleDeletion(bp, bpv7.BlockUnsupported)
			return
		}

		if cb.BlockControlFlags.Has(bpv7.RemoveBlock) {
			log.WithFields(log.Fields{
				"bundle": bp.ID().String(),
				"number": i,
				"type":   cb.TypeCode(),
			}).Info("Bundle's unknown canonical block requested to be removed")

			bp.MustBundle().CanonicalBlocks = append(
				bp.MustBundle().CanonicalBlocks[:i], bp.MustBundle().CanonicalBlocks[i+1:]...)
		}
	}

	c.routing.NotifyNewBundle(bp)

	c.dispatching(bp)
}

// dispatching handles the dispatching of received bundles.
func (c *Core) dispatching(bp BundleDescriptor) {
	log.WithField("bundle", bp.ID().String()).Info("Dispatching bundle")

	if !c.routing.DispatchingAllowed(bp) {
		log.WithFields(log.Fields{
			"bundle":  bp.ID().String(),
			"routing": c.routing,
		}).Info("Routing Algorithm has not allowed dispatching of bundle")
		return
	}

	bndl, err := bp.Bundle()
	if err != nil {
		log.WithFields(log.Fields{
			"bundleID": bp.ID().String(),
			"error":    err,
		}).Error("Error retrieving bundle")
		return
	}

	if c.HasEndpoint(bndl.PrimaryBlock.Destination) {
		c.localDelivery(bp)
	} else {
		c.forward(bp)
	}
}

// forward forwards a bundle pack's bundle to another node.
func (c *Core) forward(bp BundleDescriptor) {
	log.WithField("bundle", bp.ID().String()).Printf("Bundle will be forwarded")

	bp.AddConstraint(ForwardPending)
	bp.RemoveConstraint(DispatchPending)
	_ = bp.Sync()

	if hcBlock, err := bp.MustBundle().ExtensionBlock(bpv7.ExtBlockTypeHopCountBlock); err == nil {
		hc := hcBlock.Value.(*bpv7.HopCountBlock)
		hc.Increment()
		hcBlock.Value = hc

		log.WithFields(log.Fields{
			"bundle":    bp.ID().String(),
			"hop_count": hc,
		}).Debug("Bundle contains hop count block")

		if exceeded := hc.IsExceeded(); exceeded {
			log.WithFields(log.Fields{
				"bundle":    bp.ID().String(),
				"hop_count": hc,
			}).Info("Bundle hop count exceeded")

			c.bundleDeletion(bp, bpv7.HopLimitExceeded)
			return
		}
	}

	if bp.MustBundle().IsLifetimeExceeded() {
		log.WithFields(log.Fields{
			"bundle":        bp.ID().String(),
			"primary_block": bp.MustBundle().PrimaryBlock,
		}).Warn("Bundle lifetime exceeded")

		c.bundleDeletion(bp, bpv7.LifetimeExpired)
		return
	}

	if age, err := bp.UpdateBundleAge(); err == nil {
		if age >= bp.MustBundle().PrimaryBlock.Lifetime {
			log.WithField("bunde", bp.ID().String()).Warn("Bundle lifetime expired")

			c.bundleDeletion(bp, bpv7.LifetimeExpired)
			return
		}
	}

	if pnBlock, err := bp.MustBundle().ExtensionBlock(bpv7.ExtBlockTypePreviousNodeBlock); err == nil {
		// Replace the PreviousNodeBlock
		prevEid := pnBlock.Value.(*bpv7.PreviousNodeBlock).Endpoint()
		pnBlock.Value = bpv7.NewPreviousNodeBlock(c.NodeId)

		log.WithFields(log.Fields{
			"bundle":  bp.ID().String(),
			"old_eid": prevEid,
			"new_eid": c.NodeId,
		}).Debug("Previous Node Block updated")
	} else {
		// Append a new PreviousNodeBlock
		if err := bp.MustBundle().AddExtensionBlock(bpv7.NewCanonicalBlock(
			0, 0, bpv7.NewPreviousNodeBlock(c.NodeId))); err != nil {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
				"error":  err,
			}).Error("Error attaching PreviousNodeBlock")
		}
	}

	var nodes []cla.ConvergenceSender
	var deleteAfterwards = true

	// Try a direct delivery or consult the Algorithm otherwise.
	nodes = c.senderForDestination(bp.MustBundle().PrimaryBlock.Destination)
	if nodes == nil {
		nodes, deleteAfterwards = c.routing.SenderForBundle(bp)
	}

	var bundleSent = false

	var wg sync.WaitGroup
	var once sync.Once

	wg.Add(len(nodes))

	for _, node := range nodes {
		go func(node cla.ConvergenceSender) {
			log.WithFields(log.Fields{
				"bundle": bp.ID().String(),
				"cla":    node,
			}).Info("Sending bundle to a CLA (ConvergenceSender)")

			if err := node.Send(*bp.MustBundle()); err != nil {
				log.WithFields(log.Fields{
					"bundle": bp.ID().String(),
					"cla":    node,
					"error":  err,
				}).Warn("Sending bundle failed")

				c.routing.ReportFailure(bp, node)
			} else {
				log.WithFields(log.Fields{
					"bundle": bp.ID().String(),
					"cla":    node,
				}).Printf("Sending bundle succeeded")

				once.Do(func() { bundleSent = true })
			}

			wg.Done()
		}(node)
	}

	wg.Wait()

	if hcBlock, err := bp.MustBundle().ExtensionBlock(bpv7.ExtBlockTypeHopCountBlock); err == nil {
		hc := hcBlock.Value.(*bpv7.HopCountBlock)
		hc.Decrement()
		hcBlock.Value = hc

		log.WithFields(log.Fields{
			"bundle":    bp.ID().String(),
			"hop_count": hc,
		}).Debug("Reset bundle hop count")
	}

	if bundleSent {
		if bp.MustBundle().PrimaryBlock.BundleControlFlags.Has(bpv7.StatusRequestForward) {
			c.SendStatusReport(bp, bpv7.ForwardedBundle, bpv7.NoInformation)
		}

		if deleteAfterwards {
			bp.PurgeConstraints()
			_ = bp.Sync()
		} else if c.InspectAllBundles && bp.MustBundle().IsAdministrativeRecord() {
			c.bundleContraindicated(bp)
			c.checkAdministrativeRecord(bp)
		} else {
			c.bundleContraindicated(bp)
		}
	} else {
		log.WithField("bundle", bp.ID().String()).Info("Failed to forward bundle to any CLA")
		c.bundleContraindicated(bp)
	}
}

// checkAdministrativeRecord checks administrative records. If this method
// returns false, an error occured.
func (c *Core) checkAdministrativeRecord(bp BundleDescriptor) bool {
	if !bp.MustBundle().IsAdministrativeRecord() {
		log.WithField("bundle", bp.ID().String()).Debug("Bundle does not contain an administrative record")
		return false
	}

	canonicalAr, err := bp.MustBundle().PayloadBlock()
	if err != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID().String(),
			"error":  err,
		}).Warn("Bundle with an administrative record flag missing payload block")

		return false
	}

	payload := canonicalAr.Value.(*bpv7.PayloadBlock).Data()
	ar, err := bpv7.NewAdministrativeRecordFromCbor(payload)
	if err != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID().String(),
			"error":  err,
		}).Warn("Bundle with an administrative record could not be parsed")

		return false
	}

	log.WithFields(log.Fields{
		"bundle":    bp.ID().String(),
		"admin_rec": ar,
	}).Info("Received bundle with administrative record")

	// Currently there are only status reports. This must be changed if more
	// types of administrative records are introduced.
	c.inspectStatusReport(bp, ar)

	return true
}

func (c *Core) inspectStatusReport(bp BundleDescriptor, ar bpv7.AdministrativeRecord) {
	if ar.RecordTypeCode() != bpv7.AdminRecordTypeStatusReport {
		log.WithFields(log.Fields{
			"bundle":    bp.ID().String(),
			"type_code": ar.RecordTypeCode(),
		}).Warn("Administrative record is not a status report")

		return
	}

	var status = *ar.(*bpv7.StatusReport)
	var sips = status.StatusInformations()

	if len(sips) == 0 {
		log.WithFields(log.Fields{
			"bundle":    bp.ID().String(),
			"admin_rec": ar,
		}).Warn("Administrative record contains no status information")
		return
	}

	var bpStore, err = c.store.QueryId(status.RefBundle)
	if err != nil {
		log.WithFields(log.Fields{
			"bundle":     bp.ID().String(),
			"status_rep": status,
		}).Warn("Status Report's bundle is unknown")
		return
	}

	log.WithFields(log.Fields{
		"bundle":        bp.ID().String(),
		"status_rep":    status,
		"status_bundle": bpStore.Id,
	}).Debug("Status Report's referenced bundle was loaded")

	for _, sip := range sips {
		log.WithFields(log.Fields{
			"bundle":        bp.ID().String(),
			"status_rep":    status,
			"status_bundle": bpStore.Id,
			"information":   sip,
		}).Info("Parsing status report")

		switch sip {
		case bpv7.ReceivedBundle, bpv7.ForwardedBundle, bpv7.DeletedBundle:
			// Nothing to do

		case bpv7.DeliveredBundle:
			logger := log.WithFields(log.Fields{
				"bundle":        bp.ID().String(),
				"status_rep":    status,
				"status_bundle": bpStore.Id,
			})

			if err := c.store.Delete(bpStore.BId); err != nil {
				logger.WithError(err).Warn("Failed to delete delivered bundle")
			} else {
				logger.Info("Status report indicates delivered bundle, deleting bundle")
			}

		default:
			log.WithFields(log.Fields{
				"bundle":        bp.ID().String(),
				"status_rep":    status,
				"status_bundle": bpStore.Id,
				"information":   int(sip),
			}).Warn("Status report has an unknown status information code")
		}
	}
}

func (c *Core) localDelivery(bp BundleDescriptor) {
	// TODO: check fragmentation

	log.WithField("bundle", bp.ID().String()).Info("Received bundle for local delivery")

	if bp.MustBundle().IsAdministrativeRecord() {
		if !c.checkAdministrativeRecord(bp) {
			c.bundleDeletion(bp, bpv7.NoInformation)
			return
		}
	}

	bp.AddConstraint(LocalEndpoint)
	_ = bp.Sync()

	if err := c.agentManager.Deliver(bp); err != nil {
		log.WithField("bundle", bp.ID().String()).WithError(err).Warn("Delivering local bundle erred")
	}

	if bp.MustBundle().PrimaryBlock.BundleControlFlags.Has(bpv7.StatusRequestDelivery) {
		c.SendStatusReport(bp, bpv7.DeliveredBundle, bpv7.NoInformation)
	}

	bp.PurgeConstraints()
	_ = bp.Sync()
}

func (c *Core) bundleContraindicated(bp BundleDescriptor) {
	log.WithField("bundle", bp.ID().String()).Info("Bundle was marked for contraindication")

	bp.AddConstraint(Contraindicated)
	_ = bp.Sync()
}

func (c *Core) bundleDeletion(bp BundleDescriptor, reason bpv7.StatusReportReason) {
	if bp.MustBundle().PrimaryBlock.BundleControlFlags.Has(bpv7.StatusRequestDeletion) {
		c.SendStatusReport(bp, bpv7.DeletedBundle, reason)
	}

	bp.PurgeConstraints()
	_ = bp.Sync()

	log.WithField("bundle", bp.ID().String()).Info("Bundle was marked for deletion")
}
