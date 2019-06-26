package bundle

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func setupPrimaryBlock() PrimaryBlock {
	bcf := StatusRequestDeletion |
		StatusRequestDelivery |
		MustNotFragmented

	destination, _ := NewEndpointID("dtn:foobar")
	source, _ := NewEndpointID("dtn:me")

	creationTimestamp := NewCreationTimestamp(DtnTimeEpoch, 0)
	lifetime := uint64(10 * 60 * 1000)

	return NewPrimaryBlock(bcf, destination, source, creationTimestamp, lifetime)
}

func TestNewPrimaryBlock(t *testing.T) {
	pb := setupPrimaryBlock()

	if pb.HasCRC() {
		t.Error("Primary Block has no CRC, but says so")
	}

	if pb.HasFragmentation() {
		t.Error("Primary Block has no fragmentation, but says so")
	}
}

func TestPrimaryBlockCRC(t *testing.T) {
	pb := setupPrimaryBlock()
	pb.CRCType = CRC16

	if !pb.HasCRC() {
		t.Error("Primary Block should need a CRC")
	}
}

func TestPrimaryBlockFragmentation(t *testing.T) {
	pb := setupPrimaryBlock()
	pb.BundleControlFlags = IsFragment

	if !pb.HasFragmentation() {
		t.Error("Primary Block should be fragmented")
	}
}

func TestPrimaryBlockCbor(t *testing.T) {
	ep, _ := NewEndpointID("dtn:test")
	ts := NewCreationTimestamp(DtnTimeEpoch, 23)

	tests := []struct {
		pb1 PrimaryBlock
		len int
	}{
		// No CRC, No Fragmentation
		{PrimaryBlock{7, 0, CRCNo, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 8},
		// CRC, No Fragmentation
		{PrimaryBlock{7, 0, CRC16, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 9},
		// No CRC, Fragmentation
		{PrimaryBlock{7, IsFragment, CRCNo, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 10},
		// CRC, Fragmentation
		{PrimaryBlock{7, IsFragment, CRC16, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 11},
	}

	for _, test := range tests {
		buff := new(bytes.Buffer)
		if err := cboring.Marshal(&test.pb1, buff); err != nil {
			t.Fatal(err)
		}

		var pb2 PrimaryBlock
		if err := cboring.Unmarshal(&pb2, buff); err != nil {
			t.Fatalf("CBOR decoding failed: %v", err)
		}

		if !reflect.DeepEqual(test.pb1, pb2) {
			t.Fatalf("PrimaryBlocks differ:\n%v\n%v", test.pb1, pb2)
		}
	}
}

func TestPrimaryBlockCheckValid(t *testing.T) {
	tests := []struct {
		pb    PrimaryBlock
		valid bool
	}{
		// Wrong version
		{PrimaryBlock{
			23, MustNotFragmented, CRCNo, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil}, false},
		{PrimaryBlock{
			7, MustNotFragmented, CRCNo, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil}, true},

		// Reserved bits in bundle control flags
		{PrimaryBlock{
			7, 0xFF00, CRCNo, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil}, false},

		// Illegal EndpointID
		{PrimaryBlock{
			7, 0, CRCNo,
			EndpointID{SchemeName: endpointURISchemeIPN, SchemeSpecificPart: [2]uint64{0, 0}},
			DtnNone(), DtnNone(), NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil},
			false},

		// Everything from above
		{PrimaryBlock{
			23, 0xFF00, CRCNo,
			EndpointID{SchemeName: endpointURISchemeIPN, SchemeSpecificPart: [2]uint64{0, 0}},
			DtnNone(), DtnNone(), NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil},
			false},

		// Source Node = dtn:none, "Must Not Be Fragmented"-flag is zero
		{PrimaryBlock{
			7, 0, CRCNo, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil}, false},

		// Source Node = dtn:none, a status flag is one
		{PrimaryBlock{
			7, MustNotFragmented | StatusRequestReception,
			CRCNo, DtnNone(), DtnNone(), DtnNone(), NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil},
			false},
	}

	for _, test := range tests {
		if err := test.pb.checkValid(); (err == nil) != test.valid {
			t.Errorf("PrimaryBlock validation failed: %v resulted in %v",
				test.pb, err)
		}
	}
}
