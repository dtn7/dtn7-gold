// SPDX-FileCopyrightText: 2018, 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"testing"
)

func TestNewCanonicalBlock(t *testing.T) {
	b := NewCanonicalBlock(1, ReplicateBlock, NewPayloadBlock([]byte("hello world")))

	if b.HasCRC() {
		t.Errorf("Canonical Block (Payload Block) has CRC: %v", b)
	}

	b.CRCType = CRC32
	if !b.HasCRC() {
		t.Errorf("Canonical Block (Payload Block) has no CRC: %v", b)
	}
}

func TestCanonicalBlockCheckValid(t *testing.T) {
	tests := []struct {
		cb    CanonicalBlock
		valid bool
	}{
		// Payload block with a block number != one
		{CanonicalBlock{9, 0, CRCNo, nil, NewPayloadBlock(nil)}, false},
		{CanonicalBlock{1, 0, CRCNo, nil, NewPayloadBlock(nil)}, true},

		// Reserved bits in block control flags
		{CanonicalBlock{1, 0x80, CRCNo, nil, NewPayloadBlock(nil)}, true},

		// Illegal EndpointID in Previous Node Block
		{CanonicalBlock{2, 0, CRCNo, nil, NewPreviousNodeBlock(DtnNone())}, true},
	}

	for _, test := range tests {
		if err := test.cb.CheckValid(); (err == nil) != test.valid {
			t.Errorf("CanonicalBlock validation failed: %v resulted in %v",
				test.cb, err)
		}
	}
}

func TestHopCountBlock(t *testing.T) {
	tests := []struct {
		hcb                    *HopCountBlock
		exceeded               bool
		exceededAfterIncrement bool
	}{
		{NewHopCountBlock(10), false, false},
		{NewHopCountBlock(1), false, false},
		{NewHopCountBlock(0), false, true},
		{&HopCountBlock{Limit: 23, Count: 20}, false, false},
		{&HopCountBlock{Limit: 23, Count: 22}, false, false},
		{&HopCountBlock{Limit: 23, Count: 23}, false, true},
	}

	for _, test := range tests {
		if state := test.hcb.IsExceeded(); state != test.exceeded {
			t.Errorf("Hop count block's %v state is wrong: expected %t, real %t",
				test.hcb, test.exceeded, state)
		}

		if state := test.hcb.Increment(); state != test.exceededAfterIncrement {
			t.Errorf("Hop count block's state %v is wrong after increment: expected %t, real %t",
				test.hcb, test.exceededAfterIncrement, state)
		}
	}
}

func TestCanonicalBlockJson(t *testing.T) {
	tests := []struct {
		cb        CanonicalBlock
		jsonBytes []byte
	}{
		{CanonicalBlock{
			BlockNumber: 1,
			Value:       NewPayloadBlock([]byte("hello world")),
		}, []byte(`{"blockNumber":1,"blockTypeCode":1,"blockType":"Payload Block","blockControlFlags":null,"data":"aGVsbG8gd29ybGQ="}`)},
		{CanonicalBlock{
			BlockNumber:       23,
			BlockControlFlags: DeleteBundle,
			Value:             NewGenericExtensionBlock(nil, 42),
		}, []byte(`{"blockNumber":23,"blockTypeCode":42,"blockType":"N/A","blockControlFlags":["DELETE_BUNDLE"],"data":"QA=="}`)},
		{CanonicalBlock{
			BlockNumber: 1,
			Value:       NewBundleAgeBlock(23),
		}, []byte(`{"blockNumber":1,"blockTypeCode":7,"blockType":"Bundle Age Block","blockControlFlags":null,"data":"23 ms"}`)},
		{CanonicalBlock{
			BlockNumber: 1,
			Value:       NewHopCountBlock(23),
		}, []byte(`{"blockNumber":1,"blockTypeCode":10,"blockType":"Hop Count Block","blockControlFlags":null,"data":{"limit":23,"count":0}}`)},
		{CanonicalBlock{
			BlockNumber: 1,
			Value:       NewPreviousNodeBlock(MustNewEndpointID("dtn://foo/23")),
		}, []byte(`{"blockNumber":1,"blockTypeCode":6,"blockType":"Previous Node Block","blockControlFlags":null,"data":"dtn://foo/23"}`)},
	}

	for _, test := range tests {
		if jsonBytes, err := test.cb.MarshalJSON(); err != nil {
			t.Fatal(err)
		} else if !bytes.Equal(test.jsonBytes, jsonBytes) {
			t.Fatalf("expected %s, got %s", test.jsonBytes, jsonBytes)
		}
	}
}
