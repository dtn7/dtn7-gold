// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
)

// Fragment is a part of a Transmission. Multiple Fragments represent an entire Transmission.
//
// For identification, a tuple consisting of a transmission ID, a sequence number, a start bit, an end bit,
// and a fail bit is used. Because of memory reasons, the whole header has a size of two bytes. First the
// transmission ID with a length of one byte is given, followed by the two five long sequence number and
// the three flags, the start, end and fail bits.
//
// The transmission ID is taken from the respective Transmission. If this Fragment is the first of a
// Transmission, the start bit is set to one. The same applies to the end bit for the last Fragment.
//
// The five bit sequence number represents a simple binary counter, which is incremented for each Fragment.
// Missing Fragments can be detected this way
//
// The header is followed by the payload. The total data can be as long as the particular MTU allows.
//
//     0   1   2   3   4   5   6   7
//   +---+---+---+---+---+---+---+---+
//   |Transmission ID                |
//   +---+---+---+---+---+---+---+---+
//   |Seq. No            |SB |EB |FB |
//   +---+---+---+---+---+---+---+---+
//   |                               |
//   +            Payload            +
//   |                               |
//
type Fragment struct {
	transmissionId byte
	identifier     byte
	Payload        []byte
}

// fragmentIdentifierSize is the additional size for each Fragment's header.
const fragmentIdentifierSize int = 2

// NewFragment creates a new Fragment based on the given arguments.
func NewFragment(transmissionId, sequenceNo byte, start, end, fail bool, payload []byte) Fragment {
	var identifier byte = 0x00
	identifier |= (sequenceNo & 0x1F) << 3
	if start {
		identifier |= 0x04
	}
	if end {
		identifier |= 0x02
	}
	if fail {
		identifier |= 0x01
	}

	return Fragment{
		transmissionId: transmissionId,
		identifier:     identifier,
		Payload:        payload,
	}
}

func ParseFragment(data []byte) (f Fragment, err error) {
	if len(data) < fragmentIdentifierSize {
		err = fmt.Errorf("byte array has %d bytes, but needs to be at least %d", len(data), fragmentIdentifierSize)
		return
	}

	f.transmissionId = data[0]
	f.identifier = data[1]
	f.Payload = data[2:]

	return
}

func (f Fragment) String() string {
	return fmt.Sprintf("Fragment(TID: %d, Seq.No: %d, SB: %t, EB: %t, FB: %t)",
		f.TransmissionID(), f.SequenceNumber(), f.StartBit(), f.EndBit(), f.FailBit())
}

// TransmissionID returns the four bit transmission ID.
func (f Fragment) TransmissionID() byte {
	return f.transmissionId
}

// SequenceNumber returns the two bit sequence number.
func (f Fragment) SequenceNumber() byte {
	return f.identifier >> 3 & 0x1F
}

// StartBit checks if the start bit is set.
func (f Fragment) StartBit() bool {
	return f.identifier&0x04 != 0
}

// EndBit checks if the end bit is set.
func (f Fragment) EndBit() bool {
	return f.identifier&0x02 != 0
}

// FailBit checks if the fail bit is set.
func (f Fragment) FailBit() bool {
	return f.identifier&0x01 != 0
}

// Bytes creates a byte array for this Fragment.
func (f Fragment) Bytes() []byte {
	buf := new(bytes.Buffer)
	for _, v := range []interface{}{f.transmissionId, f.identifier, f.Payload} {
		_ = binary.Write(buf, binary.LittleEndian, v)
	}

	return buf.Bytes()
}

// ReportFailure creates a failure Fragment based on the current one.
func (f Fragment) ReportFailure() Fragment {
	return NewFragment(f.TransmissionID(), f.SequenceNumber(), false, false, true, []byte{})
}

// randomTransmissionId creates a pseudorandom transmission ID.
func randomTransmissionId() byte {
	randInt, _ := rand.Int(rand.Reader, big.NewInt(256))
	return byte(randInt.Int64())
}

// nextTransmissionId returns the succeeding transmission ID.
func nextTransmissionId(tid byte) byte {
	return tid + 1
}

// nextSequenceNumber returns the succeeding sequence number.
func nextSequenceNumber(seq byte) byte {
	return (seq + 1) % 16
}
