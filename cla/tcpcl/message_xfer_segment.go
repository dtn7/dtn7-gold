// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// SegmentFlags are an one-octet field of single-bit flags for a XFER_SEGMENT.
type SegmentFlags uint8

const (
	// SegmentEnd indicates that this segment is the last of the transfer.
	SegmentEnd SegmentFlags = 0x01

	// SegmentStart indicates that this segment is the first of the transfer.
	SegmentStart SegmentFlags = 0x02
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

func (dtm DataTransmissionMessage) Marshal(w io.Writer) error {
	var fields = []interface{}{
		XFER_SEGMENT,
		dtm.Flags,
		dtm.TransferId,
		uint32(0), // TODO: Transfer Extension Items
		uint64(len(dtm.Data))}

	for _, field := range fields {
		if err := binary.Write(w, binary.BigEndian, field); err != nil {
			return err
		}
	}

	if n, err := w.Write(dtm.Data); err != nil {
		return err
	} else if n != len(dtm.Data) {
		return fmt.Errorf("XFER_SEGMENT Data length is %d, but only wrote %d bytes", len(dtm.Data), n)
	}

	return nil
}

func (dtm *DataTransmissionMessage) Unmarshal(r io.Reader) error {
	var messageHeader uint8
	if err := binary.Read(r, binary.BigEndian, &messageHeader); err != nil {
		return err
	} else if messageHeader != XFER_SEGMENT {
		return fmt.Errorf("XFER_SEGMENT's Message Header is wrong: %d instead of %d", messageHeader, XFER_SEGMENT)
	}

	var transferExtLen uint32
	var fields = []interface{}{&dtm.Flags, &dtm.TransferId, &transferExtLen}

	for _, field := range fields {
		if err := binary.Read(r, binary.BigEndian, field); err != nil {
			return err
		}
	}

	// TODO: Transfer Extension Items
	if transferExtLen > 0 {
		transferExtBuff := make([]byte, transferExtLen)

		if _, err := io.ReadFull(r, transferExtBuff); err != nil {
			return err
		}
	}

	var dataLen uint64
	if err := binary.Read(r, binary.BigEndian, &dataLen); err != nil {
		return err
	} else if dataLen > 0 {
		dtm.Data = make([]byte, dataLen)
		if _, err := io.ReadFull(r, dtm.Data); err != nil {
			return err
		} else if dataLen != uint64(len(dtm.Data)) {
			return fmt.Errorf("XFER_SEGMENT's data length should be %d, got %d bytes", dataLen, len(dtm.Data))
		}
	}

	return nil
}
