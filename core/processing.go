package core

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7/bundle"
	"github.com/dtn7/dtn7/cla"
)

// SendBundle transmits an outbounding bundle.
func (c *Core) SendBundle(bndl bundle.Bundle) {
	c.transmit(NewBundlePack(bndl))
}

// transmit starts the transmission of an outbounding bundle pack. Therefore
// the source's endpoint ID must be dtn:none or a member of this node.
func (c *Core) transmit(bp BundlePack) {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Transmission of bundle requested")

	c.idKeeper.update(bp.Bundle)

	bp.AddConstraint(DispatchPending)
	c.store.Push(bp)

	src := bp.Bundle.PrimaryBlock.SourceNode
	if src != bundle.DtnNone() && !c.HasEndpoint(src) {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"source": src,
		}).Info("Bundle's source is neither dtn:none nor an endpoint of this node")

		c.bundleDeletion(bp, NoInformation)
		return
	}

	c.dispatching(bp)
}

// receive handles received/incoming bundles.
func (c *Core) receive(bp BundlePack) {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Debug("Received new bundle")

	if KnowsBundle(c.store, bp) {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("Received bundle's ID is already known.")

		// bundleDeletion is _not_ called because this would delete the already
		// stored BundlePack.
		return
	}

	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Processing new received bundle")

	bp.AddConstraint(DispatchPending)
	c.store.Push(bp)

	if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestReception) {
		c.SendStatusReport(bp, ReceivedBundle, NoInformation)
	}

	for i := len(bp.Bundle.CanonicalBlocks) - 1; i >= 0; i-- {
		var cb = bp.Bundle.CanonicalBlocks[i]

		if isKnownBlockType(cb.BlockType) {
			continue
		}

		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"number": i,
			"type":   cb.BlockType,
		}).Warn("Bundle's canonical block is unknown")

		if cb.BlockControlFlags.Has(bundle.StatusReportBlock) {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
				"number": i,
				"type":   cb.BlockType,
			}).Info("Bundle's unknown canonical block requested reporting")

			c.SendStatusReport(bp, ReceivedBundle, BlockUnintelligible)
		}

		if cb.BlockControlFlags.Has(bundle.DeleteBundle) {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
				"number": i,
				"type":   cb.BlockType,
			}).Info("Bundle's unknown canonical block requested bundle deletion")

			c.bundleDeletion(bp, BlockUnintelligible)
			return
		}

		if cb.BlockControlFlags.Has(bundle.RemoveBlock) {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
				"number": i,
				"type":   cb.BlockType,
			}).Info("Bundle's unknown canonical block requested to be removed")

			bp.Bundle.CanonicalBlocks = append(
				bp.Bundle.CanonicalBlocks[:i], bp.Bundle.CanonicalBlocks[i+1:]...)
		}
	}

	c.dispatching(bp)
}

// dispatching handles the dispatching of received bundles.
func (c *Core) dispatching(bp BundlePack) {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Dispatching bundle")

	if c.HasEndpoint(bp.Bundle.PrimaryBlock.Destination) {
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
	c.store.Push(bp)

	if hcBlock, err := bp.Bundle.ExtensionBlock(bundle.HopCountBlock); err == nil {
		hc := hcBlock.Data.(bundle.HopCount)
		hc.Increment()
		hcBlock.Data = hc

		log.WithFields(log.Fields{
			"bundle":    bp.ID(),
			"hop_count": hc,
		}).Debug("Bundle contains an hop count block")

		if exceeded := hc.IsExceeded(); exceeded {
			log.WithFields(log.Fields{
				"bundle":    bp.ID(),
				"hop_count": hc,
			}).Info("Bundle contains an exceeded hop count block")

			c.bundleDeletion(bp, HopLimitExceeded)
			return
		}
	}

	if bp.Bundle.PrimaryBlock.IsLifetimeExceeded() {
		log.WithFields(log.Fields{
			"bundle":        bp.ID(),
			"primary_block": bp.Bundle.PrimaryBlock,
		}).Warn("Bundle's primary block's lifetime is exceeded")

		c.bundleDeletion(bp, LifetimeExpired)
		return
	}

	if age, err := bp.UpdateBundleAge(); err == nil {
		if age >= bp.Bundle.PrimaryBlock.Lifetime {
			log.WithFields(log.Fields{
				"bundle": bp.ID(),
			}).Warn("Bundle's lifetime is expired")

			c.bundleDeletion(bp, LifetimeExpired)
			return
		}
	}

	if pnBlock, err := bp.Bundle.ExtensionBlock(bundle.PreviousNodeBlock); err == nil {
		// Replace the PreviousNodeBlock
		prevEid := pnBlock.Data.(bundle.EndpointID)
		pnBlock.Data = c.NodeId

		log.WithFields(log.Fields{
			"bundle":  bp.ID(),
			"old_eid": prevEid,
			"new_eid": c.NodeId,
		}).Debug("Previous Node Block was updated")
	} else {
		// Append a new PreviousNodeBlock
		bp.Bundle.AddExtensionBlock(bundle.NewPreviousNodeBlock(0, 0, c.NodeId))
	}

	var nodes []cla.ConvergenceSender
	var deleteAfterwards = true

	// Try a direct delivery or consult the RoutingAlgorithm otherwise.
	nodes = c.senderForDestination(bp.Bundle.PrimaryBlock.Destination)
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

			if err := node.Send(*bp.Bundle); err != nil {
				log.WithFields(log.Fields{
					"bundle": bp.ID(),
					"cla":    node,
					"error":  err,
				}).Warn("Sending bundle failed")

				node.Close()
				c.RestartConvergence(node)
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

	if hcBlock, err := bp.Bundle.ExtensionBlock(bundle.HopCountBlock); err == nil {
		hc := hcBlock.Data.(bundle.HopCount)
		hc.Decrement()
		hcBlock.Data = hc

		log.WithFields(log.Fields{
			"bundle":    bp.ID(),
			"hop_count": hc,
		}).Debug("Bundle's hop count block was resetted")
	}

	if bundleSent {
		if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestForward) {
			c.SendStatusReport(bp, ForwardedBundle, NoInformation)
		}

		if deleteAfterwards {
			bp.PurgeConstraints()
			c.store.Push(bp)
		} else if c.InspectAllBundles && bp.Bundle.IsAdministrativeRecord() {
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
	if !bp.Bundle.IsAdministrativeRecord() {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
		}).Debug("Bundle does not contain an administrative record")
		return false
	}

	canonicalAr, err := bp.Bundle.PayloadBlock()
	if err != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"error":  err,
		}).Warn("Bundle with an administrative record flag misses payload block")

		return false
	}

	ar, err := NewAdministrativeRecordFromCbor(canonicalAr.Data.([]byte))
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

func (c *Core) inspectStatusReport(bp BundlePack, ar AdministrativeRecord) {
	var status = ar.Content
	var sips = status.StatusInformations()

	if len(sips) == 0 {
		log.WithFields(log.Fields{
			"bundle":    bp.ID(),
			"admin_rec": ar,
		}).Warn("Administrative record contains no status information")
		return
	}

	var bpStores = QueryFromStatusReport(c.store, status)
	if len(bpStores) != 1 {
		log.WithFields(log.Fields{
			"bundle":     bp.ID(),
			"status_rep": status,
			"store_numb": len(bpStores),
		}).Warn("Status Report's bundle is unknown")
		return
	}

	var bpStore = bpStores[0]
	log.WithFields(log.Fields{
		"bundle":        bp.ID(),
		"status_rep":    status,
		"status_bundle": bpStore.ID(),
	}).Debug("Status Report's referenced bundle was loaded")

	for _, sip := range sips {
		log.WithFields(log.Fields{
			"bundle":        bp.ID(),
			"status_rep":    status,
			"status_bundle": bpStore.ID(),
			"information":   sip,
		}).Info("Parsing status report")

		switch sip {
		case ReceivedBundle, ForwardedBundle, DeletedBundle:
			// Nothing to do

		case DeliveredBundle:
			log.WithFields(log.Fields{
				"bundle":        bp.ID(),
				"status_rep":    status,
				"status_bundle": bpStore.ID(),
			}).Info("Status report indicates delivered bundle, deleting bundle")

			bpStore.PurgeConstraints()
			c.store.Push(bpStore)

		default:
			log.WithFields(log.Fields{
				"bundle":        bp.ID(),
				"status_rep":    status,
				"status_bundle": bpStore.ID(),
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

	if bp.Bundle.IsAdministrativeRecord() {
		if !c.checkAdministrativeRecord(bp) {
			c.bundleDeletion(bp, NoInformation)
			return
		}
	}

	for _, agent := range c.Agents {
		if agent.EndpointID() == bp.Bundle.PrimaryBlock.Destination {
			agent.Deliver(bp.Bundle)
		}
	}

	c.routing.NotifyIncoming(bp)

	if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestDelivery) {
		c.SendStatusReport(bp, DeliveredBundle, NoInformation)
	}

	bp.PurgeConstraints()
	c.store.Push(bp)
}

func (c *Core) bundleContraindicated(bp BundlePack) {
	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Bundle was marked for contraindication")

	bp.AddConstraint(Contraindicated)
	c.store.Push(bp)
}

func (c *Core) bundleDeletion(bp BundlePack, reason StatusReportReason) {
	if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestDeletion) {
		c.SendStatusReport(bp, DeletedBundle, reason)
	}

	bp.PurgeConstraints()
	c.store.Push(bp)

	log.WithFields(log.Fields{
		"bundle": bp.ID(),
	}).Info("Bundle was marked for deletion")
}
