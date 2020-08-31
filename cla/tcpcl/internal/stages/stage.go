// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"errors"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla/tcpcl/internal/msgs"
)

// Configuration for stages.
type Configuration struct {
	// ActivePeer indicates it this peer is the "active" entity in the session.
	ActivePeer bool

	// ContactFlags determine the Contact Header.
	ContactFlags msgs.ContactFlags

	// Keepalive in seconds. A zero value indicates a disabled keepalive.
	Keepalive uint16

	// SegmentMru is the largest allowed single-segment payload to be received in bytes.
	SegmentMru uint64

	// TransferMru is the largest allowed total-bundle payload to be received in bytes.
	TransferMru uint64

	// NodeId is this node's ID.
	NodeId bundle.EndpointID
}

// StageClose signals a closed stage, after calling the Close() method.
var StageClose = errors.New("stage closed down")

// State for stages, both used as input and as an altered output.
type State struct {
	// Configuration to be used; should not be altered.
	Configuration Configuration

	// MsgIn and MsgOut are channels for incoming (receiving) and outgoing (sending) TCPCL Messages.
	MsgIn  <-chan msgs.Message
	MsgOut chan<- msgs.Message

	// StageError reports back the failure of a stage.
	StageError error

	// CONTACT STAGE
	// ContactFlags are the received ContactFlags.
	ContactFlags msgs.ContactFlags
	// CONTACT STAGE END

	// SESS INIT STAGE
	// Keepalive is the minimum of the own configured and the received keepalive. Zero indicates a disabled keepalive.
	Keepalive uint16
	// SegmentMtu is the peer's segment MTU.
	SegmentMtu uint64
	// TransferMtu is the peer's transfer MTU.
	TransferMtu uint64
	// PeerNodeId is the peer's node ID.
	PeerNodeId bundle.EndpointID
	// SESS INIT STAGE END
}

// Stage described by this interface. It should be started by the Start method, which gets a configuration.
type Stage interface {
	// Start this Stage based on the previous Stage's State.
	Start(state *State)

	// Exchanges returns two optional channels for Message exchange with the peer. Those channels are only available iff
	// the third exchangeOk variable is true. First channel is to send outgoing Messages to the peer, e.g.,
	// XFER_SEGMENTs, XFER_ACKs, XFER_REFUSE, MSG_REFUSE, or SESS_TERM. The other channel receives incoming messages.
	Exchanges() (outgoing chan<- msgs.Message, incoming <-chan msgs.Message, exchangeOk bool)

	// Close this Stage down.
	Close() error

	// Finished closes this channel to indicate this Stage has finished. Afterwards the State should be inspected.
	Finished() <-chan struct{}
}
