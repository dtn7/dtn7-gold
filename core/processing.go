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

	bp.AddConstraint(DispatchPending)

	src := bp.Bundle.PrimaryBlock.SourceNode
	if src != bundle.DtnNone() && !c.HasEndpoint(src) {
		log.Printf(
			"Bundle's source %v is neither dtn:none nor an endpoint of this node", src)

		c.BundleDeletion(bp)
		return
	}

	c.Forward(bp)
}

// Receive handles received/incomming bundles.
func (c Core) Receive(bp BundlePack) {
	log.Printf("Received new bundle: %v", bp.Bundle)

	bp.AddConstraint(DispatchPending)

	// TODO: unfuck this fuckery
	c.SendStatusReport(bp, ReceivedBundle, NoInformation)

	for i := len(bp.Bundle.CanonicalBlocks) - 1; i >= 0; i-- {
		var cb = bp.Bundle.CanonicalBlocks[i]

		if isKnownBlockType(cb.BlockType) {
			continue
		}

		log.Printf("Bundle's canonical block is unknown, type %d", cb.BlockType)

		// TODO: create status report

		if cb.BlockControlFlags.Has(bundle.DeleteBundle) {
			log.Printf("Bundle's unknown canonical block requested deletion")

			c.BundleDeletion(bp)
			return
		}

		if cb.BlockControlFlags.Has(bundle.RemoveBlock) {
			log.Printf("Bundle's unknown canonical block requested to be removed")

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

	if hcBlock, err := bp.Bundle.ExtensionBlock(bundle.HopCountBlock); err == nil {
		hc := hcBlock.Data.(bundle.HopCount)
		hc.Increment()
		hcBlock.Data = hc

		log.Printf("Bundle %v contains an hop count block: %v", bp.Bundle, hc)

		if exceeded := hc.IsExceeded(); exceeded {
			log.Printf("Bundle contains an exceeded hop count block: %v", hc)

			c.BundleDeletion(bp)
			return
		}
	}

	if bp.Bundle.PrimaryBlock.IsLifetimeExceeded() {
		log.Printf("Bundle's primary block's lifetime is exceeded: %v",
			bp.Bundle.PrimaryBlock)

		c.BundleDeletion(bp)
		return
	}

	if age, err := bp.UpdateBundleAge(); err == nil {
		if age >= bp.Bundle.PrimaryBlock.Lifetime {
			log.Printf("Bundle's lifetime is expired")

			c.BundleDeletion(bp)
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

	var wg sync.WaitGroup
	wg.Add(len(nodes))

	for _, node := range nodes {
		go func(node cla.ConvergenceSender) {
			log.Printf("Trying to deliver bundle %v to %v", bp.Bundle, node)

			if err := node.Send(*bp.Bundle); err != nil {
				log.Printf("ConvergenceSender %v failed to transmit bundle %v: %v",
					node, bp.Bundle, err)
			} else {
				log.Printf("ConvergenceSender %v transmited bundle %v", node, bp.Bundle)
			}

			wg.Done()
		}(node)
	}

	// TODO: create status report
	// TODO: contraindicate bundle in case of failure

	wg.Wait()

	bp.RemoveConstraint(ForwardPending)
}

func (c Core) LocalDelivery(bp BundlePack) {
	// TODO: check fragmentation
	// TODO: handle delivery

	log.Printf("Received delivered bundle: %v", bp.Bundle)

	// TODO: report
}

func (c Core) BundleContraindicated(bp BundlePack) {
	// TODO: implement :^)
	log.Printf("Bundle %v was marked for contraindication", bp.Bundle)
}

func (c Core) BundleDeletion(bp BundlePack) {
	// TODO: implement (^^,)
	log.Printf("Bundle %v was marked for deletion", bp.Bundle)
}
