package core

import (
	"log"
	"sync"

	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla"
)

// Transmit starts the transmission of an outbounding bundle pack. Therefore
// the source's endpoint ID must be dtn:none or a member of this node.
func (pa ProtocolAgent) Transmit(bp BundlePack) {
	log.Printf("Transmission of bundle requested: %v", bp.Bundle)

	bp.AddConstraint(DispatchPending)

	src := bp.Bundle.PrimaryBlock.SourceNode
	if src != bundle.DtnNone() && !pa.ApplicationAgent.HasEndpoint(src) {
		log.Printf(
			"Bundle's source %v is neither dtn:none nor an endpoint of this node", src)

		pa.BundleDeletion(bp)
		return
	}

	pa.Forward(bp)
}

// Receive handles received/incomming bundles.
func (pa ProtocolAgent) Receive(bp BundlePack) {
	log.Printf("Received new bundle: %v", bp.Bundle)

	bp.AddConstraint(DispatchPending)

	// TODO: create reception report

	for i := len(bp.Bundle.CanonicalBlocks) - 1; i >= 0; i-- {
		var cb = bp.Bundle.CanonicalBlocks[i]

		if isKnownBlockType(cb.BlockType) {
			continue
		}

		log.Printf("Bundle's canonical block is unknown, type %d", cb.BlockType)

		// TODO: create status report

		if cb.BlockControlFlags.Has(bundle.DeleteBundle) {
			log.Printf("Bundle's unknown canonical block requested deletion")

			pa.BundleDeletion(bp)
			return
		}

		if cb.BlockControlFlags.Has(bundle.RemoveBlock) {
			log.Printf("Bundle's unknown canonical block requested to be removed")

			bp.Bundle.CanonicalBlocks = append(
				bp.Bundle.CanonicalBlocks[:i], bp.Bundle.CanonicalBlocks[i+1:]...)
		}
	}

	pa.Dispatching(bp)
}

// Dispatching handles the dispatching of received bundles.
func (pa ProtocolAgent) Dispatching(bp BundlePack) {
	if pa.ApplicationAgent.HasEndpoint(bp.Bundle.PrimaryBlock.Destination) {
		pa.LocalDelivery(bp)
	} else {
		pa.Forward(bp)
	}
}

// Forward forwards a bundle pack's bundle to another node.
func (pa ProtocolAgent) Forward(bp BundlePack) {
	log.Printf("Bundle will be forwarded: %v", bp.Bundle)

	bp.AddConstraint(ForwardPending)
	bp.RemoveConstraint(DispatchPending)

	if hcBlock, err := bp.Bundle.ExtensionBlock(bundle.HopCountBlock); err == nil {
		hc := hcBlock.Data.(bundle.HopCount)
		hc.Increment()
		hcBlock.Data = hc

		log.Printf("Bundle contains an hop count block: %v", hc)

		if exceeded := hc.IsExceeded(); exceeded {
			log.Printf("Bundle contains an exceeded hop count block: %v", hc)

			pa.BundleDeletion(bp)
			return
		}
	}

	var nodes []cla.ConvergenceSender

	nodes = pa.clasForDestination(bp.Bundle.PrimaryBlock.Destination)
	if nodes == nil {
		nodes = pa.clasForBudlePack(bp)
	}

	if nodes == nil {
		// No nodes could be selected, the bundle will be contraindicated.
		pa.BundleContraindicated(bp)
		return
	}

	if age, err := bp.UpdateBundleAge(); err == nil {
		if age >= bp.Bundle.PrimaryBlock.Lifetime {
			log.Printf("Bundle's lifetime is expired")

			pa.BundleDeletion(bp)
			return
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(nodes))

	for _, node := range nodes {
		go func(node cla.ConvergenceSender) {
			log.Printf("Trying to deliver bundle to %v", node)

			if err := node.Send(*bp.Bundle); err != nil {
				log.Printf("ConvergenceSender %v failed to transmit bundle %v: %v",
					node, bp, err)
			} else {
				log.Printf("ConvergenceSender %v transmited bundle %v", node, bp)
			}

			wg.Done()
		}(node)
	}

	// TODO: create status report
	// TODO: contraindicate bundle in case of failure

	wg.Wait()

	bp.RemoveConstraint(ForwardPending)
}

func (pa ProtocolAgent) LocalDelivery(bp BundlePack) {
	// TODO: check fragmentation
	// TODO: handle delivery

	log.Printf("Received delivered bundle: %v", bp.Bundle)

	// TODO: report
}

func (pa ProtocolAgent) BundleContraindicated(bp BundlePack) {
	// TODO: implement :^)
	log.Printf("Bundle %v was marked for contraindication", bp)
}

func (pa ProtocolAgent) BundleDeletion(bp BundlePack) {
	// TODO: implement (^^,)
	log.Printf("Bundle %v was marked for deletion", bp)
}
