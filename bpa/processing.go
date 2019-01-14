package bpa

import (
	"fmt"
)

// Transmit starts the transmission of an outbounding bundle pack. Therefore
// the source's endpoint ID must be dtn:none or a member of this node.
func (pa ProtocolAgent) Transmit(bp BundlePack) error {
	bp.AddConstraint(DispatchPending)

	src := bp.BundlePack.Bundle.PrimaryBlock.SourceNode
	if src != bundle.DtnNone() || !pa.HasEndpoint(src) {
		return newBpaError(fmt.Sprintf(
			"Bundle's source endpoint %v is neither dtn:none nor member of this node",
			src))
	}

	pa.Forward(bp)
}

// Forward forwards a bundle pack's bundle to another node.
func (pa ProtocolAgent) Forward(bp BundlePack) error {
	bp.AddConstraint(ForwardPending)
	bp.RemoveConstraint(DispatchPending)

	if hcBlock, err := pa.Block.ExtensionBlock(bundle.HopCountBlock); err == nil {
		if exceeded := hcBlock.Data.(bundle.HopCount).IsExceeded(); exceeded {
			return newBpaError("Bundle's hop limit exceeded")
		}
	}

	// TODO: continue work here from 5.4, step 2
}
