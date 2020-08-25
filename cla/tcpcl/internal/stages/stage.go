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

	// Keepalive in seconds.
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
	MsgIn  chan msgs.Message
	MsgOut chan msgs.Message

	// StageError reports back the failure of a stage.
	StageError error

	// CONTACT STAGE
	ContactFlags msgs.ContactFlags
	// CONTACT STAGE END
}

// Stage described by this interface. It should be started by the Start method, which gets a configuration.
type Stage interface {
	// Start this Stage based on the previous Stage's State.
	Start(state *State)

	// Close this Stage down.
	Close() error

	// Finished closes this channel to indicate this Stage has finished. Afterwards the State should be inspected.
	Finished() chan struct{}
}
