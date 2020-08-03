// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

import (
	"bytes"
	"fmt"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/ulikunitz/xz"
)

// Transmission allows the transmission of data, for example Bundles, between several endpoints.
//
// Due to the maximum transmission unit (MTU), most Transmissions are likely to be fragmented. This results
// in Fragments. In order to keep the Fragments apart and to associate them with their Transmission, a
// transmission ID exists. Since the data size is very limited, the transmission ID is reduced to a byte.
// In order to avoid collisions, the transmission ID should be chosen as randomly or cleverly as possible.
//
// In the following, a distinction is made between incoming and outgoing Transmissions: IncomingTransmission
// and OutgoingTransmission.
type Transmission struct {
	TransmissionID byte
	Payload        []byte

	finished bool
}

// IsFinished indicates a finished Transmission.
func (t Transmission) IsFinished() bool {
	return t.finished
}

func (t Transmission) String() string {
	return fmt.Sprintf("Transmission(TID: %d, Finished: %t)", t.TransmissionID, t.IsFinished())
}

// IncomingTransmission are the incoming Transmissions from external sources.
type IncomingTransmission struct {
	Transmission
	prevSequenceNo byte
}

// NewIncomingTransmission creates a new IncomingTransmission from a Fragment with the start bit set.
func NewIncomingTransmission(f Fragment) (t *IncomingTransmission, err error) {
	if !f.StartBit() {
		err = fmt.Errorf("Fragment has no start bit")
		return
	}

	t = &IncomingTransmission{
		Transmission: Transmission{
			TransmissionID: f.TransmissionID(),
			Payload:        f.Payload,
			finished:       f.EndBit(),
		},
		prevSequenceNo: f.SequenceNumber(),
	}
	return
}

// ReadFragment processes the next Fragment for this IncomingTransmission.
func (t *IncomingTransmission) ReadFragment(f Fragment) (finished bool, err error) {
	if t.IsFinished() {
		err = fmt.Errorf("Transmission was already marked as finished")
		return
	}

	if f.TransmissionID() != t.TransmissionID {
		err = fmt.Errorf("transmission ID mismatches: Fragment got %x, expected %x",
			f.TransmissionID(), t.TransmissionID)
		return
	}

	if expected := nextSequenceNumber(t.prevSequenceNo); f.SequenceNumber() != expected {
		err = fmt.Errorf("expected sequence number of %x, got %x", expected, f.SequenceNumber())
		return
	}

	if f.StartBit() {
		err = fmt.Errorf("Fragment has start bit, but previous data was already read")
		return
	}

	t.Payload = append(t.Payload, f.Payload...)
	t.finished = f.EndBit()
	t.prevSequenceNo = f.SequenceNumber()

	finished = t.IsFinished()
	return
}

func (t *IncomingTransmission) Bundle() (bndl bundle.Bundle, err error) {
	if !t.IsFinished() {
		err = fmt.Errorf("Transmission is not finished yet")
		return
	}

	xzR, xzErr := xz.NewReader(bytes.NewBuffer(t.Payload))
	if xzErr != nil {
		err = xzErr
		return
	}

	err = bndl.UnmarshalCbor(xzR)
	return
}

// OutgoingTransmission are the outgoing Transmissions to external sources.
type OutgoingTransmission struct {
	Transmission
	mtu           int
	start         bool
	nextSegmentNo byte
}

// NewOutgoingTransmission creates a new OutgoingTransmission for a Bundle.
func NewOutgoingTransmission(transmissionID byte, bndl bundle.Bundle, mtu int) (t *OutgoingTransmission, err error) {
	var buf bytes.Buffer
	if xzW, xzErr := xz.NewWriter(&buf); xzErr != nil {
		err = xzErr
		return
	} else if err = bndl.WriteBundle(xzW); err != nil {
		return
	} else if err = xzW.Close(); err != nil {
		return
	}

	return newPlainOutgoingTransmission(transmissionID, buf.Bytes(), mtu)
}

func newPlainOutgoingTransmission(transmissionID byte, payload []byte, mtu int) (t *OutgoingTransmission, err error) {
	var fin = false
	if len(payload) == 0 {
		fin = true
	}

	t = &OutgoingTransmission{
		Transmission: Transmission{
			TransmissionID: transmissionID,
			Payload:        payload,
			finished:       fin,
		},
		mtu:           mtu - fragmentIdentifierSize,
		start:         true,
		nextSegmentNo: 0,
	}
	return
}

// WriteFragment creates the next Fragment for an OutgoingTransmission.
func (t *OutgoingTransmission) WriteFragment() (f Fragment, finished bool, err error) {
	if t.IsFinished() {
		err = fmt.Errorf("Transmission was already marked as finished")
		return
	}

	var nextPayload []byte
	if len(t.Payload) <= t.mtu {
		nextPayload = t.Payload
		t.Payload = nil
		t.finished = true
	} else {
		nextPayload = t.Payload[:t.mtu]
		t.Payload = t.Payload[t.mtu:]
	}

	t.nextSegmentNo = nextSequenceNumber(t.nextSegmentNo)
	f = NewFragment(t.TransmissionID, t.nextSegmentNo, t.start, t.finished, false, nextPayload)
	t.start = false

	finished = t.IsFinished()
	return
}
