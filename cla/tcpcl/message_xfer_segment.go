package tcpcl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

// SegmentFlags are an one-octet field of single-bit flags for a XFER_SEGMENT.
type SegmentFlags uint8

const (
	// SegmentEnd indicates that this segment is the last of the transfer.
	SegmentEnd SegmentFlags = 0x01

	// SegmentStart indicates that this segment is the first of the transfer.
	SegmentStart SegmentFlags = 0x02

	// segmentFlags_INVALID is a bit field of all invalid ContactFlags.
	segmentFlags_INVALID SegmentFlags = 0xFC
)

func (sf SegmentFlags) String() string {
	var flags []string

	if sf&SegmentEnd != 0 {
		flags = append(flags, "END")
	}
	if sf&SegmentStart != 0 {
		flags = append(flags, "START")
	}

	return strings.Join(flags, ",")
}

// XFER_SEGMENT is the Message Header code for a Data Transmission Message.
const XFER_SEGMENT uint8 = 0x01

// DataTransmissionMessage is the XFER_SEGMENT message for data transmission.
type DataTransmissionMessage struct {
	Flags      SegmentFlags
	TransferId uint64
	Data       []byte

	// TODO: Transfer Extension Items
}

// NewDataTransmissionMessage creates a new DataTransmissionMessage with given fields.
func NewDataTransmissionMessage(flags SegmentFlags, tid uint64, data []byte) DataTransmissionMessage {
	return DataTransmissionMessage{
		Flags:      flags,
		TransferId: tid,
		Data:       data,
	}
}

func (dtm DataTransmissionMessage) String() string {
	return fmt.Sprintf(
		"XFER_SEGMENT(Message Flags=%v, Transfer ID=%d, Data=%x)",
		dtm.Flags, dtm.TransferId, dtm.Data)
}

// MarshalBinary encodes this DataTransmissionMessage into its binary form.
func (dtm DataTransmissionMessage) MarshalBinary() (data []byte, err error) {
	var buf = new(bytes.Buffer)
	var fields = []interface{}{
		XFER_SEGMENT,
		dtm.Flags,
		dtm.TransferId,
		uint32(0), // TODO: Transfer Extension Items
		uint64(len(dtm.Data))}

	for _, field := range fields {
		if binErr := binary.Write(buf, binary.BigEndian, field); binErr != nil {
			err = binErr
			return
		}
	}

	if n, _ := buf.Write(dtm.Data); n != len(dtm.Data) {
		err = fmt.Errorf("XFER_SEGMENT Data length is %d, but only wrote %d bytes", len(dtm.Data), n)
		return
	}

	data = buf.Bytes()
	return
}

// UnmarshalBinary decodes a DataTransmissionMessage from its binary form.
func (dtm *DataTransmissionMessage) UnmarshalBinary(data []byte) error {
	var buf = bytes.NewReader(data)

	var messageHeader uint8
	if err := binary.Read(buf, binary.BigEndian, &messageHeader); err != nil {
		return err
	} else if messageHeader != XFER_SEGMENT {
		return fmt.Errorf("XFER_SEGMENT's Message Header is wrong: %d instead of %d", messageHeader, XFER_SEGMENT)
	}

	var transferExtLen uint32
	var fields = []interface{}{&dtm.Flags, &dtm.TransferId, &transferExtLen}

	for _, field := range fields {
		if err := binary.Read(buf, binary.BigEndian, field); err != nil {
			return err
		}
	}

	// TODO: Transfer Extension Items
	if transferExtLen > 0 {
		transferExtBuff := make([]byte, transferExtLen)

		if n, err := buf.Read(transferExtBuff); err != nil {
			return err
		} else if uint32(n) != transferExtLen {
			return fmt.Errorf(
				"XFER_SEGMENT's Transfer Extension Length differs: expected %d and got %d",
				transferExtLen, n)
		}
	}

	var dataLen uint64
	if err := binary.Read(buf, binary.BigEndian, &dataLen); err != nil {
		return err
	} else if dataLen > 0 {
		dataBuff := make([]byte, dataLen)

		if n, err := buf.Read(dataBuff); err != nil {
			return err
		} else if uint64(n) != dataLen {
			return fmt.Errorf(
				"XFER_SEGMENT's Data length differs: expected %d and got %d",
				dataLen, n)
		} else {
			dtm.Data = dataBuff
		}
	}

	return nil
}
