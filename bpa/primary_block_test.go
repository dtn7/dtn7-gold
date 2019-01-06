package bpa

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ugorji/go/codec"
)

func setupPrimaryBlock() PrimaryBlock {
	bcf := BndlCFBundleDeletionStatusReportsAreRequested |
		BndlCFBundleDeliveryStatusReportsAreRequested |
		BndlCFBundleMustNotBeFragmented

	destination, _ := NewEndpointID("dtn", "foobar")
	source, _ := NewEndpointID("dtn", "me")

	creationTimestamp := NewCreationTimestamp(DTNTimeNow(), 0)
	lifetime := uint(10 * 60 * 1000)

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
	pb.BundleControlFlags = BndlCFBundleIsAFragment

	if !pb.HasFragmentation() {
		t.Error("Primary Block should be fragmented")
	}
}

func TestPrimaryBlockCbor(t *testing.T) {
	ep, _ := NewEndpointID("dtn", "test")
	ts := NewCreationTimestamp(DTNTimeNow(), 23)

	tests := []struct {
		pb1 PrimaryBlock
		len int
	}{
		// No CRC, No Fragmentation
		{PrimaryBlock{7, 0, CRCNo, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 8},
		// CRC, No Fragmentation
		{PrimaryBlock{7, 0, CRC16, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 9},
		// No CRC, Fragmentation
		{PrimaryBlock{7, BndlCFBundleIsAFragment, CRCNo, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 10},
		// CRC, Fragmentation
		{PrimaryBlock{7, BndlCFBundleIsAFragment, CRC16, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 11},
	}

	for _, test := range tests {
		var b []byte = make([]byte, 0, 64)
		var h codec.Handle = new(codec.CborHandle)
		var enc *codec.Encoder = codec.NewEncoderBytes(&b, h)

		// If we are going to test block's with a CRC value, we also have to
		// calculate it.
		if test.pb1.HasCRC() {
			setCRC(&test.pb1)
		}

		err := enc.Encode(test.pb1)
		if err != nil {
			t.Errorf("CBOR encoding failed: %v", err)
		}

		var decGeneric interface{}
		err = codec.NewDecoderBytes(b, h).Decode(&decGeneric)
		if err != nil {
			t.Errorf("Generic CBOR decoding failed: %v", err)
		}

		if ty := reflect.TypeOf(decGeneric); ty.Kind() != reflect.Slice {
			t.Errorf("Decoded CBOR has wrong type: %v instead of slice", ty.Kind())
		}

		var arr []interface{} = decGeneric.([]interface{})
		if len(arr) != test.len {
			t.Errorf("CBOR-Array has wrong length: %d instead of %d",
				len(arr), test.len)
		}

		var pb2 PrimaryBlock
		err = codec.NewDecoderBytes(b, h).Decode(&pb2)
		if err != nil {
			t.Errorf("CBOR decoding failed: %v", err)
		}

		v1 := reflect.ValueOf(test.pb1)
		v2 := reflect.ValueOf(pb2)

		if v1.NumField() != v2.NumField() {
			t.Errorf("PrimaryBlock's number of fields changed after CBOR: %d to %d",
				v1.NumField(), v2.NumField())
		}

		for i := 0; i < v1.NumField(); i++ {
			val1 := v1.Field(i)
			val2 := v2.Field(i)

			if val1.Type() != val2.Type() {
				t.Errorf("Type of value no %d differs: %T and %T",
					i, val1.Type(), val2.Type())
			}

			s1 := fmt.Sprintf("%v", val1)
			s2 := fmt.Sprintf("%v", val2)

			if s1 != s2 {
				t.Errorf("String representation of value no %d differs: %v and %v",
					i, s1, s2)
			}
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
			23, 0, CRCNo, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DTNTimeEpoch, 0), 0, 0, 0, nil}, false},
		{PrimaryBlock{
			7, 0, CRCNo, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DTNTimeEpoch, 0), 0, 0, 0, nil}, true},

		// Reserved bits in bundle control flags
		{PrimaryBlock{
			7, 0xFF00, CRCNo, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DTNTimeEpoch, 0), 0, 0, 0, nil}, false},

		// Illegal EndpointID
		{PrimaryBlock{
			7, 0, CRCNo,
			EndpointID{SchemeName: URISchemeIPN, SchemeSpecificPart: [2]uint64{0, 0}},
			DtnNone(), DtnNone(), NewCreationTimestamp(DTNTimeEpoch, 0), 0, 0, 0, nil},
			false},

		// Everything from above
		{PrimaryBlock{
			23, 0xFF00, CRCNo,
			EndpointID{SchemeName: URISchemeIPN, SchemeSpecificPart: [2]uint64{0, 0}},
			DtnNone(), DtnNone(), NewCreationTimestamp(DTNTimeEpoch, 0), 0, 0, 0, nil},
			false},
	}

	for _, test := range tests {
		if err := test.pb.checkValid(); (err == nil) != test.valid {
			t.Errorf("PrimaryBlock validation failed: %v resulted in %v",
				test.pb, err)
		}
	}
}
