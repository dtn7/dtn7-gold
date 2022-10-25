// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestBundleBuilderSimple(t *testing.T) {
	bndl, err := Builder().
		CRC(CRC32).
		Source("dtn://myself/").
		Destination("dtn://dest/").
		CreationTimestampEpoch().
		Lifetime("10m").
		HopCountBlock(64).
		BundleAgeBlock(0).
		PayloadBlock([]byte("hello world!")).
		Build()

	if err != nil {
		t.Fatalf("Builder erred: %v", err)
	}

	buff := new(bytes.Buffer)
	if err := bndl.MarshalCbor(buff); err != nil {
		t.Fatal(err)
	}
	bndlCbor := buff.Bytes()

	bndl2 := Bundle{}
	if err = bndl2.UnmarshalCbor(buff); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(bndl, bndl2) {
		t.Fatalf("Bundle changed after serialization: %v, %v", bndl, bndl2)
	}

	bndl3, err := NewBundle(
		NewPrimaryBlock(
			StatusRequestDelivery,
			MustNewEndpointID("dtn://dest/"),
			MustNewEndpointID("dtn://myself/"),
			NewCreationTimestamp(DtnTimeEpoch, 0),
			1000*60*10),
		[]CanonicalBlock{
			NewCanonicalBlock(2, ReplicateBlock, NewHopCountBlock(64)),
			NewCanonicalBlock(3, ReplicateBlock, NewBundleAgeBlock(0)),
			NewCanonicalBlock(1, 0, NewPayloadBlock([]byte("hello world!")))})
	if err != nil {
		t.Fatal(err)
	}

	buff.Reset()
	bndl3.PrimaryBlock.ReportTo = bndl3.PrimaryBlock.SourceNode
	bndl3.SetCRCType(CRC32)

	if err := bndl3.MarshalCbor(buff); err != nil {
		t.Fatal(err)
	}
	bndl3Cbor := buff.Bytes()

	if !bytes.Equal(bndlCbor, bndl3Cbor) {
		t.Fatalf("CBOR has changed:\n%x\n%x", bndlCbor, bndl3Cbor)
	}

	if !reflect.DeepEqual(bndl, bndl3) {
		t.Fatalf("Bundles differ: %v, %v", bndl, bndl3)
	}
}

func TestBldrParseEndpoint(t *testing.T) {
	eidIn, _ := NewEndpointID("dtn://foo/bar/")
	if eidTmp, _ := bldrParseEndpoint(eidIn); eidTmp != eidIn {
		t.Fatalf("Endpoint does not match: %v != %v", eidTmp, eidIn)
	}

	if eidTmp, _ := bldrParseEndpoint("dtn://foo/bar/"); eidTmp != eidIn {
		t.Fatalf("Parsed endpoint does not match: %v != %v", eidTmp, eidIn)
	}

	if _, errTmp := bldrParseEndpoint(23.42); errTmp == nil {
		t.Fatalf("Invalid endpoint type does not resulted in an error")
	}
}

func TestBldrParseLifetime(t *testing.T) {
	tests := []struct {
		val interface{}
		ms  uint64
		err bool
	}{
		{1000, 1000, false},
		{uint64(1000), 1000, false},
		{"1000ms", 1000, false},
		{"1000us", 1, false},
		{"1000s", 1000000, false},
		{"1s", 1000, false},
		{"1m", 60000, false},
		{time.Millisecond, 1, false},
		{time.Second, 1000, false},
		{time.Minute, 60000, false},
		{10 * time.Minute, 600000, false},
		{-23, 0, true},
		{"-10m", 0, true},
		{true, 0, true},
	}

	for _, test := range tests {
		ms, err := bldrParseLifetime(test.val)

		if test.err == (err == nil) {
			t.Fatalf("Error value for %v was unexpected: %v != %v",
				test.val, test.err, err)
		}

		if test.ms != ms {
			t.Fatalf("Value for %v was unexpected: %v != %v", test.val, test.ms, ms)
		}
	}
}

func TestBundleBuilderAdministrativeRecord(t *testing.T) {
	originBundle, err := Builder().
		CRC(CRC32).
		Source("dtn://host-a/").
		Destination("dtn://host-b/").
		CreationTimestampNow().
		Lifetime(time.Hour).
		PayloadBlock([]byte("hello world")).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	reportBundle, err := Builder().
		CRC(CRC32).
		Source("dtn://host-b/").
		Destination(originBundle.PrimaryBlock.ReportTo).
		CreationTimestampNow().
		Lifetime(time.Hour).
		StatusReport(originBundle, DeliveredBundle, NoInformation).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	report, err := reportBundle.AdministrativeRecord()
	if err != nil {
		t.Fatal(err)
	}

	statusReport, ok := report.(*StatusReport)
	if !ok {
		t.Fatalf("report %v / %T is not an StatusReprot", report, report)
	}

	if statusReport.RefBundle != originBundle.ID() {
		t.Fatalf("reference bundle id is %v, not %v", statusReport.RefBundle, originBundle.ID())
	}
	if statusReport.ReportReason != NoInformation {
		t.Fatalf("status reason is %v, not %v", statusReport.ReportReason, NoInformation)
	}
	if sr := statusReport.StatusInformations(); len(sr) != 1 || sr[0] != DeliveredBundle {
		t.Fatalf("status information are invalid: %v", sr)
	}
}

func TestBuildFromMap(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		wantBndl Bundle
		wantErr  bool
	}{
		{
			name: "simple",
			args: map[string]interface{}{
				"destination":              "dtn://dst/",
				"source":                   "dtn://src/",
				"creation_timestamp_epoch": true,
				"lifetime":                 "24h",
				"bundle_age_block":         23,
				"payload_block":            []byte("hello world"),
			},
			wantBndl: Builder().
				Destination("dtn://dst/").
				Source("dtn://src/").
				CreationTimestampEpoch().
				Lifetime("24h").
				BundleAgeBlock(23).
				PayloadBlock([]byte("hello world")).
				mustBuild(),
			wantErr: false,
		},
		{
			name: "payload as string",
			args: map[string]interface{}{
				"destination":              "dtn://dst/",
				"source":                   "dtn://src/",
				"creation_timestamp_epoch": true,
				"lifetime":                 "24h",
				"bundle_age_block":         23,
				"payload_block":            "hello world",
			},
			wantBndl: Builder().
				Destination("dtn://dst/").
				Source("dtn://src/").
				CreationTimestampEpoch().
				Lifetime("24h").
				BundleAgeBlock(23).
				PayloadBlock([]byte("hello world")).
				mustBuild(),
			wantErr: false,
		},
		{
			name: "illegal method",
			args: map[string]interface{}{
				"nope": "nope",
			},
			wantBndl: Bundle{},
			wantErr:  true,
		},
		{
			name: "no source",
			args: map[string]interface{}{
				"destination":              "dtn://dst/",
				"creation_timestamp_epoch": true,
				"lifetime":                 "24h",
				"payload_block":            []byte("hello world"),
			},
			wantBndl: Bundle{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBndl, err := BuildFromMap(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("BuildFromMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotBndl, tt.wantBndl) {
				t.Fatalf("BuildFromMap() gotBndl = %v, want %v", gotBndl, tt.wantBndl)
			}
		})
	}
}

func TestBuildFromMapJSON(t *testing.T) {
	var args map[string]interface{}
	data := []byte(`{
		"destination":            "dtn://dst/",
		"source":                 "dtn://src/",
		"creation_timestamp_epoch": 1,
		"lifetime":               "24h",
		"bundle_age_block":        23,
		"payload_block":          "hello world"
	}`)

	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatal(err)
	}

	expectedBndl := Builder().
		Destination("dtn://dst/").
		Source("dtn://src/").
		CreationTimestampEpoch().
		Lifetime("24h").
		BundleAgeBlock(23).
		PayloadBlock([]byte("hello world")).
		mustBuild()

	if bndl, err := BuildFromMap(args); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(expectedBndl, bndl) {
		t.Fatalf("%v != %v", expectedBndl, bndl)
	}
}
