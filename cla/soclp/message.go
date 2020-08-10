// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"fmt"
	"io"

	"github.com/dtn7/cboring"
)

const (
	// MsgIdentity is a IdentityMessage type code, uint 0.
	MsgIdentity uint64 = 0

	// MsgStatus is a StatusMessage type code, uint 1.
	MsgStatus uint64 = 1

	// MsgTransfer is a TransferMessage type code, uint 2.
	MsgTransfer uint64 = 2

	// MsgTransferAck is a TransferAckMessage type code, uint 3.
	MsgTransferAck uint64 = 3
)

// MessageType is an implementation of a Message, identified by its type code.
type MessageType interface {
	// Type code of this MessageType.
	Type() uint64

	fmt.Stringer
	cboring.CborMarshaler
}

// Message is the data structure to be exchanged between two peers.
//
// A message consist of two fields: A type code to identify the specific Message data and the data itself.
type Message struct {
	MessageType MessageType
}

// Type code of the MessageType.
func (m Message) Type() uint64 {
	return m.MessageType.Type()
}

func (m Message) String() string {
	return m.MessageType.String()
}

// MarshalCbor creates a CBOR array of two elements: type code followed by the MessageType representation.
func (m *Message) MarshalCbor(w io.Writer) (err error) {
	if err = cboring.WriteArrayLength(2, w); err != nil {
		return
	}

	if err = cboring.WriteUInt(m.Type(), w); err != nil {
		return
	}
	if err = cboring.Marshal(m.MessageType, w); err != nil {
		return
	}

	return
}

// UnmarshalCbor a CBOR array back to a Message.
func (m *Message) UnmarshalCbor(r io.Reader) (err error) {
	if n, arrErr := cboring.ReadArrayLength(r); arrErr != nil {
		return arrErr
	} else if n != 2 {
		return fmt.Errorf("Message expected array of length 2, got %d", n)
	}

	if typeCode, typeErr := cboring.ReadUInt(r); typeErr != nil {
		return typeErr
	} else {
		switch typeCode {
		case MsgIdentity:
			m.MessageType = new(IdentityMessage)
		case MsgStatus:
			m.MessageType = new(StatusMessage)
		case MsgTransfer:
			m.MessageType = new(TransferMessage)
		case MsgTransferAck:
			m.MessageType = new(TransferAckMessage)
		default:
			return fmt.Errorf("Message type code %d is undefined", typeCode)
		}

		if err = cboring.Unmarshal(m.MessageType, r); err != nil {
			return
		}
	}

	return
}
