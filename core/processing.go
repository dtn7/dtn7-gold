package core

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/bundle/arecord"
	"github.com/dtn7/dtn7-go/cla"
)

// SendBundle transmits an outbounding bundle.
func (c *Core) SendBundle(bndl *bundle.Bundle) {
	bp := NewBundlePackFromBundle(*bndl, c.store)

	c.routing.NotifyIncoming(bp)
	c.transmit(bp)
}

// transmit starts the transmission of an outbounding bundle pack. Therefore
// the source's endpoint ID must be dtn:none or a member of this node.
func (c *Core) transmit(bp BundlePack) {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Transmission of bundle requested")

	c.idKeeper.update(bp.MustBundle())

	bp.AddConstraint(DispatchPending)
	bp.Sync()

	src := bp.MustBundle().PrimaryBlock.SourceNode
	if src != bundle.DtnNone() && !c.HasEndpoint(src) {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"source": src,
		}).Info("Bundle's source is neither dtn:none nor an endpoint of this node")

		c.bundleDeletion(bp, arecord.NoInformation)
		return
	}

	c.dispatching(bp)
}

// receive handles received/incoming bundles.
func (c *Core) receive(bp BundlePack) {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Debug("Received new bundle")

	/*
		// TODO
		if c.store.KnowsBundle(bp.Id) {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
			}).Debug("Received bundle's ID is already known.")

			// bundleDeletion is _not_ called because this would delete the already
			// stored BundlePack.
			return
		}
	*/

	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Processing new received bundle")

	bp.AddConstraint(DispatchPending)
	bp.Sync()

	if bp.MustBundle().PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestReception) {
		c.SendStatusReport(bp, arecord.ReceivedBundle, arecord.NoInformation)
	}

	for i := len(bp.MustBundle().CanonicalBlocks) - 1; i >= 0; i-- {
		var cb = &bp.MustBundle().CanonicalBlocks[i]

		if bundle.GetExtensionBlockManager().IsKnown(cb.BlockTypeCode()) {
			continue
		}

		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"number": i,
			"type":   cb.BlockTypeCode(),
		}).Warn("Bundle's canonical block is unknown")

		if cb.BlockControlFlags.Has(bundle.StatusReportBlock) {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
				"number": i,
				"type":   cb.BlockTypeCode(),
			}).Info("Bundle's unknown canonical block requested reporting")

			c.SendStatusReport(bp, arecord.ReceivedBundle, arecord.BlockUnintelligible)
		}

		if cb.BlockControlFlags.Has(bundle.DeleteBundle) {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
				"number": i,
				"type":   cb.BlockTypeCode(),
			}).Info("Bundle's unknown canonical block requested bundle deletion")

			c.bundleDeletion(bp, arecord.BlockUnintelligible)
			return
		}

		if cb.BlockControlFlags.Has(bundle.RemoveBlock) {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
				"number": i,
				"type":   cb.BlockTypeCode(),
			}).Info("Bundle's unknown canonical block requested to be removed")

			bp.MustBundle().CanonicalBlocks = append(
				bp.MustBundle().CanonicalBlocks[:i], bp.MustBundle().CanonicalBlocks[i+1:]...)
		}
	}

	c.routing.NotifyIncoming(bp)

	c.dispatching(bp)
}

// dispatching handles the dispatching of received bundles.
func (c *Core) dispatching(bp BundlePack) {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Dispatching bundle")

	if c.HasEndpoint(bp.MustBundle().PrimaryBlock.Destination) {
		c.localDelivery(bp)
	} else {
		c.forward(bp)
	}
}

// forward forwards a bundle pack's bundle to another node.
func (c *Core) forward(bp BundlePack) {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Printf("Bundle will be forwarded")

	bp.AddConstraint(ForwardPending)
	bp.RemoveConstraint(DispatchPending)
	bp.Sync()

	if hcBlock, err := bp.MustBundle().ExtensionBlock(bundle.ExtBlockTypeHopCountBlock); err == nil {
		hc := hcBlock.Value.(*bundle.HopCountBlock)
		hc.Increment()
		hcBlock.Value = hc

		log.WithFields(log.Fields{
			"bundle":    bp.ID(),
			"hop_count": hc,
		}).Debug("Bundle contains an hop count block")

		if exceeded := hc.IsExceeded(); exceeded {
			log.WithFields(log.Fields{
				"bundle":    bp.ID(),
				"hop_count": hc,
			}).Info("Bundle contains an exceeded hop count block")

			c.bundleDeletion(bp, arecord.HopLimitExceeded)
			return
		}
	}

	if bp.MustBundle().PrimaryBlock.IsLifetimeExceeded() {
		log.WithFields(log.Fields{
			"bundle":        bp.ID(),
			"primary_block": bp.MustBundle().PrimaryBlock,
		}).Warn("Bundle's primary block's lifetime is exceeded")

		c.bundleDeletion(bp, arecord.LifetimeExpired)
		return
	}

	if age, err := bp.UpdateBundleAge(); err == nil {
		if age >= bp.MustBundle().PrimaryBlock.Lifetime {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
			}).Warn("Bundle's lifetime is expired")

			c.bundleDeletion(bp, arecord.LifetimeExpired)
			return
		}
	}

	if pnBlock, err := bp.MustBundle().ExtensionBlock(bundle.ExtBlockTypePreviousNodeBlock); err == nil {
		// Replace the PreviousNodeBlock
		prevEid := pnBlock.Value.(*bundle.PreviousNodeBlock).Endpoint()
		pnBlock.Value = bundle.NewPreviousNodeBlock(c.NodeId)

		log.WithFields(log.Fields{
			"bundle":  bp.ID(),
			"old_eid": prevEid,
			"new_eid": c.NodeId,
		}).Debug("Previous Node Block was updated")
	} else {
		// Append a new PreviousNodeBlock
		bp.MustBundle().AddExtensionBlock(bundle.NewCanonicalBlock(
			0, 0, bundle.NewPreviousNodeBlock(c.NodeId)))
	}

	var nodes []cla.ConvergenceSender
	var deleteAfterwards = true

	// Try a direct delivery or consult the RoutingAlgorithm otherwise.
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
				"bundle": bp.ID(),
				"cla":    node,
			}).Info("Sending bundle to a CLA (ConvergenceSender)")

			if err := node.Send(bp.MustBundle()); err != nil {
				log.WithFields(log.Fields{
					"bundle": bp.ID(),
					"cla":    node,
					"error":  err,
				}).Warn("Sending bundle failed")

				c.routing.ReportFailure(bp, node)
			} else {
				log.WithFields(log.Fields{
					"bundle": bp.ID(),
					"cla":    node,
				}).Printf("Sending bundle succeeded")

				once.Do(func() { bundleSent = true })
			}

			wg.Done()
		}(node)
	}

	wg.Wait()

	if hcBlock, err := bp.MustBundle().ExtensionBlock(bundle.ExtBlockTypeHopCountBlock); err == nil {
		hc := hcBlock.Value.(*bundle.HopCountBlock)
		hc.Decrement()
		hcBlock.Value = hc

		log.WithFields(log.Fields{
			"bundle":    bp.ID(),
			"hop_count": hc,
		}).Debug("Bundle's hop count block was resetted")
	}

	if bundleSent {
		if bp.MustBundle().PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestForward) {
			c.SendStatusReport(bp, arecord.ForwardedBundle, arecord.NoInformation)
		}

		if deleteAfterwards {
			bp.PurgeConstraints()
			bp.Sync()
		} else if c.InspectAllBundles && bp.MustBundle().IsAdministrativeRecord() {
			c.bundleContraindicated(bp)
			c.checkAdministrativeRecord(bp)
		}
	} else {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Info("Failed to forward bundle to any CLA")
		c.bundleContraindicated(bp)
	}
}

// checkAdministrativeRecord checks administrative records. If this method
// returns false, an error occured.
func (c *Core) checkAdministrativeRecord(bp BundlePack) bool {
	if !bp.MustBundle().IsAdministrativeRecord() {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("Bundle does not contain an administrative record")
		return false
	}

	canonicalAr, err := bp.MustBundle().PayloadBlock()
	if err != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"error":  err,
		}).Warn("Bundle with an administrative record flag misses payload block")

		return false
	}

	payload := canonicalAr.Value.(*bundle.PayloadBlock).Data()
	ar, err := arecord.NewAdministrativeRecordFromCbor(payload)
	if err != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"error":  err,
		}).Warn("Bundle with an administrative record could not be parsed")

		return false
	}

	log.WithFields(log.Fields{
		"bundle":    bp.ID(),
		"admin_rec": ar,
	}).Info("Received bundle contains an administrative record")

	// Currently there are only status reports. This must be changed if more
	// types of administrative records are introduced.
	c.inspectStatusReport(bp, ar)

	return true
}

func (c *Core) inspectStatusReport(bp BundlePack, ar arecord.AdministrativeRecord) {
	if ar.TypeCode() != arecord.ARTypeStatusReport {
		log.WithFields(log.Fields{
			"bundle":    bp.ID(),
			"type_code": ar.TypeCode(),
		}).Warn("Administrative record is not a status report")

		return
	}

	var status = *ar.(*arecord.StatusReport)
	var sips = status.StatusInformations()

	if len(sips) == 0 {
		log.WithFields(log.Fields{
			"bundle":    bp.ID(),
			"admin_rec": ar,
		}).Warn("Administrative record contains no status information")
		return
	}

	var bpStore, err = c.store.QueryId(status.RefBundle)
	if err != nil {
		log.WithFields(log.Fields{
			"bundle":     bp.ID(),
			"status_rep": status,
		}).Warn("Status Report's bundle is unknown")
		return
	}

	log.WithFields(log.Fields{
		"bundle":        bp.ID(),
		"status_rep":    status,
		"status_bundle": bpStore.Id,
	}).Debug("Status Report's referenced bundle was loaded")

	for _, sip := range sips {
		log.WithFields(log.Fields{
			"bundle":        bp.ID(),
			"status_rep":    status,
			"status_bundle": bpStore.Id,
			"information":   sip,
		}).Info("Parsing status report")

		switch sip {
		case arecord.ReceivedBundle, arecord.ForwardedBundle, arecord.DeletedBundle:
			// Nothing to do

		case arecord.DeliveredBundle:
			log.WithFields(log.Fields{
				"bundle":        bp.ID(),
				"status_rep":    status,
				"status_bundle": bpStore.Id,
			}).Info("Status report indicates delivered bundle, deleting bundle")

			c.store.Delete(bpStore.BId)

		default:
			log.WithFields(log.Fields{
				"bundle":        bp.ID(),
				"status_rep":    status,
				"status_bundle": bpStore.Id,
				"information":   int(sip),
			}).Warn("Status report has an unknown status information code")
		}
	}
}

func (c *Core) localDelivery(bp BundlePack) {
	// TODO: check fragmentation

	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Received bundle for local delivery")

	if bp.MustBundle().IsAdministrativeRecord() {
		if !c.checkAdministrativeRecord(bp) {
			c.bundleDeletion(bp, arecord.NoInformation)
			return
		}
	}

	bp.AddConstraint(LocalEndpoint)
	bp.Sync()

	for _, agent := range c.Agents {
		if agent.EndpointID() == bp.MustBundle().PrimaryBlock.Destination {
			agent.Deliver(bp)
		}
	}

	if bp.MustBundle().PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestDelivery) {
		c.SendStatusReport(bp, arecord.DeliveredBundle, arecord.NoInformation)
	}

	bp.PurgeConstraints()
	bp.Sync()
}

func (c *Core) bundleContraindicated(bp BundlePack) {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Bundle was marked for contraindication")

	bp.AddConstraint(Contraindicated)
	bp.Sync()
}

func (c *Core) bundleDeletion(bp BundlePack, reason arecord.StatusReportReason) {
	if bp.MustBundle().PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestDeletion) {
		c.SendStatusReport(bp, arecord.DeletedBundle, reason)
	}

	bp.PurgeConstraints()
	bp.Sync()

	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Bundle was marked for deletion")
}
