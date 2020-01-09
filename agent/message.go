package agent

import "github.com/dtn7/dtn7-go/bundle"

// Message is a generic interface to specify an information exchange between an ApplicationAgent and some Manager.
// The following types named *Message are implementations of this interface.
type Message interface{}

// BundleMessage indicates a transmitted Bundle.
// If the Message is received from an ApplicationAgent, it is an incoming Bundle.
// If the Message is sent from an ApplicationAgent, it is an outgoing Bundle.
type BundleMessage struct {
	Bundle bundle.Bundle
}

// ShutdownMessage indicates the closing down of an ApplicationAgent.
// If the Message is received from an ApplicationAgent, it must close itself down.
// If the Message is sent from an ApplicationAgent, it is closing down itself.
type ShutdownMessage struct{}
