// Package appagent describes the ApplicationAgent interface for application
// agents.
//
// The subpackages should contain implementations of this interface, which could
// be used in the core package.
package appagent

import "github.com/geistesk/dtn7/bundle"

// ApplicationAgent is an interface which describes application agent, which can
// both receive and transmit bundles in interaction with the Core.
type ApplicationAgent interface {
	// EndpointID returns this ApplicationAgent's (unique) endpoint ID.
	EndpointID() bundle.EndpointID

	// Deliver delivers a received bundle to this ApplicationAgent. This bundle
	// may contain an application specific payload or an administrative record.
	Deliver(bndl *bundle.Bundle) error
}
