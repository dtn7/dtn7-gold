// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package msgs

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

// Message describes all kind of TCPCLv4 messages, which have their serialization and deserialization in common.
type Message interface {
	Marshal(w io.Writer) error
	Unmarshal(r io.Reader) error
}

// messages maps the different TCPCLv4 message type codes to an example instance of their type.
var messages = map[uint8]Message{
	SESS_INIT:    &SessionInitMessage{},
	SESS_TERM:    &SessionTerminationMessage{},
	XFER_SEGMENT: &DataTransmissionMessage{},
	XFER_ACK:     &DataAcknowledgementMessage{},
	XFER_REFUSE:  &TransferRefusalMessage{},
	KEEPALIVE:    &KeepaliveMessage{},
	MSG_REJECT:   &MessageRejectionMessage{},

	// 0x64 is an ASCII 'd', which is the start of the ContactHeader's magic, 'dtn!'. Even when the ContactHeader is not
	// a real Message as described in the RFC, this makes parsing a lot easier.
	0x64: &ContactHeader{},
}

// NewMessage creates a new Message type for a given type code.
func NewMessage(typeCode uint8) (msg Message, err error) {
	msgType, exists := messages[typeCode]
	if !exists {
		err = fmt.Errorf("no TCPCLv4 Message registered for type code %x", typeCode)
		return
	}

	msgElem := reflect.TypeOf(msgType).Elem()
	msg = reflect.New(msgElem).Interface().(Message)
	return
}

// ReadMessage parses the next TCPCLv4 message from the Reader.
func ReadMessage(r io.Reader) (msg Message, err error) {
	msgTypeBytes := make([]byte, 1)
	if _, msgTypeErr := io.ReadFull(r, msgTypeBytes); msgTypeErr != nil {
		err = msgTypeErr
		return
	}

	msg, msgErr := NewMessage(msgTypeBytes[0])
	if msgErr != nil {
		err = msgErr
		return
	}

	mr := io.MultiReader(bytes.NewBuffer(msgTypeBytes), r)

	err = msg.Unmarshal(mr)
	return
}
