// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cla

import (
	"fmt"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// ConvergenceMessageType indicates the kind of a ConvergenceStatus.
type ConvergenceMessageType uint

const (
	_ ConvergenceMessageType = iota

	// ReceivedBundle shows the reception of a bundle. The Message's type must be
	// a ConvergenceReceivedBundle struct.
	ReceivedBundle

	// PeerDisappeared shows the disappearance of a peer. The Message's type must
	// be a bpv7.EndpointID.
	PeerDisappeared

	// PeerAppeared shows the appearance of a peer. The Message's type must be
	// a bpv7.EndpointID
	PeerAppeared
)

func (cms ConvergenceMessageType) String() string {
	switch cms {
	case ReceivedBundle:
		return "Received Bundle"
	case PeerDisappeared:
		return "Peer Disappeared"
	case PeerAppeared:
		return "Peer Appeared"
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
	Endpoint bpv7.EndpointID
	Bundle   *bpv7.Bundle
}

// NewConvergenceReceivedBundle creates a new ConvergenceStatus for a
// ReceivedBundle type, transmitting both EndpointID and Bundle pointer.
func NewConvergenceReceivedBundle(sender Convergence, eid bpv7.EndpointID, bndl *bpv7.Bundle) ConvergenceStatus {
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
func NewConvergencePeerDisappeared(sender Convergence, peerEid bpv7.EndpointID) ConvergenceStatus {
	return ConvergenceStatus{
		Sender:      sender,
		MessageType: PeerDisappeared,
		Message:     peerEid,
	}
}

// NewConvergencePeerAppeared creates a new ConvergenceStatus for a
// PeerAppeared type, transmission the appeared EndpointID.
func NewConvergencePeerAppeared(sender Convergence, peerEid bpv7.EndpointID) ConvergenceStatus {
	return ConvergenceStatus{
		Sender:      sender,
		MessageType: PeerAppeared,
		Message:     peerEid,
	}
}
