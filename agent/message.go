package agent

import "github.com/dtn7/dtn7-go/bundle"

// Message is a generic interface to specify an information exchange between an ApplicationAgent and some Manager.
// The following types named *Message are implementations of this interface.
type Message interface {
	// Recipients returns a list of endpoints to which this message is addressed.
	// However, if this message is not addressed to some specific endpoint, nil must be returned.
	Recipients() []bundle.EndpointID
}

// BundleMessage indicates a transmitted Bundle.
// If the Message is received from an ApplicationAgent, it is an incoming Bundle.
// If the Message is sent from an ApplicationAgent, it is an outgoing Bundle.
type BundleMessage struct {
	Bundle bundle.Bundle
}

// Recipients are the Bundle destination for a BundleMessage.
func (bm BundleMessage) Recipients() []bundle.EndpointID {
	return []bundle.EndpointID{bm.Bundle.PrimaryBlock.Destination}
}

// ShutdownMessage indicates the closing down of an ApplicationAgent.
// If the Message is received from an ApplicationAgent, it must close itself down.
// If the Message is sent from an ApplicationAgent, it is closing down itself.
type ShutdownMessage struct{}

// Recipients are not available for a ShutdownMessage.
func (sm ShutdownMessage) Recipients() []bundle.EndpointID {
	return nil
}

// messageForAgent checks if a Message is addressed to an ApplicationAgent.
func messageForAgent(message Message, agent ApplicationAgent) bool {
	matches := map[bundle.EndpointID]struct{}{}

	for _, eid := range message.Recipients() {
		matches[eid] = struct{}{}
	}

	for _, eid := range agent.Endpoints() {
		if _, ok := matches[eid]; ok {
			return true
		}
	}
	return false
}
