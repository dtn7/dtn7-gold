// SPDX-FileCopyrightText: 2020 Matthias Axel Kröll
// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/dtn7/cboring"
)

// BCBIOPAESGCM implements the BPSEC Block Con Block (BIB)
type BCBIOPAESGCM struct {
	Asb AbstractSecurityBlock
}

// BIB-HMAC-SHA2 Security Parameters
const (
	SecParIdBCBIOPAESGCMIV uint64 = 1

	SecParIdBCBIOPAESGCMAESVariant uint64 = 2

	SecParIdBCBIOPAESGCMWrappedKey uint64 = 3

	SecParIdBCBIOPAESGCMAADScopeFlags uint64 = 4
)

// SecConResultIDBCBIOPAESGCMAuthenticationTag SecConResultIDBCBIOPAESGCMExpectedHMAC BCB-IOP-AES-GCM ResultID
const SecConResultIDBCBIOPAESGCMAuthenticationTag uint64 = 1

// AES Variant Parameter Values for BCB-IOP-AES-GCM.
const (
	A128GCM uint64 = 1
	A256GCM uint64 = 3 // Default
)

// RFC 9173 4.3.4
// This optional parameter contains a series of flags that describe what
// information is to be included with the block-type-specific data of
// the security target as part of additional authenticated data (AAD).
//
// Bits in this field represent additional information to be included
// when generating an integrity signature over the security target.

// Default 0x7
const (
	DefaultAADScopeFlags           uint16 = 0b111
	PrimaryBlockFlagBCBIOPAESGCM   uint16 = 0b001
	TargetHeaderFlagBCBIOPAESGCM   uint16 = 0b010
	SecurityHeaderFlagBCBIOPAESGCM uint16 = 0b100
)

// BlockTypeCode BlockTypeCode must return a constant integer, indicating the block type code.
func (bcb *BCBIOPAESGCM) BlockTypeCode() uint64 {
	return ExtBlockTypeBlockConfidentialityBlock
}

// BlockTypeName must return a constant string, this block's name.
func (bcb *BCBIOPAESGCM) BlockTypeName() string {
	return SecConNameBCBIOPAESGCM
}

// MarshalCbor writes a CBOR representation for a Bundle Integrity Block.
func (bcb *BCBIOPAESGCM) MarshalCbor(w io.Writer) error {
	return bcb.Asb.MarshalCbor(w)
}

// UnmarshalCbor writes a CBOR representation for a Bundle Integrity Block
func (bcb *BCBIOPAESGCM) UnmarshalCbor(r io.Reader) error {
	return bcb.Asb.UnmarshalCbor(r)
}

// CheckValid returns an array of errors for incorrect data.
func (bcb *BCBIOPAESGCM) CheckValid() error {
	if err := bcb.Asb.CheckValid(); err != nil {
		return err
	}

	return nil
}

// CheckContextValid  TODO: Check BSPEC stuff
func (bcb *BCBIOPAESGCM) CheckContextValid(*Bundle) error {

	return bcb.CheckValid()
}

// NewBCBIOPAESGCM creates a new BCB-IOP-AES-GCM block
func NewBCBIOPAESGCM(aesVariant *uint64, wrappedKey *[]byte, AADScopeFlags *uint16, securityTarget uint64, securitySource EndpointID) *BCBIOPAESGCM {

	securityContextParametersPresentFlag := uint64(0)

	if aesVariant != nil || wrappedKey != nil || AADScopeFlags != nil {
		securityContextParametersPresentFlag = 1
	}

	var securityContextParameters []IDValueTuple

	if aesVariant != nil {
		securityContextParameters = append(securityContextParameters, &IDValueTupleUInt64{
			id:    SecParIdBCBIOPAESGCMAESVariant,
			value: *aesVariant,
		})
	}

	if wrappedKey != nil {
		securityContextParameters = append(securityContextParameters, &IDValueTupleByteString{
			id:    SecParIdBCBIOPAESGCMWrappedKey,
			value: *wrappedKey,
		})
	}

	if AADScopeFlags != nil {
		securityContextParameters = append(securityContextParameters, &IDValueTupleUInt64{
			id:    SecParIdBCBIOPAESGCMAADScopeFlags,
			value: uint64(*AADScopeFlags),
		})
	}

	securityResults := make([]TargetSecurityResults, 1)

	securityResults[0] = TargetSecurityResults{
		securityTarget: securityTarget,
		results:        []IDValueTuple{},
	}

	return &BCBIOPAESGCM{Asb: AbstractSecurityBlock{
		SecurityTargets:                      []uint64{securityTarget},
		SecurityContextID:                    SecConIdentBCBIOPAESGCM,
		SecurityContextParametersPresentFlag: securityContextParametersPresentFlag,
		SecuritySource:                       securitySource,
		SecurityContextParameters:            securityContextParameters,
		SecurityResults:                      securityResults,
	}}

}

// extractPlainText extracts the plaintext used during encryption according to RFC9173 4.7.1
func (bcb *BCBIOPAESGCM) extractPlainText(securityTargetBlock *CanonicalBlock) (plainText *bytes.Buffer, err error) {
	// 1. Create a buffer for the plaintext
	plainText = new(bytes.Buffer)

	// 2. Get Plaintext from Payloadblock
	payLoadBlock := securityTargetBlock.Value.(*PayloadBlock)
	_, err = plainText.Write(payLoadBlock.Data())
	if err != nil {
		return nil, err
	}
	return
}

// prepareAAD constructs the "Additional Authenticated Data" using the process defined in RFC9173 4.7.2
func (bcb *BCBIOPAESGCM) prepareAAD(b Bundle, securityTargetBlock *CanonicalBlock, bcbBlockNumber uint64) (aad *bytes.Buffer, err error) {
	aad = &bytes.Buffer{}

	// Default Value for aadScopeFlag, used if the optional security parameter is not present.
	aadScopeFlag := DefaultAADScopeFlags

	// Find the integrity scope flag security parameter if present.
	if bcb.Asb.HasSecurityContextParametersPresentContextFlag() {
		for _, scp := range bcb.Asb.SecurityContextParameters {
			if scp.ID() == SecParIdBCBIOPAESGCMAADScopeFlags {
				aadScopeFlag = uint16(scp.Value().(uint64))
			}
		}
	}

	// 1. The canonical form of the AAD starts as the CBOR encoding of the
	// AAD scope flags in which all unset flags, reserved bits, and unassigned bits have been set to 0.
	if err = cboring.WriteUInt(uint64(aadScopeFlag), aad); err != nil {
		return nil, err
	}

	// 2. If the primary block flag of the AAD scope flags is set to 1,
	// then a canonical form of the bundle's primary block MUST be calculated
	// and the result appended to the AAD.
	if aadScopeFlag&PrimaryBlockFlagBCBIOPAESGCM == PrimaryBlockFlagBCBIOPAESGCM {
		if err = b.PrimaryBlock.MarshalCbor(aad); err != nil {
			return nil, err
		}
	}

	// 3. If the target header flag of the AAD scope flags is set to 1,
	// then the canonical form of the block type code, block number,
	// and block processing control flags associated with the security
	// target MUST be calculated and, in that order, appended to the AAD.
	if aadScopeFlag&TargetHeaderFlagBCBIOPAESGCM == TargetHeaderFlagBCBIOPAESGCM {
		if err = cboring.WriteUInt(securityTargetBlock.TypeCode(), aad); err != nil {
			return nil, err
		}

		if err = cboring.WriteUInt(securityTargetBlock.BlockNumber, aad); err != nil {
			return nil, err
		}

		if err = cboring.WriteUInt(uint64(securityTargetBlock.BlockControlFlags), aad); err != nil {
			return nil, err
		}
	}

	// 4. If the security header flag of the AAD scope flags is set to 1,
	// then the canonical form of the block type code, block number, and
	// block processing control flags associated with the BCB MUST be
	// calculated and, in that order, appended to the AAD.
	if aadScopeFlag&SecurityHeaderFlagBCBIOPAESGCM == SecurityHeaderFlagBCBIOPAESGCM {

		if err = cboring.WriteUInt(bcb.BlockTypeCode(), aad); err != nil {
			return nil, err
		}
		if err = cboring.WriteUInt(bcbBlockNumber, aad); err != nil {
			return nil, err
		}

		var bcbCanonicalBlock *CanonicalBlock
		bcbCanonicalBlock, err = b.ExtensionBlock(bcb.BlockTypeCode())
		if err != nil {
			return nil, err
		}

		if err = cboring.WriteUInt(bcbCanonicalBlock.BlockNumber, aad); err != nil {
			return nil, err
		}

		if err = cboring.WriteUInt(uint64(bcbCanonicalBlock.BlockControlFlags), aad); err != nil {
			return nil, err
		}

	}

	return aad, nil
}

// computeAuthenticationTagAndCipherText computes the results of the BCB-IOP-AES-GCM security operation for the BCP Security Target, depending on the AESVariant SecurityContextParameter
func (bcb *BCBIOPAESGCM) computeAuthenticationTagAndCipherText(plainText *bytes.Buffer, aad *bytes.Buffer, privateKey []byte) (cipherText []byte, authenticationTag []byte, err error) {

	// Check if WrappedKey is present and throw unimplemented error if not.
	wrappedKey := func() *[]byte {
		for _, scp := range bcb.Asb.SecurityContextParameters {
			if scp.ID() == SecParIdBCBIOPAESGCMWrappedKey {
				scpValue := scp.Value().([]byte)
				return &scpValue
			}
		}
		return nil
	}()
	if wrappedKey != nil {
		return nil, nil, fmt.Errorf("wrapped key not implemented")
	}

	err = checkKeyLengthAgainstAESVariantParameter(bcb, privateKey)
	if err != nil {
		return nil, nil, err
	}

	// Get the AES IV
	aesIVParameter := func() *[]byte {
		for _, scp := range bcb.Asb.SecurityContextParameters {
			if scp.ID() == SecParIdBCBIOPAESGCMIV {
				scpValue := scp.Value().([]byte)
				return &scpValue
			}
		}
		return nil
	}()

	// AES
	block, err := aes.NewCipher(privateKey)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	// Generate a random IV if none is provided.
	if aesIVParameter == nil {
		iv := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return nil, nil, err
		}
		aesIVParameter = &iv

		// Add the IV to the Security Context Parameters
		bcb.Asb.SecurityContextParameters = append(bcb.Asb.SecurityContextParameters, &IDValueTupleByteString{SecParIdBCBIOPAESGCMIV, iv})

	}

	// Encrypt
	fullCipherText := gcm.Seal(nil, *aesIVParameter, plainText.Bytes(), aad.Bytes())

	// Split cipherText and authenticationTag
	cipherText = fullCipherText[0 : len(fullCipherText)-gcm.Overhead()]
	authenticationTag = fullCipherText[len(fullCipherText)-gcm.Overhead():]

	return
}

// checkKeyLengthAgainstAESVariantParameter checks if the key length is valid for the AESVariant SecurityContextParameter
func checkKeyLengthAgainstAESVariantParameter(bcb *BCBIOPAESGCM, privateKey []byte) (err error) {
	// Get AES variant
	aesVariantParameter := func() *uint64 {
		for _, scp := range bcb.Asb.SecurityContextParameters {
			if scp.ID() == SecParIdBCBIOPAESGCMAESVariant {
				scpValue := scp.Value().(uint64)
				return &scpValue
			}
		}
		return nil
	}()

	// Check if the AES variant is supported
	switch len(privateKey) {
	case 16:
		if aesVariantParameter != nil && *aesVariantParameter != A128GCM {
			return fmt.Errorf("AES-128 variant %d and Keylength %d does not match", *aesVariantParameter, len(privateKey))
		}
	case 32:
		if aesVariantParameter != nil && *aesVariantParameter != A256GCM {
			return fmt.Errorf("AES-256 variant %d and Keylength %d does not match", *aesVariantParameter, len(privateKey))
		}
	default:
		return fmt.Errorf("keylength %d is not supported", len(privateKey))
	}
	return nil
}

// EncryptTarget encrypts the target block using the BCB-IOP-AES-GCM security operation.
func (bcb *BCBIOPAESGCM) EncryptTarget(b Bundle, bcbBlockNumber uint64, privateKey []byte) (err error) {

	// Check if the target is a supported payload block
	securityTargetBlock, err := b.GetExtensionBlockByBlockNumber(bcb.Asb.SecurityTargets[0])
	if err != nil {
		return err
	}
	if securityTargetBlock.Value.BlockTypeCode() != ExtBlockTypePayloadBlock {
		return fmt.Errorf("unsupported security target block type code %d, %s", securityTargetBlock.Value.BlockTypeCode(), securityTargetBlock.Value.BlockTypeName())
	}

	// Remove CRC if present
	if securityTargetBlock.CRCType != CRCNo {
		securityTargetBlock.CRCType = CRCNo
		securityTargetBlock.CRC = nil
	}

	// Extract the security target plaintext
	plainText, err := bcb.extractPlainText(securityTargetBlock)
	if err != nil {
		return err
	}

	// Prepare the AAD
	aad, err := bcb.prepareAAD(b, securityTargetBlock, bcbBlockNumber)
	if err != nil {
		return err
	}

	// Compute the cipherText and authenticationTag
	cipherText, authenticationTag, err := bcb.computeAuthenticationTagAndCipherText(plainText, aad, privateKey)
	if err != nil {
		return err
	}

	// Set the cipherText as the payload
	securityTargetBlock.Value = NewPayloadBlock(cipherText)

	// Set the authenticationTag as security result
	bcb.Asb.SecurityResults[0].results = append(bcb.Asb.SecurityResults[0].results, &IDValueTupleByteString{
		id:    SecConResultIDBCBIOPAESGCMAuthenticationTag,
		value: authenticationTag,
	})

	return nil

}

// DecryptTarget decrypts the payload of the security target block and verifies the authentication tag
func (bcb *BCBIOPAESGCM) DecryptTarget(b Bundle, bcbBlockNumber uint64, privateKey []byte) (err error) {

	// Check if the target is a supported payload block
	securityTargetBlock, err := b.GetExtensionBlockByBlockNumber(bcb.Asb.SecurityTargets[0])
	if err != nil {
		return err
	}

	if securityTargetBlock.Value.BlockTypeCode() != ExtBlockTypePayloadBlock {
		return fmt.Errorf("unsupported security target block type code %d, %s", securityTargetBlock.Value.BlockTypeCode(), securityTargetBlock.Value.BlockTypeName())
	}

	// Decrypt and Authenticate
	plainText, err := bcb.decryptAndAuthenticate(b, securityTargetBlock, bcbBlockNumber, privateKey)
	if err != nil {
		return err
	}

	// Set the plainText as the payload
	securityTargetBlock.Value = NewPayloadBlock(plainText)

	// Set CRC
	securityTargetBlock.CRCType = CRC32

	return
}

// Decrypt and authenticate the security target
func (bcb *BCBIOPAESGCM) decryptAndAuthenticate(b Bundle, targetBlock *CanonicalBlock, number uint64, key []byte) (plainText []byte, err error) {

	// Get the AES IV
	aesIVParameter := func() *[]byte {
		for _, scp := range bcb.Asb.SecurityContextParameters {
			if scp.ID() == SecParIdBCBIOPAESGCMIV {
				scpValue := scp.Value().([]byte)
				return &scpValue
			}
		}
		return nil
	}()

	// Get the authentication tag
	authenticationTag := func() *[]byte {
		for _, scp := range bcb.Asb.SecurityResults[0].results {
			if scp.ID() == SecConResultIDBCBIOPAESGCMAuthenticationTag {
				scpValue := scp.Value().([]byte)
				return &scpValue
			}
		}
		return nil
	}()

	// Check if the authentication tag is present
	if authenticationTag == nil {
		return nil, fmt.Errorf("authentication tag is missing")
	}

	// Check if the AES IV is present
	if aesIVParameter == nil {
		return nil, fmt.Errorf("AES IV Security Parameter is missing")
	}

	// Get the cipherText
	cipherText := targetBlock.Value.(*PayloadBlock).Data()

	// Prepare the AAD
	aad, err := bcb.prepareAAD(b, targetBlock, number)
	if err != nil {
		return nil, err
	}

	// Check key length
	err = checkKeyLengthAgainstAESVariantParameter(bcb, key)
	if err != nil {
		return nil, err
	}

	// AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	fullChipherText := append(cipherText, *authenticationTag...)

	// Decrypt
	plainText, err = gcm.Open(nil, *aesIVParameter, fullChipherText, aad.Bytes())
	if err != nil {
		return nil, err
	}

	return plainText, nil
}
