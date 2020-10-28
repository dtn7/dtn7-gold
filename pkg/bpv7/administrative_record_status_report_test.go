// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func TestBundleStatusItemCbor(t *testing.T) {
	tests := []struct {
		bsi BundleStatusItem
		len int
	}{
		{NewTimeReportingBundleStatusItem(DtnTimeNow()), 2},
		{NewTimeReportingBundleStatusItem(DtnTimeEpoch), 2},
		{NewBundleStatusItem(true), 1},
		{NewBundleStatusItem(false), 1},
	}

	for _, test := range tests {
		buff := new(bytes.Buffer)

		// CBOR encoding
		if err := cboring.Marshal(&test.bsi, buff); err != nil {
			t.Fatalf("Encoding %v failed: %v", test.bsi, err)
		}

		// CBOR decoding
		var bsiComp BundleStatusItem
		if err := cboring.Unmarshal(&bsiComp, buff); err != nil {
			t.Fatalf("Decoding %v failed: %v", test.bsi, err)
		}

		if test.bsi.Asserted != bsiComp.Asserted || test.bsi.Time != bsiComp.Time {
			t.Fatalf("Decoded BundleStatusItem differs: %v, %v", test.bsi, bsiComp)
		}
	}
}

func TestStatusReportCreation(t *testing.T) {
	var bndl, err = Builder().
		Source("dtn://src/").
		Destination("dtn://dest/").
		CreationTimestampNow().
		Lifetime("60s").
		BundleCtrlFlags(MustNotFragmented | RequestStatusTime).
		PayloadBlock([]byte("hello world!")).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	var initTime = DtnTimeNow()
	var statusRep = NewStatusReport(
		bndl, ReceivedBundle, NoInformation, initTime)

	// Check bundle status report's fields
	bsi := statusRep.StatusInformation[ReceivedBundle]
	if !bsi.Asserted || bsi.Time != initTime {
		t.Fatalf("ReceivedBundle's status item is incorrect: %v", bsi)
	}

	for i := 0; i < maxStatusInformationPos; i++ {
		if StatusInformationPos(i) == ReceivedBundle {
			continue
		}
		if statusRep.StatusInformation[i].Asserted {
			t.Fatalf("Invalid status item is asserted: %d", i)
		}
	}

	// CBOR
	buff := new(bytes.Buffer)
	if err := cboring.Marshal(statusRep, buff); err != nil {
		t.Fatal(err)
	}

	statusRepDec := new(StatusReport)
	if err := cboring.Unmarshal(statusRepDec, buff); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(statusRep, statusRepDec) {
		t.Fatalf("CBOR result differs:\n%v\n%v", statusRep, statusRepDec)
	}
}

func TestStatusReportCreationNoTime(t *testing.T) {
	var bndl, err = Builder().
		Source("dtn://src/").
		Destination("dtn://dest/").
		CreationTimestampNow().
		Lifetime("60s").
		BundleCtrlFlags(MustNotFragmented).
		PayloadBlock([]byte("hello world!")).
		Build()
	if err != nil {
		panic(err)
	}

	var statusRep = NewStatusReport(bndl, ReceivedBundle, NoInformation, DtnTimeNow())

	// Test no time is present.
	bsi := statusRep.StatusInformation[ReceivedBundle]
	if !bsi.Asserted || bsi.Time != DtnTimeEpoch {
		t.Fatalf("ReceivedBundle's status item is incorrect: %v", bsi)
	}
}

func TestStatusReportApplicationRecord(t *testing.T) {
	bndl, err := Builder().
		Source("dtn://src/").
		Destination("dtn://dest/").
		CreationTimestampNow().
		Lifetime("60s").
		BundleCtrlFlags(MustNotFragmented | RequestStatusTime).
		PayloadBlock([]byte("hello world!")).
		Build()
	if err != nil {
		t.Fatalf("Creating bundle failed: %v", err)
	}

	initTime := DtnTimeNow()
	statusRep := NewStatusReport(
		bndl, ReceivedBundle, NoInformation, initTime)

	adminRec, adminRecErr := AdministrativeRecordToCbor(statusRep)
	if adminRecErr != nil {
		t.Fatal(adminRecErr)
	}

	outBndl, err := Builder().
		Source("dtn://foo/").
		Destination(bndl.PrimaryBlock.ReportTo).
		CreationTimestampNow().
		Lifetime("60m").
		BundleCtrlFlags(AdministrativeRecordPayload).
		Canonical(adminRec).
		Build()
	if err != nil {
		t.Fatalf("Creating new bundle failed: %v", err)
	}

	buff := new(bytes.Buffer)
	if err := outBndl.WriteBundle(buff); err != nil {
		t.Fatal(err)
	}

	inBndl, inBndlErr := ParseBundle(buff)
	if inBndlErr != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(outBndl, inBndl) {
		t.Fatalf("CBOR result differs: %v, %v", outBndl, inBndl)
	}
}
