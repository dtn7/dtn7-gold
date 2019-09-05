package tcpcl

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// XFER_ACK is the Message Header code for a Data Acknowledgement Message.
const XFER_ACK uint8 = 0x02

// DataAcknowledgementMessage is the XFER_ACK message for data acknowledgements.
type DataAcknowledgementMessage struct {
	Flags      SegmentFlags
	TransferId uint64
	AckLen     uint64
}

// NewDataAcknowledgementMessage creates a new DataAcknowledgementMessage with given fields.
func NewDataAcknowledgementMessage(flags SegmentFlags, tid, acklen uint64) DataAcknowledgementMessage {
	return DataAcknowledgementMessage{
		Flags:      flags,
		TransferId: tid,
		AckLen:     acklen,
	}
}

func (dam DataAcknowledgementMessage) String() string {
	return fmt.Sprintf(
		"XFER_ACK(Message Flags=%v, Transfer ID=%d, Acknowledged length=%d)",
		dam.Flags, dam.TransferId, dam.AckLen)
}

// MarshalBinary encodes this DataAcknowledgementMessage into its binary form.
func (dam DataAcknowledgementMessage) MarshalBinary() (data []byte, err error) {
	var buf = new(bytes.Buffer)
	var fields = []interface{}{
		XFER_ACK,
		dam.Flags,
		dam.TransferId,
		dam.AckLen}

	for _, field := range fields {
		if binErr := binary.Write(buf, binary.BigEndian, field); binErr != nil {
			err = binErr
			return
		}
	}

	data = buf.Bytes()
	return
}

// UnmarshalBinary decodes a DataAcknowledgementMessage from its binary form.
func (dam *DataAcknowledgementMessage) UnmarshalBinary(data []byte) error {
	var buf = bytes.NewReader(data)

	var messageHeader uint8
	if err := binary.Read(buf, binary.BigEndian, &messageHeader); err != nil {
		return err
	} else if messageHeader != XFER_ACK {
		return fmt.Errorf("XFER_ACK's Message Header is wrong: %d instead of %d", messageHeader, XFER_ACK)
	}

	var fields = []interface{}{&dam.Flags, &dam.TransferId, &dam.AckLen}

	for _, field := range fields {
		if err := binary.Read(buf, binary.BigEndian, field); err != nil {
			return err
		}
	}

	return nil
}
