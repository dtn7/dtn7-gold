package tcpcl

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

	// messageRejectionReason_INVALID is a bit field of all invalid MessageRejectionMessages.
	messageRejectionReason_INVALID = 0xFC
)

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

// MarshalBinary encodes this MessageRejectionMessage into its binary form.
func (mrm MessageRejectionMessage) MarshalBinary() (data []byte, err error) {
	var buf = new(bytes.Buffer)
	var fields = []interface{}{
		MSG_REJECT,
		mrm.ReasonCode,
		mrm.MessageHeader}

	for _, field := range fields {
		if binErr := binary.Write(buf, binary.BigEndian, field); binErr != nil {
			err = binErr
			return
		}
	}

	data = buf.Bytes()
	return
}

// UnmarshalBinary decodes a MessageRejectionMessage from its binary form.
func (mrm *MessageRejectionMessage) UnmarshalBinary(data []byte) error {
	var buf = bytes.NewReader(data)

	var messageHeader uint8
	if err := binary.Read(buf, binary.BigEndian, &messageHeader); err != nil {
		return err
	} else if messageHeader != MSG_REJECT {
		return fmt.Errorf("MSG_REJECT's Message Header is wrong: %d instead of %d", messageHeader, MSG_REJECT)
	}

	var fields = []interface{}{&mrm.ReasonCode, &mrm.MessageHeader}

	for _, field := range fields {
		if err := binary.Read(buf, binary.BigEndian, field); err != nil {
			return err
		}
	}

	if mrm.ReasonCode&messageRejectionReason_INVALID != 0 {
		return fmt.Errorf("MSG_REJECT's Reason Code %x is invalid", mrm.ReasonCode)
	}

	return nil
}
