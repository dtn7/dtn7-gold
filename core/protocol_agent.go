package core

import (
	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla"
)

// isKnownBlockType checks if this program's core knows the given block type.
func isKnownBlockType(blocktype bundle.CanonicalBlockType) bool {
	switch blocktype {
	case
		bundle.PayloadBlock,
		bundle.PreviousNodeBlock,
		bundle.BundleAgeBlock,
		bundle.HopCountBlock:
		return true

	default:
		return false
	}
}

// ProtocolAgent is the Bundle Protocol Agent (BPA) which handles transmission
// and reception of bundles.
type ProtocolAgent struct {
	ApplicationAgent     *ApplicationAgent
	ConvergenceSenders   []cla.ConvergenceSender
	ConvergenceReceivers []cla.ConvergenceReceiver
}

func (pa *ProtocolAgent) RegisterConvergenceSender(sender cla.ConvergenceSender) {
	pa.ConvergenceSenders = append(pa.ConvergenceSenders, sender)
}

func (pa *ProtocolAgent) RegisterConvergenceReceiver(rec cla.ConvergenceReceiver) {
	pa.ConvergenceReceivers = append(pa.ConvergenceReceivers, rec)

	go func() {
		var chnl = rec.Channel()
		for {
			select {
			case bndl := <-chnl:
				pa.Receive(NewBundlePack(bndl))
			}
		}
	}()
}

func (pa ProtocolAgent) clasForDestination(endpoint bundle.EndpointID) []cla.ConvergenceSender {
	var clas []cla.ConvergenceSender

	for _, cla := range pa.ConvergenceSenders {
		if cla.GetPeerEndpointID() == endpoint {
			clas = append(clas, cla)
		}
	}

	return clas
}

func (pa ProtocolAgent) clasForBudlePack(bp BundlePack) []cla.ConvergenceSender {
	// TODO: This software is kind of stupid at this moment and will return all
	// currently known CLAs.

	return pa.ConvergenceSenders
}
