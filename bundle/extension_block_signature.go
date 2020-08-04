// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bundle

import (
	"bytes"
	"crypto/ed25519"
	"fmt"

	"github.com/dtn7/cboring"
	"github.com/hashicorp/go-multierror"
)

// SignatureBlock is a custom block to sign a Bundle's Payload Block via ed25519.
//
// Although this block is present in the bundle package, it is not specified in ietf-dtn-bpbis. It will probably be
// removed once ietf-dtn-bpsec is implemented.
type SignatureBlock struct {
	PublicKey []byte
	Signature []byte
}

// BlockTypeCode must return a constant integer, indicating the block type code.
func (s *SignatureBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeSignatureBlock
}

// signaturePayloadData creates a Buffer of a Bundle's Payload Block data, used as the message to be signed.
func signaturePayloadData(b Bundle) (pbData bytes.Buffer, err error) {
	if pb, pbErr := b.ExtensionBlock(ExtBlockTypePayloadBlock); pbErr != nil {
		err = pbErr
	} else {
		err = cboring.Marshal(pb, &pbData)
	}
	return
}

// NewSignatureBlock for a Bundle's PayloadBlock with a private key.
func NewSignatureBlock(b Bundle, priv ed25519.PrivateKey) (s *SignatureBlock, err error) {
	pbData, pbDataErr := signaturePayloadData(b)
	if pbDataErr != nil {
		err = pbDataErr
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
		Signature: ed25519.Sign(priv, pbData.Bytes()),
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

// Verify the signature against a Bundle's Payload Block.
func (s *SignatureBlock) Verify(b Bundle) (valid bool) {
	if validErr := s.CheckValid(); validErr != nil {
		return false
	}

	pbData, pbDataErr := signaturePayloadData(b)
	if pbDataErr != nil {
		return false
	}

	// ed25519.Verify panics for an invalid key size..
	defer func() {
		if recover() != nil {
			valid = false
		}
	}()

	return ed25519.Verify(s.PublicKey, pbData.Bytes(), s.Signature)
}

// TODO: cboring.CborMarshaler
