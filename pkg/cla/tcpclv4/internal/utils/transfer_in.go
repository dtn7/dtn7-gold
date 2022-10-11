// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"bytes"
	"fmt"
	"io"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

// IncomingTransfer represents an incoming Bundle Transfer for the TCPCLv4.
type IncomingTransfer struct {
	Id uint64

	endFlag bool
	buf     *bytes.Buffer
}

// NewIncomingTransfer creates a new IncomingTransfer for the given Transfer iD.
func NewIncomingTransfer(id uint64) *IncomingTransfer {
	return &IncomingTransfer{
		Id:  id,
		buf: new(bytes.Buffer),
	}
}

func (t IncomingTransfer) String() string {
	return fmt.Sprintf("INCOMING_TRANSFER(%d)", t.Id)
}

// IsFinished indicates if this Transfer is finished.
func (t IncomingTransfer) IsFinished() bool {
	return t.endFlag
}

// NextSegment reads data from a XFER_SEGMENT and returns a XFER_ACK or an error.
func (t *IncomingTransfer) NextSegment(dtm *msgs.DataTransmissionMessage) (dam *msgs.DataAcknowledgementMessage, err error) {
	if t.IsFinished() {
		err = fmt.Errorf("transfer has already received an end flag")
		return
	}

	if t.Id != dtm.TransferId {
		err = fmt.Errorf("XFER_SEGMENT's Transfer iD %d mismatches %d", dtm.TransferId, t.Id)
		return
	}

	if n, dtmErr := t.buf.Write(dtm.Data); dtmErr != nil && dtmErr != io.EOF {
		err = dtmErr
		return
	} else if n != len(dtm.Data) {
		err = fmt.Errorf("expected %d bytes instead of  %d", len(dtm.Data), n)
		return
	}

	if dtm.Flags&msgs.SegmentEnd != 0 {
		t.endFlag = true
	}

	dam = msgs.NewDataAcknowledgementMessage(dtm.Flags, dtm.TransferId, uint64(t.buf.Len()))
	return
}

// ToBundle returns the Bundle for a finished Transfer.
func (t *IncomingTransfer) ToBundle() (bndl bpv7.Bundle, err error) {
	if !t.IsFinished() {
		err = fmt.Errorf("transfer has not been finished")
		return
	}

	err = bndl.UnmarshalCbor(t.buf)
	return
}
