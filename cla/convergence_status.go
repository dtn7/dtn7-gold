package cla

import (
	"fmt"

	"github.com/dtn7/dtn7-go/bundle"
)

// ConvergenceMessageType indicates the kind of a ConvergenceStatus.
type ConvergenceMessageType uint

const (
	_ ConvergenceMessageType = iota

	// ReceivedBundle shows the reception of a bundle. The Message's type must be
	// a ConvergenceReceivedBundle struct.
	ReceivedBundle

	// PeerDisappeared shows the disappearance of a peer. The Message's type must
	// be a bundle.EndpointID.
	PeerDisappeared
)

func (cms ConvergenceMessageType) String() string {
	switch cms {
	case ReceivedBundle:
		return "Received Bundle"
	case PeerDisappeared:
		return "Peer Disappeared"
	default:
		return "Unknown Type"
	}
}

// ConvergenceStatus allows transmission of information via a return channel
// from a Convergence instance.
type ConvergenceStatus struct {
	Sender      Convergence
	MessageType ConvergenceMessageType
	Message     interface{}
}

func (cs ConvergenceStatus) String() string {
	return fmt.Sprintf("%v-Convergence Status from %v", cs.MessageType, cs.Sender)
}

// ConvergenceReceivedBundle is an optional Message content for a
// ConvergenceStatus for the ReceivedBundle MessageType.
type ConvergenceReceivedBundle struct {
	Endpoint bundle.EndpointID
	Bundle   *bundle.Bundle
}

// NewConvergenceReceivedBundle creates a new ConvergenceStatus for a
// ReceivedBundle type, transmitting both EndpointID and Bundle pointer.
func NewConvergenceReceivedBundle(sender Convergence, eid bundle.EndpointID, bndl *bundle.Bundle) ConvergenceStatus {
	return ConvergenceStatus{
		Sender:      sender,
		MessageType: ReceivedBundle,
		Message: ConvergenceReceivedBundle{
			Endpoint: eid,
			Bundle:   bndl,
		},
	}
}

// NewConvergencePeerDisappeared creates a new ConvergenceStatus for a
// PeerDisappeared type, transmission the disappeared EndpointID.
func NewConvergencePeerDisappeared(sender Convergence, peerEid bundle.EndpointID) ConvergenceStatus {
	return ConvergenceStatus{
		Sender:      sender,
		MessageType: PeerDisappeared,
		Message:     peerEid,
	}
}
