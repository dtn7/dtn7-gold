package core

import (
	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla"
)

// ProtocolAgent is the Bundle Protocol Agent (BPA) which handles transmission
// and reception of bundles.
type ProtocolAgent struct {
	ApplicationAgent  *ApplicationAgent
	ConvergenceLayers []cla.ConvergenceLayer
}

func (pa ProtocolAgent) clasForDestination(endpoint bundle.EndpointID) []cla.ConvergenceLayer {
	var clas []cla.ConvergenceLayer

	for _, cla := range pa.ConvergenceLayers {
		if cla.GetPeerEndpointID() == endpoint {
			clas = append(clas, cla)
		}
	}

	return clas
}

func (pa ProtocolAgent) clasForBudlePack(bp BundlePack) []cla.ConvergenceLayer {
	// TODO: This software is kind of stupid at this moment and will return all
	// currently known CLAs.

	return pa.ConvergenceLayers
}
