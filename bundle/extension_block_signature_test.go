// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bundle

import (
	"bytes"
	"crypto/ed25519"
	"testing"
	"time"
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
		t.Fatalf("SignatureBlock cannot be verified")
	}
}
