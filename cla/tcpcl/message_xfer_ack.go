// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"encoding/binary"
	"fmt"
	"io"
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

func (dam DataAcknowledgementMessage) Marshal(w io.Writer) error {
	var fields = []interface{}{XFER_ACK, dam}

	for _, field := range fields {
		if err := binary.Write(w, binary.BigEndian, field); err != nil {
			return err
		}
	}

	return nil
}

func (dam *DataAcknowledgementMessage) Unmarshal(r io.Reader) error {
	var messageHeader uint8
	if err := binary.Read(r, binary.BigEndian, &messageHeader); err != nil {
		return err
	} else if messageHeader != XFER_ACK {
		return fmt.Errorf("XFER_ACK's Message Header is wrong: %d instead of %d", messageHeader, XFER_ACK)
	}

	if err := binary.Read(r, binary.BigEndian, dam); err != nil {
		return err
	}

	return nil
}
