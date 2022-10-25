// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"reflect"
	"testing"
)

func TestExtensionBlockManager(t *testing.T) {
	var ebm = NewExtensionBlockManager()

	payloadBlock := NewPayloadBlock(nil)
	if err := ebm.Register(payloadBlock); err != nil {
		t.Fatal(err)
	}
	if err := ebm.Register(payloadBlock); err == nil {
		t.Fatal("Registering the PayloadBlock twice did not erred")
	}

	if !ebm.IsKnown(payloadBlock.BlockTypeCode()) {
		t.Fatal("PayloadBlock's type code is unknown")
	}

	extBlock := ebm.createBlock(payloadBlock.BlockTypeCode())
	if extBlock.BlockTypeCode() != payloadBlock.BlockTypeCode() {
		t.Fatalf("Block type code differs: %d != %d",
			extBlock.BlockTypeCode(), payloadBlock.BlockTypeCode())
	}

	if ebm.IsKnown(9001) {
		t.Fatal("CreateBlock for an unknown number is possible")
	}

	ebm.Unregister(payloadBlock)
	if ebm.IsKnown(payloadBlock.BlockTypeCode()) {
		t.Fatal("PayloadBlock's type code is known")
	}
}

func TestExtensionBlockManagerRWBlock(t *testing.T) {
	var ebm = GetExtensionBlockManager()

	tests := []struct {
		from     ExtensionBlock
		to       []byte
		typeCode uint64
	}{
		// CBOR; wrapped within a CBOR byte string
		{NewBundleAgeBlock(23), []byte{0x41, 0x17}, ExtBlockTypeBundleAgeBlock},
		{NewHopCountBlock(16), []byte{0x43, 0x82, 0x10, 0x00}, ExtBlockTypeHopCountBlock},
		{NewPreviousNodeBlock(MustNewEndpointID("dtn://23/")), []byte{0x48, 0x82, 0x01, 0x65, 0x2F, 0x2F, 0x32, 0x33, 0x2F}, ExtBlockTypePreviousNodeBlock},

		// Binary; also wrapped, of course
		{NewGenericExtensionBlock([]byte{0xFF}, 192), []byte{0x41, 0xFF}, 192},
		{NewPayloadBlock([]byte("lel")), []byte{0x43, 0x6C, 0x65, 0x6C}, ExtBlockTypePayloadBlock},
	}

	for _, test := range tests {
		// Block -> Binary / CBOR
		var buff = new(bytes.Buffer)
		if err := ebm.WriteBlock(test.from, buff); err != nil {
			t.Fatal(err)
		} else if to := buff.Bytes(); !bytes.Equal(to, test.to) {
			t.Fatalf("Bytes are not equal: %x != %x", test.to, to)
		}

		// Binary / CBOR -> Block
		buff = bytes.NewBuffer(test.to)
		if b, err := ebm.ReadBlock(test.typeCode, buff); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(b, test.from) {
			t.Fatalf("Blocks differ: %v %v", test.from, b)
		}
	}
}

func TestExtensionBlockManagerGenericRegister(t *testing.T) {
	var ebm = NewExtensionBlockManager()
	var geb = NewGenericExtensionBlock([]byte("nope"), 192)

	if err := ebm.Register(geb); err == nil {
		t.Fatalf("Registering a GenericExtensionBlock did not erred")
	}
}
