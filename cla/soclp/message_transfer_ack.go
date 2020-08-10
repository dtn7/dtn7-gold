// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"fmt"
	"io"

	"github.com/dtn7/cboring"
)

// TransferAckMessage acknowledges a successful TransferMessage, referring its identifier.
type TransferAckMessage struct {
	Identifier uint64
}

// NewTransferAckMessage referring a TransferMessage's identifier.
func NewTransferAckMessage(identifier uint64) *TransferAckMessage {
	return &TransferAckMessage{Identifier: identifier}
}

// Type code of a TransferAckMessage, always 3.
func (am *TransferAckMessage) Type() uint64 {
	return MsgTransferAck
}

func (am *TransferAckMessage) String() string {
	return fmt.Sprintf("TransferAckMessage(%d)", am.Identifier)
}

// MarshalCbor serializes the identifier.
func (am *TransferAckMessage) MarshalCbor(w io.Writer) error {
	return cboring.WriteUInt(am.Identifier, w)
}

// UnmarshalCbor the CBOR uint identifier back to a TransferAckMessage.
func (am *TransferAckMessage) UnmarshalCbor(r io.Reader) (err error) {
	am.Identifier, err = cboring.ReadUInt(r)
	return
}
