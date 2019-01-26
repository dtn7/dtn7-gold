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

// Forward forwards a bundle pack's bundle to another node.
func (pa ProtocolAgent) Forward(bp BundlePack) {
	bp.AddConstraint(ForwardPending)
	bp.RemoveConstraint(DispatchPending)

	if hcBlock, err := bp.Bundle.ExtensionBlock(bundle.HopCountBlock); err == nil {
		if exceeded := hcBlock.Data.(bundle.HopCount).IsExceeded(); exceeded {
			pa.BundleDeletion(bp)
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
			pa.BundleDeletion(bp)
			return
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(nodes))

	for _, node := range nodes {
		go func(node cla.ConvergenceSender) {
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

func (pa ProtocolAgent) BundleContraindicated(bp BundlePack) {
	// TODO: implement :^)
	log.Printf("Bundle %v was marked for contraindication", bp)
}

func (pa ProtocolAgent) BundleDeletion(bp BundlePack) {
	// TODO: implement (^^,)
	log.Printf("Bundle %v was marked for deletion", bp)
}
