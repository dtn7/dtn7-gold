package core

import (
	"github.com/geistesk/dtn7/bundle"
)

type ApplicationAgent struct {
	ProtocolAgent *ProtocolAgent
	AppEndpoints  []bundle.EndpointID
}

// HasEndpoint returns true if the given endpoint ID is assigned either to an
// application or a CLA governed by this Application Agent.
func (aa ApplicationAgent) HasEndpoint(endpoint bundle.EndpointID) bool {
	for _, ep := range aa.AppEndpoints {
		if ep == endpoint {
			return true
		}
	}

	for _, cla := range aa.ProtocolAgent.ConvergenceLayers {
		if cla.GetEndpointID() == endpoint {
			return true
		}
	}

	return false
}
