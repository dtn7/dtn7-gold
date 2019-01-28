package core

import (
	"reflect"
	"testing"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

func TestBundleStatusItemCbor(t *testing.T) {
	tests := []struct {
		bsi BundleStatusItem
		len int
	}{
		{NewReportingBundleStatusItem(bundle.DtnTimeNow()), 2},
		{NewReportingBundleStatusItem(bundle.DtnTimeEpoch), 2},
		{NewNegativeBundleStatusItem(), 1},
	}

	for _, test := range tests {
		// CBOR encoding
		var b []byte = make([]byte, 0, 64)
		var enc = codec.NewEncoderBytes(&b, new(codec.CborHandle))

		if err := enc.Encode(test.bsi); err != nil {
			t.Errorf("Encoding %v failed: %v", test.bsi, err)
		}

		// CBOR decoding back to BundleStatusItem
		var dec = codec.NewDecoderBytes(b, new(codec.CborHandle))
		var bsiComp BundleStatusItem

		if err := dec.Decode(&bsiComp); err != nil {
			t.Errorf("Decoding %v failed: %v", test.bsi, err)
		}

		if test.bsi.Asserted != bsiComp.Asserted || test.bsi.Time != bsiComp.Time {
			t.Errorf("Decoded BundleStatusItem differs: %v, %v", test.bsi, bsiComp)
		}

		// CBOR decoding to unknown array
		var unknown interface{}

		dec = codec.NewDecoderBytes(b, new(codec.CborHandle))
		if err := dec.Decode(&unknown); err != nil {
			t.Errorf("Decoding %v into interface failed: %v", test.bsi, err)
		}

		if ty := reflect.TypeOf(unknown).Kind(); ty != reflect.Slice {
			t.Errorf("Decoded BundleStatusItem is not a slice, %v", ty)
		}

		if arr := unknown.([]interface{}); len(arr) != test.len {
			t.Errorf("Decoded array has wrong length: %d instead of %d",
				len(arr), test.len)
		}
	}
}
