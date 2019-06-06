package core

import (
	"reflect"
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/ugorji/go/codec"
)

func TestBundleStatusItemCbor(t *testing.T) {
	tests := []struct {
		bsi BundleStatusItem
		len int
	}{
		{NewTimeReportingBundleStatusItem(bundle.DtnTimeNow()), 2},
		{NewTimeReportingBundleStatusItem(bundle.DtnTimeEpoch), 2},
		{NewBundleStatusItem(true), 1},
		{NewBundleStatusItem(false), 1},
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

func TestStatusReportCreation(t *testing.T) {
	var bndl, err = bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented|bundle.RequestStatusTime,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		panic(err)
	}

	var initTime = bundle.DtnTimeNow()
	var statusRep = NewStatusReport(
		bndl, ReceivedBundle, NoInformation, initTime)

	// Check bundle status report's fields
	bsi := statusRep.StatusInformation[ReceivedBundle]
	if bsi.Asserted != true || bsi.Time != initTime {
		t.Errorf("ReceivedBundle's status item is incorrect: %v", bsi)
	}

	for i := 0; i < maxStatusInformationPos; i++ {
		if StatusInformationPos(i) == ReceivedBundle {
			continue
		}
		if statusRep.StatusInformation[i].Asserted == true {
			t.Errorf("Invalid status item is asserted: %d", i)
		}
	}

	// CBOR
	var b []byte = make([]byte, 0, 64)
	var enc = codec.NewEncoderBytes(&b, new(codec.CborHandle))

	if err := enc.Encode(statusRep); err != nil {
		t.Errorf("Encoding failed: %v", err)
	}

	var dec = codec.NewDecoderBytes(b, new(codec.CborHandle))
	var statusRepDec StatusReport

	if err := dec.Decode(&statusRepDec); err != nil {
		t.Errorf("Decoding failed: %v", err)
	}

	if !reflect.DeepEqual(statusRep, statusRepDec) {
		t.Errorf("CBOR result differs: %v, %v", statusRep, statusRepDec)
	}
}

func TestStatusReportCreationNoTime(t *testing.T) {
	var bndl, err = bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		panic(err)
	}

	var statusRep = NewStatusReport(
		bndl, ReceivedBundle, NoInformation, bundle.DtnTimeNow())

	// Test no time is present.
	bsi := statusRep.StatusInformation[ReceivedBundle]
	if bsi.Asserted != true || bsi.Time != bundle.DtnTimeEpoch {
		t.Errorf("ReceivedBundle's status item is incorrect: %v", bsi)
	}
}

func TestStatusReportApplicationRecord(t *testing.T) {
	bndl, err := bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented|bundle.RequestStatusTime,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0), 60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		t.Errorf("Creating bundle failed: %v", err)
	}

	initTime := bundle.DtnTimeNow()
	statusRep := NewStatusReport(
		bndl, ReceivedBundle, NoInformation, initTime)

	adminRec := NewAdministrativeRecord(BundleStatusReportTypeCode, statusRep)

	primary := bundle.NewPrimaryBlock(
		bundle.AdministrativeRecordPayload,
		bndl.PrimaryBlock.ReportTo,
		bundle.MustNewEndpointID("dtn:foo"),
		bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0),
		60*60*1000000)

	outBndl, err := bundle.NewBundle(
		primary,
		[]bundle.CanonicalBlock{
			adminRec.ToCanonicalBlock(),
		})
	if err != nil {
		t.Errorf("Creating new bundle failed: %v", err)
	}

	outBndlData := outBndl.ToCbor()

	inBndl, err := bundle.NewBundleFromCbor(&outBndlData)
	if err != nil {
		t.Errorf("Parsing bundle failed: %v", err)
	}

	if !reflect.DeepEqual(outBndl, inBndl) {
		t.Errorf("CBOR result differs: %v, %v", outBndl, inBndl)
	}
}
