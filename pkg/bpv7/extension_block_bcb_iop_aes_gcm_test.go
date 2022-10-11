// SPDX-FileCopyrightText: 2020 Matthias Axel Kröll
// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/dtn7/cboring"
)

func Test(t *testing.T) {
	payLoadBeforeEncryption := []byte("hello world!")

	b, bErr := Builder().
		CRC(CRC32).
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime(30 * time.Minute).
		PayloadBlock(payLoadBeforeEncryption).
		Build()
	if bErr != nil {
		t.Fatal(bErr)
	}

	privateKey := "dtnislovedtnislovedtnislovedtnis"

	payloadSecurityTarget, _ := b.ExtensionBlock(ExtBlockTypePayloadBlock)

	aesVariant := A256GCM

	bcb := NewBCBIOPAESGCM(&aesVariant, nil, nil, payloadSecurityTarget.BlockNumber, b.PrimaryBlock.SourceNode)

	eb := CanonicalBlock{
		BlockNumber:       0,
		BlockControlFlags: 0,
		CRCType:           CRCNo,
		CRC:               nil,
		Value:             bcb,
	}

	err := b.AddExtensionBlock(eb)
	if err != nil {
		t.Fatal(err)
	}

	bcbBlockAdded, _ := b.ExtensionBlock(bcb.BlockTypeCode())

	err = bcbBlockAdded.Value.(*BCBIOPAESGCM).EncryptTarget(b, bcbBlockAdded.BlockNumber, []byte(privateKey))
	if err != nil {
		t.Fatal(err)
	}

	err = bcbBlockAdded.Value.(*BCBIOPAESGCM).DecryptTarget(b, bcbBlockAdded.BlockNumber, []byte(privateKey))
	if err != nil {
		t.Fatal(err)
	}

	payLoadBlockBlockAfterDecrypt, _ := b.ExtensionBlock(ExtBlockTypePayloadBlock)

	payloadAfterDecrypt := payLoadBlockBlockAfterDecrypt.Value.(*PayloadBlock).Data()

	println("Payload after decrypt: " + string(payloadAfterDecrypt))
	println("Payload before decrypt: " + string(payLoadBeforeEncryption))

	buff := new(bytes.Buffer)
	if err := cboring.Marshal(bcbBlockAdded, buff); err != nil {
		t.Fatal(err)
	}

	fmt.Println("BCB String:")
	fmt.Printf("%X\n", buff)

	fmt.Println("Bundle String:")

	buff = new(bytes.Buffer)
	if err := cboring.Marshal(&b, buff); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%X\n", buff)

	if !bytes.Equal(payloadAfterDecrypt, payLoadBeforeEncryption) {
		t.Fatal("Decrypted payload does not match original payload")
	}

}

func TestBCBBlockCbor(t *testing.T) {
	ep, _ := NewEndpointID("dtn://test/")
	aesVariant := A256GCM
	tests := []struct {
		bcb1 BCBIOPAESGCM
	}{
		{
			bcb1: BCBIOPAESGCM{
				Asb: AbstractSecurityBlock{
					SecurityTargets:                      []uint64{0},
					SecurityContextID:                    SecConIdentBCBIOPAESGCM,
					SecurityContextParametersPresentFlag: 0x1,
					SecuritySource:                       ep,
					SecurityContextParameters: []IDValueTuple{
						//&IDValueTupleByteString{
						//
						//	id:    SecParIdBCBIOPAESGCMWrappedKey,
						//	value: []byte{37, 35, 92, 90, 54},
						//},
						&IDValueTupleUInt64{

							id:    SecParIdBCBIOPAESGCMAESVariant,
							value: aesVariant,
						},
					},
					SecurityResults: []TargetSecurityResults{{
						securityTarget: 0,
						results: []IDValueTuple{&IDValueTupleByteString{

							id:    0,
							value: []byte{37, 35, 92, 90, 54, 37, 35, 92, 90, 54},
						}},
					}},
				},
			},
		},
	}

	for _, test := range tests {
		buff := new(bytes.Buffer)
		if err := cboring.Marshal(&test.bcb1, buff); err != nil {
			t.Fatal(err)
		}

		bcb2 := BCBIOPAESGCM{}
		if err := cboring.Unmarshal(&bcb2, buff); err != nil {
			t.Fatalf("CBOR decoding failed: %v", err)
		}

		if !reflect.DeepEqual(test.bcb1, bcb2) {
			t.Fatalf("Abstract Security Blocs differ:\n%v\n%v", test.bcb1, bcb2)
		}
	}
}
