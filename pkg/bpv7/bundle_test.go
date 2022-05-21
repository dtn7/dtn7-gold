// SPDX-FileCopyrightText: 2018, 2019, 2020, 2021, 2022 Alvar Penning
// SPDX-FileCopyrightText: 2022 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func TestBundleApplyCRC(t *testing.T) {
	var epPrim, _ = NewEndpointID("dtn://foo/bar/")
	var creationTs = NewCreationTimestamp(42000000000000, 23)

	var primary = NewPrimaryBlock(
		StatusRequestDelivery,
		epPrim, epPrim, creationTs, 42000000)

	var epPrev, _ = NewEndpointID("ipn:23.42")
	var prevNode = NewCanonicalBlock(2, 0, NewPreviousNodeBlock(epPrev))

	var payload = NewCanonicalBlock(1, DeleteBundle, NewPayloadBlock([]byte("GuMo")))

	var bndle, err = NewBundle(
		primary, []CanonicalBlock{prevNode, payload})

	if err != nil {
		t.Fatal(err)
	}

	for _, crcTest := range []CRCType{CRCNo, CRC16, CRC32, CRCNo} {
		bndle.SetCRCType(crcTest)

		// NOTE: Until BPsec has landed, set a CRC for primary blocks, see PrimaryBlock.SetCRCType.
		if crcTest == CRCNo {
			crcTest = CRC32
		}

		if ty := bndle.PrimaryBlock.GetCRCType(); ty != crcTest {
			t.Fatalf("Bundle's primary block has wrong CRCType, %v instead of %v", ty, crcTest)
		}

		buff := new(bytes.Buffer)
		if err := cboring.Marshal(&bndle, buff); err != nil {
			t.Fatal(err)
		}

		bndl2 := Bundle{}
		if err := cboring.Unmarshal(&bndl2, buff); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBundleCbor(t *testing.T) {
	var epDest, _ = NewEndpointID("dtn://desty/")
	var epSource, _ = NewEndpointID("dtn://gumo/")
	var creationTs = NewCreationTimestamp(42000000000000, 23)

	var primary = NewPrimaryBlock(
		StatusRequestDelivery,
		epDest, epSource, creationTs, 42000000)

	var epPrev, _ = NewEndpointID("ipn:23.42")
	var prevNode = NewCanonicalBlock(23, 0, NewPreviousNodeBlock(epPrev))

	var payload = NewCanonicalBlock(
		1, DeleteBundle, NewPayloadBlock([]byte("GuMo meine Kernel")))

	bundle1, err := NewBundle(
		primary, []CanonicalBlock{prevNode, payload})
	if err != nil {
		t.Fatal(err)
	}

	bundle1.SetCRCType(CRC32)

	buff := new(bytes.Buffer)
	if err := cboring.Marshal(&bundle1, buff); err != nil {
		t.Fatal(err)
	}
	bundle1Cbor := buff.Bytes()

	bundle2 := Bundle{}
	if err := cboring.Unmarshal(&bundle2, buff); err != nil {
		t.Fatal(err)
	}

	buff.Reset()
	if err := cboring.Marshal(&bundle2, buff); err != nil {
		t.Fatal(err)
	}
	bundle2Cbor := buff.Bytes()

	if !bytes.Equal(bundle1Cbor, bundle2Cbor) {
		t.Fatalf("Cbor-Representations do not match:\n- %x\n- %x",
			bundle1Cbor, bundle2Cbor)
	}

	if !reflect.DeepEqual(bundle1, bundle2) {
		t.Fatalf("Bundles do not match:\n%v\n%v", bundle1, bundle2)
	}
}

func TestBundleExtensionBlock(t *testing.T) {
	var bndl, err = NewBundle(
		NewPrimaryBlock(
			MustNotFragmented,
			MustNewEndpointID("dtn://some/"), DtnNone(),
			NewCreationTimestamp(DtnTimeEpoch, 0), 3600),
		[]CanonicalBlock{
			NewCanonicalBlock(2, 0, NewBundleAgeBlock(420)),
			NewCanonicalBlock(1, 0, NewPayloadBlock([]byte("hello world"))),
		})

	if err != nil {
		t.Fatal(err)
	}

	if cb, err := bndl.ExtensionBlock(ExtBlockTypePreviousNodeBlock); err == nil {
		t.Fatalf("Bundle returned a non-existing Extension Block: %v", cb)
	}

	if _, err := bndl.ExtensionBlock(ExtBlockTypeBundleAgeBlock); err != nil {
		t.Fatalf("Bundle did not returned the existing Bundle Age block: %v", err)
	}

	if _, err := bndl.ExtensionBlock(ExtBlockTypePayloadBlock); err != nil {
		t.Fatalf("Bundle did not returned the existing Payload block: %v", err)
	}

	if _, err := bndl.PayloadBlock(); err != nil {
		t.Fatalf("Bundle did not returned the existing Payload block: %v", err)
	}
}

// createNewBundle is used in the TestBundleCheckValid function and returns
// the Bundle with an ignored error. The error will be checked in this
// test case.
func createNewBundle(primary PrimaryBlock, canonicals []CanonicalBlock) Bundle {
	b, _ := NewBundle(primary, canonicals)

	return b
}

func TestBundleCheckValid(t *testing.T) {
	tests := []struct {
		b     Bundle
		valid bool
	}{
		// Administrative record
		{createNewBundle(
			NewPrimaryBlock(MustNotFragmented|AdministrativeRecordPayload,
				DtnNone(), DtnNone(), NewCreationTimestamp(42000000000000, 0), 3600),
			[]CanonicalBlock{
				NewCanonicalBlock(1, StatusReportBlock, NewPayloadBlock(nil))}),
			false},

		{createNewBundle(
			NewPrimaryBlock(MustNotFragmented|AdministrativeRecordPayload,
				DtnNone(), DtnNone(), NewCreationTimestamp(42000000000000, 0), 3600),
			[]CanonicalBlock{NewCanonicalBlock(1, 0, NewPayloadBlock(nil))}),
			true},

		// Block number (1) occurs twice
		{createNewBundle(
			NewPrimaryBlock(MustNotFragmented|AdministrativeRecordPayload,
				DtnNone(), DtnNone(), NewCreationTimestamp(42000000000000, 0), 3600),
			[]CanonicalBlock{
				NewCanonicalBlock(1, 0, NewPayloadBlock(nil)),
				NewCanonicalBlock(1, 0, NewPayloadBlock(nil))}),
			false},

		// Two Hop Count blocks
		{createNewBundle(
			NewPrimaryBlock(MustNotFragmented|AdministrativeRecordPayload,
				DtnNone(), DtnNone(), NewCreationTimestamp(42000000000000, 0), 3600),
			[]CanonicalBlock{
				NewCanonicalBlock(23, 0, NewHopCountBlock(23)),
				NewCanonicalBlock(24, 0, NewHopCountBlock(23)),
				NewCanonicalBlock(1, 0, NewPayloadBlock(nil))}),
			false},

		// Creation Time = 0, no Bundle Age block
		{createNewBundle(
			NewPrimaryBlock(MustNotFragmented|AdministrativeRecordPayload,
				DtnNone(), DtnNone(), NewCreationTimestamp(0, 0), 3600),
			[]CanonicalBlock{
				NewCanonicalBlock(2, 0, NewBundleAgeBlock(420)),
				NewCanonicalBlock(1, 0, NewPayloadBlock(nil))}),
			true},
		{createNewBundle(
			NewPrimaryBlock(MustNotFragmented|AdministrativeRecordPayload,
				DtnNone(), DtnNone(), NewCreationTimestamp(0, 0), 3600),
			[]CanonicalBlock{
				NewCanonicalBlock(1, 0, NewPayloadBlock(nil))}),
			false},
	}

	for _, test := range tests {
		if err := test.b.CheckValid(); (err == nil) != test.valid {
			t.Fatalf("Block validation failed: %v resulted in %v",
				test.b, err)
		}
	}
}

func TestBundleAddRemoveExtensionBlocks(t *testing.T) {
	primary := NewPrimaryBlock(0,
		MustNewEndpointID("dtn://dst/"),
		MustNewEndpointID("dtn://src/"),
		NewCreationTimestamp(DtnTimeEpoch, 0),
		60*60*1000000)
	canonicals := []CanonicalBlock{
		NewCanonicalBlock(2, 0, NewBundleAgeBlock(9001)),
		NewCanonicalBlock(1, 0, NewPayloadBlock([]byte("hello world"))),
	}

	b, err := NewBundle(primary, canonicals)
	if err != nil {
		t.Fatal(err)
	}

	if b.HasExtensionBlock(ExtBlockTypePreviousNodeBlock) {
		t.Fatalf("previous node block is present")
	}
	err = b.AddExtensionBlock(NewCanonicalBlock(0, 0, NewPreviousNodeBlock(MustNewEndpointID("dtn://prev/"))))
	if err != nil {
		t.Fatalf("adding PreviousNodeBlock resulted in error: %v", err)
	}
	if !b.HasExtensionBlock(ExtBlockTypePreviousNodeBlock) {
		t.Fatalf("previous node block is not present")
	}

	if previousNodeBlock, err := b.ExtensionBlock(ExtBlockTypePreviousNodeBlock); err != nil {
		t.Fatal(err)
	} else {
		b.RemoveExtensionBlockByBlockNumber(previousNodeBlock.BlockNumber)
	}
	if b.HasExtensionBlock(ExtBlockTypePreviousNodeBlock) {
		t.Fatalf("previous node block is present")
	}
}

func BenchmarkBundleSerializationCboring(b *testing.B) {
	var sizes = []int{0, 1024, 1048576, 10485760, 104857600}
	var crcs = []CRCType{CRCNo, CRC16, CRC32}

	for _, size := range sizes {
		for _, crc := range crcs {
			payload := make([]byte, size)

			rand.Seed(0)
			rand.Read(payload)

			primary := NewPrimaryBlock(
				0,
				MustNewEndpointID("dtn://dest/"),
				MustNewEndpointID("dtn://src/"),
				NewCreationTimestamp(DtnTimeEpoch, 0),
				60*60*1000000)

			canonicals := []CanonicalBlock{
				NewCanonicalBlock(2, 0, NewBundleAgeBlock(0)),
				NewCanonicalBlock(3, 0, NewPreviousNodeBlock(MustNewEndpointID("dtn://prev/"))),
				NewCanonicalBlock(1, 0, NewPayloadBlock(payload)),
			}

			bndl := MustNewBundle(primary, canonicals)
			bndl.SetCRCType(crc)

			b.Run(fmt.Sprintf("%d-%v", size, crc), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					if err := cboring.Marshal(&bndl, new(bytes.Buffer)); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkBundleDeserializationCboring(b *testing.B) {
	var sizes = []int{0, 1024, 1048576, 10485760, 104857600}
	var crcs = []CRCType{CRCNo, CRC16, CRC32}

	for _, size := range sizes {
		for _, crc := range crcs {
			payload := make([]byte, size)

			rand.Seed(0)
			rand.Read(payload)

			primary := NewPrimaryBlock(
				0,
				MustNewEndpointID("dtn://dest/"),
				MustNewEndpointID("dtn://src/"),
				NewCreationTimestamp(DtnTimeEpoch, 0),
				60*60*1000000)

			canonicals := []CanonicalBlock{
				NewCanonicalBlock(2, 0, NewBundleAgeBlock(0)),
				NewCanonicalBlock(3, 0, NewPreviousNodeBlock(MustNewEndpointID("dtn://prev/"))),
				NewCanonicalBlock(1, 0, NewPayloadBlock(payload)),
			}

			bndl := MustNewBundle(primary, canonicals)
			bndl.SetCRCType(crc)

			buff := new(bytes.Buffer)
			if err := cboring.Marshal(&bndl, buff); err != nil {
				b.Fatal(err)
			}
			data := buff.Bytes()

			b.Run(fmt.Sprintf("%d-%v", size, crc), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					tmpBuff := bytes.NewBuffer(data)
					tmpBndl := Bundle{}

					if err := cboring.Unmarshal(&tmpBndl, tmpBuff); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
