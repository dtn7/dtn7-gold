// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"encoding/binary"
	"fmt"
	"io"
)

// MessageRejectionReason is the one-octet refusal code from a MessageRejectionMessage.
type MessageRejectionReason uint8

const (
	// RejectionTypeUnknown indicates an unknown Message Type Code.
	RejectionTypeUnknown MessageRejectionReason = 0x01

	// RejectionUnsupported indicates that this TCPCL node cannot comply with
	// the message content.
	RejectionUnsupported MessageRejectionReason = 0x02

	// RejectionUnexptected indicates that this TCPCL node received a message
	// while the session is in a state in which the message is not expected.
	RejectionUnexptected MessageRejectionReason = 0x03
)

// IsValid checks if this MessageRejectionReason represents a valid value.
func (mrr MessageRejectionReason) IsValid() bool {
	switch mrr {
	case RejectionTypeUnknown, RejectionUnsupported, RejectionUnexptected:
		return true
	default:
		return false
	}
}

func (mrr MessageRejectionReason) String() string {
	switch mrr {
	case RejectionTypeUnknown:
		return "Message Type Unknown"
	case RejectionUnsupported:
		return "Message Unsupported"
	case RejectionUnexptected:
		return "Message Unexpected"
	default:
		return "INVALID"
	}
}

// MSG_REJECT is the Message Header code for a Message Rejection Message.
const MSG_REJECT uint8 = 0x06

// MessageRejectionMessage is the MSG_REJECT message for message rejection.
type MessageRejectionMessage struct {
	ReasonCode    MessageRejectionReason
	MessageHeader uint8
}

// NewMessageRejectionMessage creates a new MessageRejectionMessage with given fields.
func NewMessageRejectionMessage(reasonCode MessageRejectionReason, messageHeader uint8) MessageRejectionMessage {
	return MessageRejectionMessage{
		ReasonCode:    reasonCode,
		MessageHeader: messageHeader,
	}
}

func (mrm MessageRejectionMessage) String() string {
	return fmt.Sprintf(
		"MSG_REJECT(Reason Code=%v, Rejected Message Header=%d)",
		mrm.ReasonCode, mrm.MessageHeader)
}

func (mrm MessageRejectionMessage) Marshal(w io.Writer) error {
	var fields = []interface{}{MSG_REJECT, mrm}

	for _, field := range fields {
		if err := binary.Write(w, binary.BigEndian, field); err != nil {
			return err
		}
	}

	return nil
}

func (mrm *MessageRejectionMessage) Unmarshal(r io.Reader) error {
	var messageHeader uint8
	if err := binary.Read(r, binary.BigEndian, &messageHeader); err != nil {
		return err
	} else if messageHeader != MSG_REJECT {
		return fmt.Errorf(
			"MSG_REJECT's Message Header is wrong: %d instead of %d",
			messageHeader, MSG_REJECT)
	}

	if err := binary.Read(r, binary.BigEndian, mrm); err != nil {
		return err
	}

	if !mrm.ReasonCode.IsValid() {
		return fmt.Errorf("MSG_REJECT's Reason Code %x is invalid", mrm.ReasonCode)
	}

	return nil
}
