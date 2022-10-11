// SPDX-FileCopyrightText: 2020 Matthias Axel Kröll
// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/dtn7/cboring"
)

func TestBIBIOPHMACSHA2_VerifyTargets(t *testing.T) {
	b, bErr := Builder().
		CRC(CRC32).
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime(30 * time.Minute).
		PayloadBlock([]byte("hello world")).
		Build()
	if bErr != nil {
		t.Fatal(bErr)
	}

	privateKey := "dtnislove"

	payloadSecurityTarget, _ := b.ExtensionBlock(ExtBlockTypePayloadBlock)

	securityTargets := []uint64{payloadSecurityTarget.BlockNumber}

	shaVariant := HMAC256SHA256

	bib := NewBIBIOPHMACSHA2(&shaVariant, nil, nil, securityTargets, b.PrimaryBlock.SourceNode)

	eb := CanonicalBlock{
		BlockNumber:       0,
		BlockControlFlags: 0,
		CRCType:           CRCNo,
		CRC:               nil,
		Value:             bib,
	}

	err := b.AddExtensionBlock(eb)
	if err != nil {
		t.Fatal(err)
	}

	bibBlockAdded, _ := b.ExtensionBlock(bib.BlockTypeCode())

	err = bibBlockAdded.Value.(*BIBIOPHMACSHA2).SignTargets(b, bibBlockAdded.BlockNumber, []byte(privateKey))
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	if err := cboring.Marshal(bibBlockAdded, buff); err != nil {
		t.Fatal(err)
	}

	fmt.Println("BIB String:")
	fmt.Printf("%X\n", buff)

	fmt.Println("Bundle String:")

	buff = new(bytes.Buffer)
	if err := cboring.Marshal(&b, buff); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%X\n", buff)

	err = bibBlockAdded.Value.(*BIBIOPHMACSHA2).VerifyTargets(b, bibBlockAdded.BlockNumber, []byte(privateKey))
	if err != nil {
		t.Fatal(err)
	}

}
