package cla

import (
	"fmt"

	"github.com/dtn7/dtn7-go/bundle"
)

// ConvergenceMessageType indicates the kind of a ConvergenceStatus.
type ConvergenceMessageType uint

const (
	// ReceivedBundle shows the reception of a bundle.
	ReceivedBundle ConvergenceMessageType = iota
)

func (cms ConvergenceMessageType) String() string {
	switch cms {
	case ReceivedBundle:
		return "Received Bundle"
	default:
		return "Unknown Type"
	}
}

// ConvergenceStatus allows transmission of information via a return channel
// from a Convergence instance.
type ConvergenceStatus struct {
	Sender          Convergence
	RelatedEndpoint bundle.EndpointID
	MessageType     ConvergenceMessageType
	Message         interface{}
}

func (cs ConvergenceStatus) String() string {
	return fmt.Sprintf("%v-Convergence Status regarding %v from %v",
		cs.MessageType, cs.RelatedEndpoint, cs.Sender)
}

// NewConvergenceStatus creates a new ConvergenceStatus.
func NewConvergenceStatus(sender Convergence, relEid bundle.EndpointID,
	msgType ConvergenceMessageType, msg interface{}) ConvergenceStatus {
	return ConvergenceStatus{
		Sender:          sender,
		RelatedEndpoint: relEid,
		MessageType:     msgType,
		Message:         msg,
	}
}
