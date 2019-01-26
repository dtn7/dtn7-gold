package core

import (
	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla"
)

// ProtocolAgent is the Bundle Protocol Agent (BPA) which handles transmission
// and reception of bundles.
type ProtocolAgent struct {
	ApplicationAgent     *ApplicationAgent
	ConvergenceSenders   []cla.ConvergenceSender
	ConvergenceReceivers []cla.ConvergenceReceiver
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
