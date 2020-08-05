// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bundle

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"io"

	"github.com/dtn7/cboring"
	"github.com/hashicorp/go-multierror"
)

// SignatureBlock is a custom block to sign a Bundle's Primary and Payload Block via ed25519.
//
// Although this block is present in the bundle package, it is NOT specified in ietf-dtn-bpbis. It will probably be
// removed once ietf-dtn-bpsec is implemented.
//
// To create a SignatureBlock, a Bundle with a PayloadBlock needs to exist first. Afterwards, one needs to create a
// SignatureBlock for this Bundle and attach it to the Bundle.
//
// 	sb, sbErr := NewSignatureBlock(b, priv)
// 	if sbErr != nil { ... }
// 	b.AddExtensionBlock(NewCanonicalBlock(0, ReplicateBlock|DeleteBundle, sb))
//
// The ed25519 signature will be created for the concatenated CBOR representation of the Bundle's Primary Block and
// Payload Block.
//
// The block-type-specific data in a SignatureBlock MUST be represented as a CBOR array comprising two elements. These
// two elements are firstly the PublicKey and secondly the Signature, both represented as a CBOR byte string. Both the
// array and the byte strings MUST be of a defined length, NOT indefinite-length items.
type SignatureBlock struct {
	PublicKey []byte
	Signature []byte
}

// BlockTypeCode must return a constant integer, indicating the block type code.
func (s *SignatureBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeSignatureBlock
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

// Verify the signature against a Bundle.
func (s *SignatureBlock) Verify(b Bundle) (valid bool) {
	if validErr := s.CheckValid(); validErr != nil {
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
