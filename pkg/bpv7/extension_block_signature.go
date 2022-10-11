// SPDX-FileCopyrightText: 2020, 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"io"

	"github.com/dtn7/cboring"
	"github.com/hashicorp/go-multierror"
)

// SignatureBlock is a custom block to sign a Bundle's Primary Block and Payload Block via ed25519.
//
// The signature will be created for the concatenated CBOR representation of the Bundle's Primary Block and Payload
// Block using ed25519. Other blocks, like the Hop Count or Previous Node Block, might be altered on the way to the
// Bundle's destination. Therefore this signature is limited to these two blocks. It also follows that fragmented
// Bundles can neither be signed nor verified because the fragmentation offset is altered.
//
// To create a SignatureBlock, a Bundle with a PayloadBlock needs to exist first. Afterwards, one needs to create a
// SignatureBlock for this Bundle and attach it to the Bundle.
//
//	b, bErr := bpv7.Builder()./* ... */.Build()
//	sb, sbErr := bpv7.NewSignatureBlock(b, priv)
//	b.AddExtensionBlock(bpv7.NewCanonicalBlock(0, bpv7.ReplicateBlock|bpv7.DeleteBundle, sb))
//
// The block-type-specific data in a SignatureBlock MUST be represented as a CBOR array comprising two elements. These
// elements are firstly the PublicKey and secondly the Signature, both represented as a CBOR byte string. Both the array
// and the byte strings MUST be of a defined length, NOT indefinite-length items.
//
// Although this block is present in the bpv7 package, it is NOT specified in ietf-dtn-bpbis. It might be removed once
// ietf-dtn-bpsec is implemented.
type SignatureBlock struct {
	PublicKey []byte
	Signature []byte
}

// BlockTypeCode must return a constant integer, indicating the block type code.
func (s *SignatureBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeSignatureBlock
}

// BlockTypeName must return a constant string, this block's name.
func (s *SignatureBlock) BlockTypeName() string {
	return "Signature Block"
}

// signatureBundleData creates a Buffer of the Primary Block and Payload Block data, used as the message to be signed.
func signatureBundleData(b Bundle) (pbData bytes.Buffer, err error) {
	if err = cboring.Marshal(&b.PrimaryBlock, &pbData); err != nil {
		return
	}

	if pb, pbErr := b.ExtensionBlock(ExtBlockTypePayloadBlock); pbErr != nil {
		err = pbErr
	} else {
		err = cboring.Marshal(pb, &pbData)
	}

	return
}

// NewSignatureBlock for a Bundle from a private key.
func NewSignatureBlock(b Bundle, priv ed25519.PrivateKey) (s *SignatureBlock, err error) {
	if b.PrimaryBlock.BundleControlFlags.Has(IsFragment) {
		err = fmt.Errorf("fragmented Bundles cannot be signed")
		return
	}

	data, dataErr := signatureBundleData(b)
	if dataErr != nil {
		err = dataErr
		return
	}

	// ed25519.Sign panics for an invalid key size..
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from %v", r)
		}
	}()

	pub, pubOk := priv.Public().(ed25519.PublicKey)
	if !pubOk {
		err = fmt.Errorf("private key's public key is not an ed25519 public key (byte array)")
		return
	}

	s = &SignatureBlock{
		PublicKey: pub,
		Signature: ed25519.Sign(priv, data.Bytes()),
	}
	return
}

// CheckValid checks the field lengths for errors.
//
// This DOES NOT verify the signature. Therefore please use the Verify method.
func (s *SignatureBlock) CheckValid() (err error) {
	if l := len(s.PublicKey); l != ed25519.PublicKeySize {
		err = multierror.Append(err,
			fmt.Errorf("SignatureBlock: public key's length is %d, not required %d", l, ed25519.PublicKeySize))
	}

	if l := len(s.Signature); l != ed25519.SignatureSize {
		err = multierror.Append(err,
			fmt.Errorf("SignatureBlock: signature's length is %d, not required %d", l, ed25519.SignatureSize))
	}

	return
}

// CheckContextValid against its signature.
func (s *SignatureBlock) CheckContextValid(b *Bundle) error {
	// Cannot verify fragmented Bundles.
	if b.PrimaryBlock.BundleControlFlags.Has(IsFragment) {
		return nil
	}

	if !s.Verify(*b) {
		return fmt.Errorf("block verification failed")
	}

	return nil
}

// Verify the signature against a Bundle.
func (s *SignatureBlock) Verify(b Bundle) (valid bool) {
	if validErr := s.CheckValid(); validErr != nil {
		return false
	}

	if b.PrimaryBlock.BundleControlFlags.Has(IsFragment) {
		return false
	}

	data, dataErr := signatureBundleData(b)
	if dataErr != nil {
		return false
	}

	// ed25519.Verify panics for an invalid key size..
	defer func() {
		if recover() != nil {
			valid = false
		}
	}()

	return ed25519.Verify(s.PublicKey, data.Bytes(), s.Signature)
}

// MarshalCbor writes the CBOR representation of a SignatureBlock.
func (s *SignatureBlock) MarshalCbor(w io.Writer) error {
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	fields := []*[]byte{&s.PublicKey, &s.Signature}
	for _, field := range fields {
		if err := cboring.WriteByteString(*field, w); err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalCbor reads a CBOR representation of a SignatureBlock.
func (s *SignatureBlock) UnmarshalCbor(r io.Reader) error {
	if n, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if n != 2 {
		return fmt.Errorf("SignatureBlock: array has %d instead of 2 elements", n)
	}

	fields := []*[]byte{&s.PublicKey, &s.Signature}
	for _, field := range fields {
		if data, err := cboring.ReadByteString(r); err != nil {
			return err
		} else {
			*field = data
		}
	}

	return nil
}
