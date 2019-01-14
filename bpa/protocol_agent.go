package bpa

import (
	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla"
)

// ProtocolAgent is the Bundle Protocol Agent (BPA) which handles transmission
// and reception of bundles.
type ProtocolAgent struct {
	ConvergenceLayers []ConvergenceLayer
}

// HasEndpoint returns true if the given endpoint ID is assigned to this
// present node.
func (pa ProtocolAgent) HasEndpoint(endpoint bundle.EndpointID) bool {
	// TODO: check other Endpoint IDs next to those of the CLAs

	for _, cla := range pa.ConvergenceLayers {
		if cla.GetEndpointID() == endpoint {
			return true
		}
	}

	return false
}
