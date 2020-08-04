// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bundle

import (
	"bytes"
	"crypto/ed25519"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/dtn7/cboring"
)

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

	sb, sbErr := NewSignatureBlock(b1, priv)
	if sbErr != nil {
		t.Fatal(sbErr)
	}

	b1.AddExtensionBlock(NewCanonicalBlock(0, ReplicateBlock|DeleteBundle, sb))
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

	if pb, pbErr := b2.PayloadBlock(); pbErr != nil {
		t.Fatal(pbErr)
	} else {
		pb.Value.(*PayloadBlock).Data()[0] = 0
	}

	if sbCan.Value.(*SignatureBlock).Verify(b2) {
		t.Fatal("SignatureBlock with invalid PayloadBlock succeeded")
	}
}
