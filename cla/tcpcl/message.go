package tcpcl

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

// Message describes all kind of TCPCL messages, which have their serialization
// and deserialization in common.
type Message interface {
	Marshal(w io.Writer) error
	Unmarshal(r io.Reader) error
}

// messages maps the different TCPCL message type codes to an example instance
// of their type.
var messages = map[uint8]Message{
	SESS_INIT:    &SessionInitMessage{},
	SESS_TERM:    &SessionTerminationMessage{},
	XFER_SEGMENT: &DataTransmissionMessage{},
	XFER_ACK:     &DataAcknowledgementMessage{},
	XFER_REFUSE:  &TransferRefusalMessage{},
	KEEPALIVE:    &KeepaliveMessage{},
	MSG_REJECT:   &MessageRejectionMessage{},
}

// NewMessage creates a new Message type for a given type code.
func NewMessage(typeCode uint8) (msg Message, err error) {
	msgType, exists := messages[typeCode]
	if !exists {
		err = fmt.Errorf("No TCPCL Message registered for type code %d", typeCode)
		return
	}

	msgElem := reflect.TypeOf(msgType).Elem()
	msg = reflect.New(msgElem).Interface().(Message)
	return
}

// ReadMessage parses the next TCPCL message from the Reader.
func ReadMessage(r io.Reader) (msg Message, err error) {
	msgTypeBytes := make([]byte, 1)
	if n, msgTypeErr := r.Read(msgTypeBytes); msgTypeErr != nil {
		err = msgTypeErr
		return
	} else if n != 1 {
		err = fmt.Errorf("Expected one byte, got %d bytes", n)
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
