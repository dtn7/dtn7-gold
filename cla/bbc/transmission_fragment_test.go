package bbc

import (
	"bytes"
	"reflect"
	"testing"
)

func TestFragmentBitMask(t *testing.T) {
	tests := []struct {
		mask           byte
		transmissionID byte
		sequenceNo     byte
		start          bool
		end            bool
	}{
		{0xA2, 0x0A, 0x00, true, false},
		{0x14, 0x01, 0x01, false, false},
		{0x0D, 0x00, 0x03, false, true},
	}

	for _, test := range tests {
		f1 := NewFragment(test.transmissionID, test.sequenceNo, test.start, test.end, nil)
		if f1.identifier != test.mask {
			t.Fatalf("Fragment %v has identifier mask of %x instead of %x", test, f1.identifier, test.mask)
		}

		f2 := Fragment{identifier: test.mask}
		if tid := f2.TransmissionID(); tid != test.transmissionID {
			t.Fatalf("Fragment %v has transmission ID of %x instead of %x", test, tid, test.transmissionID)
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
	}
}

func TestFragmentAllCombinations(t *testing.T) {
	for i := 0x00; i <= 0xFF; i++ {
		mask := byte(i)

		transmissionID := mask >> 4 & 0x0F
		sequenceNo := mask >> 2 & 0x03
		start := mask&0x02 != 0
		end := mask&0x01 != 0

		f := Fragment{identifier: mask}
		if tid := f.TransmissionID(); tid != transmissionID {
			t.Fatalf("Fragment %x has transmission ID of %x instead of %x", mask, tid, transmissionID)
		}
		if s := f.SequenceNumber(); s != sequenceNo {
			t.Fatalf("Fragment %x has sequence no of %x instead of %x", mask, s, sequenceNo)
		}
		if b := f.StartBit(); b != start {
			t.Fatalf("Fragment %x has start bit of %t instead of %t", mask, b, start)
		}
		if b := f.EndBit(); b != end {
			t.Fatalf("Fragment %x has end bit of %t instead of %t", mask, b, end)
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
		{2, 3},
		{3, 0},
	}

	for _, test := range tests {
		if succ := nextSequenceNumber(test.seq); succ != test.succ {
			t.Fatalf("Succeeding sequence number of %x is %x, not %x", test.seq, succ, test.succ)
		}
	}
}

func TestFragmentBytes(t *testing.T) {
	tests := []struct {
		seq []byte
		f   Fragment
	}{
		{[]byte{0xC0, 0xFF, 0xEE}, Fragment{0xC0, []byte{0xFF, 0xEE}}},
		{[]byte{0xAC, 0xAB}, Fragment{0xAc, []byte{0xAB}}},
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
