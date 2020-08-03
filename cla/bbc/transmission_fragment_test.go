// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

import (
	"bytes"
	"reflect"
	"testing"
)

func TestFragmentBitMask(t *testing.T) {
	tests := []struct {
		mask           byte
		transmissionId byte
		sequenceNo     byte
		start          bool
		end            bool
		fail           bool
	}{
		{0x04, 0x0A, 0x00, true, false, false},
		{0x08, 0x01, 0x01, false, false, false},
		{0x1A, 0x00, 0x03, false, true, false},
		{0x05, 0x0A, 0x00, true, false, true},
		{0x09, 0x01, 0x01, false, false, true},
		{0x1B, 0x00, 0x03, false, true, true},
	}

	for _, test := range tests {
		f1 := NewFragment(test.transmissionId, test.sequenceNo, test.start, test.end, test.fail, nil)
		if f1.identifier != test.mask {
			t.Fatalf("Fragment %v has identifier mask of %x instead of %x", test, f1.identifier, test.mask)
		}

		f2 := Fragment{transmissionId: test.transmissionId, identifier: test.mask}
		if tid := f2.TransmissionID(); tid != test.transmissionId {
			t.Fatalf("Fragment %v has transmission ID of %x instead of %x", test, tid, test.transmissionId)
		}
		if s := f2.SequenceNumber(); s != test.sequenceNo {
			t.Fatalf("Fragment %v has sequence no of %x instead of %x", test, s, test.sequenceNo)
		}
		if b := f2.StartBit(); b != test.start {
			t.Fatalf("Fragment %v has start bit of %t instead of %t", test, b, test.start)
		}
		if b := f2.EndBit(); b != test.end {
			t.Fatalf("Fragment %v has end bit of %t instead of %t", test, b, test.end)
		}
		if b := f2.FailBit(); b != test.fail {
			t.Fatalf("Fragment %v has fail bit of %t instead of %t", test, b, test.fail)
		}
	}
}

func TestFragmentAllIdentifierCombinations(t *testing.T) {
	for i := 0x00; i <= 0xFF; i++ {
		mask := byte(i)

		sequenceNo := mask >> 3 & 0x1F
		start := mask&0x04 != 0
		end := mask&0x02 != 0
		fail := mask&0x01 != 0

		f := Fragment{identifier: mask}
		if s := f.SequenceNumber(); s != sequenceNo {
			t.Fatalf("Fragment %x has sequence no of %x instead of %x", mask, s, sequenceNo)
		}
		if b := f.StartBit(); b != start {
			t.Fatalf("Fragment %x has start bit of %t instead of %t", mask, b, start)
		}
		if b := f.EndBit(); b != end {
			t.Fatalf("Fragment %x has end bit of %t instead of %t", mask, b, end)
		}
		if b := f.FailBit(); b != fail {
			t.Fatalf("Fragment %x has fail bit of %t instead of %t", mask, b, fail)
		}
	}
}

func TestNextSequenceNumber(t *testing.T) {
	tests := []struct {
		seq  byte
		succ byte
	}{
		{0, 1},
		{1, 2},
		{14, 15},
		{15, 0},
	}

	for _, test := range tests {
		if succ := nextSequenceNumber(test.seq); succ != test.succ {
			t.Fatalf("Succeeding sequence number of %d is %d, not %d", test.seq, succ, test.succ)
		}
	}
}

func TestNextTransmissionId(t *testing.T) {
	tests := []struct {
		tid  byte
		succ byte
	}{
		{0, 1},
		{1, 2},
		{254, 255},
		{255, 0},
	}

	for _, test := range tests {
		if succ := nextTransmissionId(test.tid); succ != test.succ {
			t.Fatalf("Succeeding transmission ID of %d is %d, not %d", test.tid, succ, test.succ)
		}
	}
}

func TestFragmentBytes(t *testing.T) {
	tests := []struct {
		seq []byte
		f   Fragment
	}{
		{[]byte{0xC0, 0xFF, 0xEE}, Fragment{0xC0, 0xFF, []byte{0xEE}}},
		{[]byte{0xAC, 0xAB}, Fragment{0xAC, 0xAB, []byte{}}},
	}

	for _, test := range tests {
		if f, err := ParseFragment(test.seq); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(f, test.f) {
			t.Fatalf("Fragments do not match: %v != %v", f, test.f)
		}

		if blob := test.f.Bytes(); !bytes.Equal(blob, test.seq) {
			t.Fatalf("Bytes do not match: %x != %x", blob, test.seq)
		}
	}
}
