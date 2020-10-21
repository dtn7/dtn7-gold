// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package stages

import (
	"errors"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
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
	NodeId bpv7.EndpointID
}

// StageClose signals a closed stage, after calling the Close() method.
var StageClose = errors.New("stage closed down")

// State for stages, both used as input and as an altered output.
type State struct {
	// Configuration to be used; should not be altered.
	Configuration Configuration

	// MsgIn and MsgOut are channels for incoming (receiving) and outgoing (sending) TCPCLv4 messages with an underlying
	// connector, e.g., an util.MessageSwitch.
	MsgIn  <-chan msgs.Message
	MsgOut chan<- msgs.Message

	// ExchangeMsgIn and ExchangeMsgOut are channels for incoming (receiving) and outgoing (sending) TCPCLv4 messages
	// with a higher-level util, e.g., an util.TransferManager.
	ExchangeMsgIn  chan msgs.Message
	ExchangeMsgOut chan msgs.Message

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
	PeerNodeId bpv7.EndpointID
	// SESS INIT STAGE END
}

// Stage described by this interface.
type Stage interface {
	// Handle this Stage's action based on the previous Stage's State and the StageHandler's close channel.
	Handle(state *State, closeChan <-chan struct{})
}
