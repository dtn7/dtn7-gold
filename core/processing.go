package core

import (
	"log"
	"sync"

	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla"
)

// Transmit starts the transmission of an outbounding bundle pack. Therefore
// the source's endpoint ID must be dtn:none or a member of this node.
func (c Core) Transmit(bp BundlePack) {
	log.Printf("Transmission of bundle requested: %v", bp.Bundle)

	c.IdKeeper.update(bp.Bundle)

	bp.AddConstraint(DispatchPending)
	c.Store.Push(bp)

	src := bp.Bundle.PrimaryBlock.SourceNode
	if src != bundle.DtnNone() && !c.HasEndpoint(src) {
		log.Printf(
			"Bundle's source %v is neither dtn:none nor an endpoint of this node", src)

		c.BundleDeletion(bp, NoInformation)
		return
	}

	c.Forward(bp)
}

// Receive handles received/incomming bundles.
func (c Core) Receive(bp BundlePack) {
	log.Printf("Received new bundle: %v", bp.Bundle)

	bp.AddConstraint(DispatchPending)
	c.Store.Push(bp)

	if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestReception) {
		c.SendStatusReport(bp, ReceivedBundle, NoInformation)
	}

	for i := len(bp.Bundle.CanonicalBlocks) - 1; i >= 0; i-- {
		var cb = bp.Bundle.CanonicalBlocks[i]

		if isKnownBlockType(cb.BlockType) {
			continue
		}

		log.Printf("Bundle's %v canonical block is unknown, type %d",
			bp.Bundle, cb.BlockType)

		if cb.BlockControlFlags.Has(bundle.StatusReportBlock) {
			log.Printf("Bundle's %v unknown canonical block requested reporting",
				bp.Bundle)

			c.SendStatusReport(bp, ReceivedBundle, BlockUnintelligible)
		}

		if cb.BlockControlFlags.Has(bundle.DeleteBundle) {
			log.Printf("Bundle's %v unknown canonical block requested bundle deletion",
				bp.Bundle)

			c.BundleDeletion(bp, BlockUnintelligible)
			return
		}

		if cb.BlockControlFlags.Has(bundle.RemoveBlock) {
			log.Printf("Bundle's %v unknown canonical block requested to be removed",
				bp.Bundle)

			bp.Bundle.CanonicalBlocks = append(
				bp.Bundle.CanonicalBlocks[:i], bp.Bundle.CanonicalBlocks[i+1:]...)
		}
	}

	c.Dispatching(bp)
}

// Dispatching handles the dispatching of received bundles.
func (c Core) Dispatching(bp BundlePack) {
	log.Printf("Dispatching bundle %v", bp.Bundle)

	if c.HasEndpoint(bp.Bundle.PrimaryBlock.Destination) {
		c.LocalDelivery(bp)
	} else {
		c.Forward(bp)
	}
}

// Forward forwards a bundle pack's bundle to another node.
func (c Core) Forward(bp BundlePack) {
	log.Printf("Bundle will be forwarded: %v", bp.Bundle)

	bp.AddConstraint(ForwardPending)
	bp.RemoveConstraint(DispatchPending)
	c.Store.Push(bp)

	if hcBlock, err := bp.Bundle.ExtensionBlock(bundle.HopCountBlock); err == nil {
		hc := hcBlock.Data.(bundle.HopCount)
		hc.Increment()
		hcBlock.Data = hc

		log.Printf("Bundle %v contains an hop count block: %v", bp.Bundle, hc)

		if exceeded := hc.IsExceeded(); exceeded {
			log.Printf("Bundle contains an exceeded hop count block: %v", hc)

			c.BundleDeletion(bp, HopLimitExceeded)
			return
		}
	}

	if bp.Bundle.PrimaryBlock.IsLifetimeExceeded() {
		log.Printf("Bundle's primary block's lifetime is exceeded: %v",
			bp.Bundle.PrimaryBlock)

		c.BundleDeletion(bp, LifetimeExpired)
		return
	}

	if age, err := bp.UpdateBundleAge(); err == nil {
		if age >= bp.Bundle.PrimaryBlock.Lifetime {
			log.Printf("Bundle's lifetime is expired")

			c.BundleDeletion(bp, LifetimeExpired)
			return
		}
	}

	var nodes []cla.ConvergenceSender

	nodes = c.clasForDestination(bp.Bundle.PrimaryBlock.Destination)
	if nodes == nil {
		nodes = c.clasForBudlePack(bp)
	}

	if nodes == nil {
		// No nodes could be selected, the bundle will be contraindicated.
		c.BundleContraindicated(bp)
		return
	}

	var bundleSent = false

	var wg sync.WaitGroup
	var once sync.Once

	wg.Add(len(nodes))

	for _, node := range nodes {
		go func(node cla.ConvergenceSender) {
			log.Printf("Trying to deliver bundle %v to %v", bp.Bundle, node)

			if err := node.Send(*bp.Bundle); err != nil {
				log.Printf("Transmission of bundle %v failed to %v: %v",
					bp.Bundle, node, err)
			} else {
				log.Printf("Transmission of bundle %v succeeded to %v", bp.Bundle, node)

				once.Do(func() { bundleSent = true })
			}

			wg.Done()
		}(node)
	}

	wg.Wait()

	if bundleSent {
		if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestForward) {
			c.SendStatusReport(bp, ForwardedBundle, NoInformation)
		}

		bp.RemoveConstraint(ForwardPending)
		c.Store.Push(bp)
	} else {
		log.Printf("Failed to forward %v", bp.Bundle)

		c.BundleContraindicated(bp)
	}
}

func (c Core) LocalDelivery(bp BundlePack) {
	// TODO: check fragmentation
	// TODO: handle delivery

	log.Printf("Received delivered bundle: %v", bp.Bundle)

	if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.AdministrativeRecordPayload) {
		canonicalAr, err := bp.Bundle.PayloadBlock()
		if err != nil {
			log.Printf("Bundle %v with an administrative record payload misses payload: %v",
				bp.Bundle, err)

			c.BundleDeletion(bp, NoInformation)
			return
		}

		ar, err := NewAdministrativeRecordFromCbor(canonicalAr.Data.([]byte))
		if err != nil {
			log.Printf("Bundle %v with an administrative record could not be parsed: %v",
				bp.Bundle, err)

			c.BundleDeletion(bp, NoInformation)
			return
		}

		log.Printf("Received bundle %v contains an administrative record: %v",
			bp.Bundle, ar)
	}

	if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestDelivery) {
		c.SendStatusReport(bp, DeliveredBundle, NoInformation)
	}
}

func (c Core) BundleContraindicated(bp BundlePack) {
	// TODO: implement :^)
	log.Printf("Bundle %v was marked for contraindication", bp.Bundle)
}

func (c Core) BundleDeletion(bp BundlePack, reason StatusReportReason) {
	if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.StatusRequestDeletion) {
		c.SendStatusReport(bp, DeletedBundle, reason)
	}

	bp.PurgeConstraints()
	c.Store.Push(bp)

	// TODO: implement (^^,)
	log.Printf("Bundle %v was marked for deletion", bp.Bundle)
}
