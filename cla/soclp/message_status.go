// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"fmt"
	"io"
	"strings"

	"github.com/dtn7/cboring"
)

const (
	// StatusShutdown represents a Shutdown Status Message.
	StatusShutdown uint64 = 0

	// StatusHeartbeat represents a Heartbeat Status Message.
	StatusHeartbeat uint64 = 1
)

// StatusMessage informs the peer about some status, e.g., a heartbeat or a scheduled shutdown.
type StatusMessage struct {
	StatusCode uint64
}

// NewShutdownStatusMessage creates a Shutdown Status Message.
func NewShutdownStatusMessage() *StatusMessage {
	return &StatusMessage{StatusShutdown}
}

// NewHeartbeatStatusMessage creates a new Heartbeat Status Message.
func NewHeartbeatStatusMessage() *StatusMessage {
	return &StatusMessage{StatusHeartbeat}
}

// Type code of a StatusMessage is always 1.
func (sm *StatusMessage) Type() uint64 {
	return MsgStatus
}

// IsShutdown checks if this StatusMessage represents a Shutdown Status Message.
func (sm *StatusMessage) IsShutdown() bool {
	return sm.StatusCode == StatusShutdown
}

// IsHeartbeat checks if this StatusMessage represents a Heartbeat Status Message.
func (sm *StatusMessage) IsHeartbeat() bool {
	return sm.StatusCode == StatusHeartbeat
}

func (sm *StatusMessage) String() string {
	var builder strings.Builder

	_, _ = fmt.Fprint(&builder, "StatusMessage(")
	switch sm.StatusCode {
	case StatusShutdown:
		_, _ = fmt.Fprint(&builder, "Shutdown")
	case StatusHeartbeat:
		_, _ = fmt.Fprint(&builder, "Heartbeat")
	default:
		_, _ = fmt.Fprint(&builder, "Invalid Status Code")
	}
	_, _ = fmt.Fprint(&builder, ")")

	return builder.String()
}

// MarshalCbor serializes the status code.
func (sm *StatusMessage) MarshalCbor(w io.Writer) error {
	return cboring.WriteUInt(sm.StatusCode, w)
}

// UnmarshalCbor a CBOR-represented status code back to an IdentityMessage.
func (sm *StatusMessage) UnmarshalCbor(r io.Reader) (err error) {
	sm.StatusCode, err = cboring.ReadUInt(r)
	return err
}
