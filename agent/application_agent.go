package agent

import "github.com/dtn7/dtn7-go/bundle"

// ApplicationAgent is an interface to describe application agents, which can both receive and transmit Bundles.
type ApplicationAgent interface {
	// Endpoints returns the EndpointIDs that this ApplicationAgent answers to.
	Endpoints() []bundle.EndpointID

	// MessageReceiver is a channel on which the ApplicationAgent must listen for incoming Messages.
	MessageReceiver() chan Message

	// MessageSender is a channel to which the ApplicationAgent can send outgoing Messages.
	MessageSender() chan Message
}
