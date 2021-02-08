// SPDX-FileCopyrightText: 2018, 2019, 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func setupPrimaryBlock() PrimaryBlock {
	bcf := StatusRequestDeletion |
		StatusRequestDelivery |
		MustNotFragmented

	destination, _ := NewEndpointID("dtn://foobar/")
	source, _ := NewEndpointID("dtn://me/")

	creationTimestamp := NewCreationTimestamp(DtnTimeEpoch, 0)
	lifetime := uint64(10 * 60 * 1000)

	return NewPrimaryBlock(bcf, destination, source, creationTimestamp, lifetime)
}

func TestNewPrimaryBlock(t *testing.T) {
	pb := setupPrimaryBlock()

	if !pb.HasCRC() {
		t.Fatal("Primary Block has no CRC")
	}

	if pb.HasFragmentation() {
		t.Fatal("Primary Block is fragmented")
	}
}

func TestPrimaryBlockCRC(t *testing.T) {
	pb := setupPrimaryBlock()
	pb.CRCType = CRC16

	if !pb.HasCRC() {
		t.Fatal("Primary Block should need a CRC")
	}
}

func TestPrimaryBlockFragmentation(t *testing.T) {
	pb := setupPrimaryBlock()
	pb.BundleControlFlags = IsFragment

	if !pb.HasFragmentation() {
		t.Fatal("Primary Block should be fragmented")
	}
}

func TestPrimaryBlockCbor(t *testing.T) {
	ep, _ := NewEndpointID("dtn://test/")
	ts := NewCreationTimestamp(DtnTimeEpoch, 23)

	tests := []struct {
		pb1 PrimaryBlock
		len int
	}{
		// No CRC, No Fragmentation
		{PrimaryBlock{7, 0, CRCNo, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 8},
		// No Fragmentation, CRC
		{PrimaryBlock{7, 0, CRC16, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 9},
		// Fragmentation, No CRC
		{PrimaryBlock{7, IsFragment, CRCNo, ep, ep, DtnNone(), ts, 1000000, 0, 0, nil}, 10},
		// Fragmentation, CRC
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

func TestPrimaryBlockJson(t *testing.T) {
	tests := []struct {
		pb        PrimaryBlock
		jsonBytes []byte
	}{
		// CRC, No Fragmentation
		{PrimaryBlock{
			BundleControlFlags: 0,
			CRCType:            CRC32,
			Destination:        MustNewEndpointID("dtn://dst/"),
			SourceNode:         MustNewEndpointID("dtn://src/"),
			ReportTo:           MustNewEndpointID("dtn://rprt/"),
			CreationTimestamp:  NewCreationTimestamp(0, 42),
			Lifetime:           3600,
		}, []byte(`{"bundleControlFlags":null,"destination":"dtn://dst/","source":"dtn://src/","reportTo":"dtn://rprt/","creationTimestamp":{"date":"2000-01-01 00:00:00.000","sequenceNo":42},"lifetime":3600}`)},
		{PrimaryBlock{
			BundleControlFlags: MustNotFragmented,
			CRCType:            CRCNo,
			Destination:        MustNewEndpointID("ipn:23.42"),
			SourceNode:         MustNewEndpointID("dtn://foo/"),
			ReportTo:           MustNewEndpointID("dtn://bar/"),
			CreationTimestamp:  NewCreationTimestamp(0, 0),
			Lifetime:           10,
		}, []byte(`{"bundleControlFlags":["MUST_NOT_BE_FRAGMENTED"],"destination":"ipn:23.42","source":"dtn://foo/","reportTo":"dtn://bar/","creationTimestamp":{"date":"2000-01-01 00:00:00.000","sequenceNo":0},"lifetime":10}`)},
	}

	for _, test := range tests {
		if jsonBytes, err := json.Marshal(test.pb); err != nil {
			t.Fatal(err)
		} else if !bytes.Equal(test.jsonBytes, jsonBytes) {
			t.Fatalf("expected %s, got %s", test.jsonBytes, jsonBytes)
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
			23, MustNotFragmented, CRC32, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil}, false},
		{PrimaryBlock{
			7, MustNotFragmented, CRC32, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil}, true},

		// Reserved bits in bundle control flags
		{PrimaryBlock{
			7, 0xFF00, CRCNo, DtnNone(), DtnNone(), DtnNone(),
			NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil}, false},

		// Illegal EndpointID
		{PrimaryBlock{
			7, 0, CRCNo,
			EndpointID{&IpnEndpoint{0, 0}},
			DtnNone(), DtnNone(), NewCreationTimestamp(DtnTimeEpoch, 0), 0, 0, 0, nil},
			false},

		// Everything from above
		{PrimaryBlock{
			23, 0xFF00, CRCNo,
			EndpointID{&IpnEndpoint{0, 0}},
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
		if err := test.pb.CheckValid(); (err == nil) != test.valid {
			t.Fatalf("PrimaryBlock validation failed: %v resulted in %v",
				test.pb, err)
		}
	}
}
