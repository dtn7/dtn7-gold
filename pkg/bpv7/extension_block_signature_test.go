// SPDX-FileCopyrightText: 2020, 2022 Alvar Penning
// SPDX-FileCopyrightText: 2022 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"crypto/ed25519"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/dtn7/cboring"
)

// testSignatureBlockRandBytes is a util function to create len pseudo random bytes.
func testSignatureBlockRandBytes(seed int64, len int, t *testing.T) []byte {
	random := rand.New(rand.NewSource(seed))
	b := make([]byte, len)
	if n, err := random.Read(b); err != nil {
		t.Fatal(err)
	} else if n != len {
		t.Fatalf("generated %d instead of %d bytes", n, len)
	}
	return b
}

func TestSignatureBlockCheckValid(t *testing.T) {
	tests := []struct {
		name      string
		publicKey []byte
		signature []byte
		wantErr   bool
	}{
		{
			name:      "good",
			publicKey: make([]byte, ed25519.PublicKeySize),
			signature: make([]byte, ed25519.SignatureSize),
			wantErr:   false,
		},
		{
			name:      "invalid public key length",
			publicKey: make([]byte, ed25519.PublicKeySize+1),
			signature: make([]byte, ed25519.SignatureSize),
			wantErr:   true,
		},
		{
			name:      "invalid signature length",
			publicKey: make([]byte, ed25519.PublicKeySize),
			signature: make([]byte, ed25519.SignatureSize+1),
			wantErr:   true,
		},
		{
			name:      "everything is wrong",
			publicKey: make([]byte, ed25519.PublicKeySize+1),
			signature: make([]byte, ed25519.SignatureSize+1),
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := &SignatureBlock{
				PublicKey: test.publicKey,
				Signature: test.signature,
			}

			if err := s.CheckValid(); (err != nil) != test.wantErr {
				t.Fatalf("CheckValid() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestSignatureBlockVerifySimple(t *testing.T) {
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

	pub, priv, ed25519KeyErr := ed25519.GenerateKey(nil)
	if ed25519KeyErr != nil {
		t.Fatal(ed25519KeyErr)
	}

	sb, sbErr := NewSignatureBlock(b, priv)
	if sbErr != nil {
		t.Fatal(sbErr)
	}

	if !bytes.Equal(pub, sb.PublicKey) {
		t.Fatalf("SignatureBlock's public key %x differs from %x", sb.PublicKey, pub)
	}

	if err := sb.CheckValid(); err != nil {
		t.Fatal(err)
	}

	if !sb.Verify(b) {
		t.Fatal("SignatureBlock cannot be verified")
	}
}

func TestSignatureBlockCborSimple(t *testing.T) {
	sb1 := &SignatureBlock{
		PublicKey: testSignatureBlockRandBytes(1, ed25519.PublicKeySize, t),
		Signature: testSignatureBlockRandBytes(1, ed25519.SignatureSize, t),
	}
	sb2 := &SignatureBlock{}

	var buff bytes.Buffer
	if err := cboring.Marshal(sb1, &buff); err != nil {
		t.Fatal(err)
	}

	if err := cboring.Unmarshal(sb2, &buff); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(sb1, sb2) {
		t.Fatalf("SignatureBlock differs: %v != %v", sb1, sb2)
	}
}

func TestSignatureBlockIntegration(t *testing.T) {
	// By default, the SignatureBlock is not registered in the singleton ExtensionManager
	if regErr := GetExtensionBlockManager().Register(&SignatureBlock{}); regErr != nil {
		t.Fatal(regErr)
	}
	defer GetExtensionBlockManager().Unregister(&SignatureBlock{})

	b1, bErr := Builder().
		CRC(CRC32).
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampNow().
		Lifetime(30000).
		PayloadBlock([]byte("hello world")).
		Build()
	if bErr != nil {
		t.Fatal(bErr)
	}

	pub, priv, ed25519KeyErr := ed25519.GenerateKey(nil)
	if ed25519KeyErr != nil {
		t.Fatal(ed25519KeyErr)
	}

	sb, sbErr := NewSignatureBlock(b1, priv)
	if sbErr != nil {
		t.Fatal(sbErr)
	}

	err := b1.AddExtensionBlock(NewCanonicalBlock(0, ReplicateBlock|DeleteBundle, sb))
	if err != nil {
		t.Fatalf("Adding ExtensionBlock caused error: %v", err)
	}

	b1.SetCRCType(CRC32)

	var buff bytes.Buffer
	var b2 Bundle
	if buffErr := b1.MarshalCbor(&buff); buffErr != nil {
		t.Fatal(buffErr)
	} else if buffErr := b2.UnmarshalCbor(&buff); buffErr != nil {
		t.Fatal(buffErr)
	}

	sbCan, sbCanErr := b2.ExtensionBlock(ExtBlockTypeSignatureBlock)
	if sbCanErr != nil {
		t.Fatal(sbCanErr)
	}

	if _, isSignatureBlock := sbCan.Value.(*SignatureBlock); !isSignatureBlock {
		t.Fatalf("Block is not of type *SignatureBlock, but %T", sbCan.Value)
	}

	if sbPub := sbCan.Value.(*SignatureBlock).PublicKey; !bytes.Equal(sbPub, pub) {
		t.Fatalf("SignatureBlock's public key %x differs from %x", sbPub, pub)
	}

	if !sbCan.Value.(*SignatureBlock).Verify(b2) {
		t.Fatal("SignatureBlock cannot be verified")
	}

	// Alter the Primary Block
	prim := &b2.PrimaryBlock
	primTmp := prim.Lifetime

	prim.Lifetime = 0
	if sbCan.Value.(*SignatureBlock).Verify(b2) {
		t.Fatal("SignatureBlock with invalid PrimaryBlock succeeded")
	}

	prim.Lifetime = primTmp
	if !sbCan.Value.(*SignatureBlock).Verify(b2) {
		t.Fatal("SignatureBlock with fixed PrimaryBlock failed")
	}

	// Alter the Payload Block
	if pb, pbErr := b2.PayloadBlock(); pbErr != nil {
		t.Fatal(pbErr)
	} else {
		tmp := pb.Value.(*PayloadBlock).Data()[0]

		pb.Value.(*PayloadBlock).Data()[0] = 0
		if sbCan.Value.(*SignatureBlock).Verify(b2) {
			t.Fatal("SignatureBlock with invalid PayloadBlock succeeded")
		}

		pb.Value.(*PayloadBlock).Data()[0] = tmp
		if !sbCan.Value.(*SignatureBlock).Verify(b2) {
			t.Fatal("SignatureBlock with fixed PayloadBlock failed")
		}
	}
}

func TestSignatureBlockFragmentSimple(t *testing.T) {
	b1, b1Err := Builder().
		CRC(CRC32).
		Source("dtn://src/").
		Destination("dtn://dst/").
		CreationTimestampTime(time.Unix(4000000000, 0)).
		Lifetime(30 * time.Minute).
		PayloadBlock(testSignatureBlockRandBytes(23, 1024, t)).
		Build()
	if b1Err != nil {
		t.Fatal(b1Err)
	}

	_, priv, ed25519KeyErr := ed25519.GenerateKey(nil)
	if ed25519KeyErr != nil {
		t.Fatal(ed25519KeyErr)
	}

	sb, sbErr := NewSignatureBlock(b1, priv)
	if sbErr != nil {
		t.Fatal(sbErr)
	}

	cb := NewCanonicalBlock(0, ReplicateBlock|DeleteBundle, sb)
	cb.SetCRCType(CRC32)
	err := b1.AddExtensionBlock(cb)
	if err != nil {
		t.Fatalf("Error adding ExtensionBlock: %v", err)
	}

	bs, bsErr := b1.Fragment(256)
	if bsErr != nil {
		t.Fatal(bsErr)
	}

	for _, frag := range bs {
		if fragCb, fragErr := frag.ExtensionBlock(ExtBlockTypeSignatureBlock); fragErr != nil {
			t.Fatal(fragErr)
		} else if !reflect.DeepEqual(cb.Value, fragCb.Value) {
			t.Fatalf("Signature Block in fragment differs: %v != %v", sb, fragCb.Value)
		} else if fragCb.Value.(*SignatureBlock).Verify(frag) {
			t.Fatal("Positive verification on fragment")
		}
	}

	b2, b2Err := ReassembleFragments(bs)
	if b2Err != nil {
		t.Fatal(b2Err)
	}

	// Marshal both Bundles to ensure the presence of a CRC value
	if err := cboring.Marshal(&b1, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := cboring.Marshal(&b2, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	if b2Sb, b2SbErr := b2.ExtensionBlock(ExtBlockTypeSignatureBlock); b2SbErr != nil {
		t.Fatal(b2SbErr)
	} else if !b2Sb.Value.(*SignatureBlock).Verify(b2) {
		t.Fatal("Verification failed")
	}
}
