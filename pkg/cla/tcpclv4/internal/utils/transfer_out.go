// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"bufio"
	"fmt"
	"io"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

// OutgoingTransfer represents an outgoing Bundle Transfer for the TCPCLv4.
type OutgoingTransfer struct {
	Id uint64

	startFlag  bool
	dataStream io.Reader
}

// NewOutgoingTransfer creates a new OutgoingTransfer for data written into the returned Writer.
func NewOutgoingTransfer(id uint64) (t *OutgoingTransfer, w io.Writer) {
	r, w := io.Pipe()
	t = &OutgoingTransfer{
		Id:         id,
		startFlag:  true,
		dataStream: r,
	}

	return
}

func (t OutgoingTransfer) String() string {
	return fmt.Sprintf("OUTGOING_TRANSFER(%d)", t.Id)
}

// NewBundleOutgoingTransfer creates a new OutgoingTransfer for a Bundle.
func NewBundleOutgoingTransfer(id uint64, b bpv7.Bundle) *OutgoingTransfer {
	var t, w = NewOutgoingTransfer(id)

	go func(w *io.PipeWriter) {
		bw := bufio.NewWriter(w)

		_ = b.MarshalCbor(bw)
		_ = bw.Flush()
		_ = w.Close()
	}(w.(*io.PipeWriter))

	return t
}

// NextSegment creates the next XFER_SEGMENT for the given MTU or an EOF in case of a finished Writer.
func (t *OutgoingTransfer) NextSegment(mtu uint64) (dtm *msgs.DataTransmissionMessage, err error) {
	var segFlags msgs.SegmentFlags

	if t.startFlag {
		t.startFlag = false
		segFlags |= msgs.SegmentStart
	}

	var buf = make([]byte, mtu)
	if n, rErr := io.ReadFull(t.dataStream, buf); rErr == io.ErrUnexpectedEOF {
		buf = buf[:n]
		segFlags |= msgs.SegmentEnd
	} else if rErr != nil {
		err = rErr
		return
	}

	dtm = msgs.NewDataTransmissionMessage(segFlags, t.Id, buf)
	return
}
