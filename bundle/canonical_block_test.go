package bundle

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func TestNewCanonicalBlock(t *testing.T) {
	b := NewPayloadBlock(
		ReplicateBlock, []byte("hello world"))

	if b.HasCRC() {
		t.Errorf("Canonical Block (Payload Block) has CRC: %v", b)
	}

	b.CRCType = CRC32
	if !b.HasCRC() {
		t.Errorf("Canonical Block (Payload Block) has no CRC: %v", b)
	}
}

func TestCanonicalBlockCbor(t *testing.T) {
	ep, _ := NewEndpointID("dtn:foo/bar")

	tests := []struct {
		cb1 CanonicalBlock
		len int
	}{
		// Generic CanonicalBlock: No CRC
		{CanonicalBlock{1, 0, 0, CRCNo, []byte("hello world"), nil}, 5},
		// Generic CanonicalBlock: CRC
		{CanonicalBlock{1, 0, 0, CRC16, []byte("hello world"), nil}, 6},
		// Payload block
		{NewPayloadBlock(0, []byte("test")), 5},
		// Previous Node block (dtn:none)
		{NewPreviousNodeBlock(23, 0, DtnNone()), 5},
		// Previous Node block (dtn:foo/bar)
		{NewPreviousNodeBlock(23, 0, ep), 5},
		// Bundle Age block
		{NewBundleAgeBlock(23, 0, 100000), 5},
		// Hop Count block
		{NewHopCountBlock(23, 0, NewHopCount(100)), 5},
	}

	for _, test := range tests {
		if test.cb1.HasCRC() {
			test.cb1.CalculateCRC()
		}

		buff := new(bytes.Buffer)
		if err := cboring.Marshal(&test.cb1, buff); err != nil {
			t.Fatal(err)
		}

		cb2 := CanonicalBlock{}
		if err := cboring.Unmarshal(&cb2, buff); err != nil {
			t.Errorf("CBOR decoding failed for %v: %v", test, err)
		}

		if !reflect.DeepEqual(test.cb1, cb2) {
			t.Fatalf("CanonicalBlocks differ:\n%v\n%v", test.cb1, cb2)
		}
	}
}

func TestCanonicalBlockCheckValid(t *testing.T) {
	tests := []struct {
		cb    CanonicalBlock
		valid bool
	}{
		// Payload block with a block number != zero
		{CanonicalBlock{PayloadBlock, 23, 0, CRCNo, nil, nil}, false},
		{CanonicalBlock{PayloadBlock, 0, 0, CRCNo, nil, nil}, true},

		// Reserved bits in block control flags
		{CanonicalBlock{PayloadBlock, 0, 0x80, CRCNo, nil, nil}, false},

		// Illegal EndpointID in Previous Node Block
		{NewPreviousNodeBlock(23, 0,
			EndpointID{SchemeName: endpointURISchemeIPN, SchemeSpecificPart: [2]uint64{0, 0}}),
			false},
		{NewPreviousNodeBlock(23, 0, DtnNone()), true},

		// Reserved block type
		{CanonicalBlock{191, 0, 0, CRCNo, nil, nil}, false},
		{CanonicalBlock{192, 0, 0, CRCNo, nil, nil}, true},
		{CanonicalBlock{255, 0, 0, CRCNo, nil, nil}, true},
		{CanonicalBlock{256, 0, 0, CRCNo, nil, nil}, false},
	}

	for _, test := range tests {
		if err := test.cb.checkValid(); (err == nil) != test.valid {
			t.Errorf("CanonicalBlock validation failed: %v resulted in %v",
				test.cb, err)
		}
	}
}

func TestHopCount(t *testing.T) {
	tests := []struct {
		hc                     HopCount
		exceeded               bool
		exceededAfterIncrement bool
	}{
		{NewHopCount(10), false, false},
		{NewHopCount(1), false, false},
		{NewHopCount(0), false, true},
		{HopCount{Limit: 23, Count: 20}, false, false},
		{HopCount{Limit: 23, Count: 22}, false, false},
		{HopCount{Limit: 23, Count: 23}, false, true},
	}

	for _, test := range tests {
		if state := test.hc.IsExceeded(); state != test.exceeded {
			t.Errorf("Hop count block's %v state is wrong: expected %t, real %t",
				test.hc, test.exceeded, state)
		}

		if state := test.hc.Increment(); state != test.exceededAfterIncrement {
			t.Errorf("Hop count block's state %v is wrong after increment: expected %t, real %t",
				test.hc, test.exceededAfterIncrement, state)
		}
	}
}
