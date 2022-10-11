// SPDX-FileCopyrightText: 2020 Matthias Axel Kröll
// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"fmt"
	"hash"

	"io"

	"crypto/hmac"

	"github.com/dtn7/cboring"
)

// BIBIOPHMACSHA2 implements the BPSEC Block Integrity Block (BIB)
type BIBIOPHMACSHA2 struct {
	Asb AbstractSecurityBlock
}

// BIB-HMAC-SHA2 Security Parameters
const (
	SecParIdBIBIOPHMACSHA2ShaVariant uint64 = 1

	SecParIdBIBIOPHMACSHA2WrappedKey uint64 = 2

	SecParIdBIBIOPHMACSHA2IntegrityScopeFlags uint64 = 3
)

// SecConResultIDBIBIOPHMACSHA2ExpectedHMAC BIB-IOP-HMAC-SHA2 ResultID
const SecConResultIDBIBIOPHMACSHA2ExpectedHMAC uint64 = 1

// SHA Variant Parameter Values for BIB-IOP-HMAC-SHA2.
const (
	HMAC256SHA256 uint64 = 5 // Default
	HMAC384SHA384 uint64 = 6
	HMAC512SHA512 uint64 = 7
)

// IntegrityScopeFlags are used to show how broadly  the concept of integrity is being applied, e.g.
// what to include in the IPPT draft-ietf-dtn-bpsec-interop-sc-02#section-3.2
// Default 0x7
const (
	BIBIOPHMACDefaultIntegrityScopeFlags uint16 = 0b111
	PrimaryBlockFlagBIBIOPHMAC           uint16 = 0b001
	TargetHeaderFlagBIBIOPHMAC           uint16 = 0b010
	SecurityHeaderFlagBIBIOPHMAC         uint16 = 0b100
)

// BlockTypeCode BlockTypeCode must return a constant integer, indicating the block type code.
func (bib *BIBIOPHMACSHA2) BlockTypeCode() uint64 {
	return ExtBlockTypeBlockIntegrityBlock
}

// BlockTypeName must return a constant string, this block's name.
func (bib *BIBIOPHMACSHA2) BlockTypeName() string {
	return SecConNameBIBIOPHMACSHA
}

// MarshalCbor writes a CBOR representation for a Bundle Integrity Block.
func (bib *BIBIOPHMACSHA2) MarshalCbor(w io.Writer) error {
	return bib.Asb.MarshalCbor(w)
}

// UnmarshalCbor writes a CBOR representation for a Bundle Integrity Block
func (bib *BIBIOPHMACSHA2) UnmarshalCbor(r io.Reader) error {
	return bib.Asb.UnmarshalCbor(r)
}

// CheckValid returns an array of errors for incorrect data.
func (bib *BIBIOPHMACSHA2) CheckValid() error {
	if err := bib.Asb.CheckValid(); err != nil {
		return err
	}

	return nil
}

// CheckContextValid  TODO: Check BSPEC stuff
func (bib *BIBIOPHMACSHA2) CheckContextValid(*Bundle) error {

	return bib.CheckValid()
}

// NewBIBIOPHMACSHA2

func NewBIBIOPHMACSHA2(shaVariant *uint64, wrappedKey *[]byte, integrityScopeFlags *uint16, securityTargets []uint64, securitySource EndpointID) *BIBIOPHMACSHA2 {

	securityContextParametersPresentFlag := uint64(0)

	if shaVariant != nil || wrappedKey != nil || integrityScopeFlags != nil {
		securityContextParametersPresentFlag = 1
	}

	var securityContextParameters []IDValueTuple

	if shaVariant != nil {
		securityContextParameters = append(securityContextParameters, &IDValueTupleUInt64{
			id:    SecParIdBIBIOPHMACSHA2ShaVariant,
			value: *shaVariant,
		})
	}

	if wrappedKey != nil {
		securityContextParameters = append(securityContextParameters, &IDValueTupleByteString{
			id:    SecParIdBIBIOPHMACSHA2WrappedKey,
			value: *wrappedKey,
		})
	}

	if integrityScopeFlags != nil {
		securityContextParameters = append(securityContextParameters, &IDValueTupleUInt64{
			id:    SecParIdBIBIOPHMACSHA2IntegrityScopeFlags,
			value: uint64(*integrityScopeFlags),
		})
	}

	securityResults := make([]TargetSecurityResults, len(securityTargets))

	for i, target := range securityTargets {
		securityResults[i] = TargetSecurityResults{
			securityTarget: target,
			results:        []IDValueTuple{},
		}
	}

	return &BIBIOPHMACSHA2{Asb: AbstractSecurityBlock{
		SecurityTargets:                      securityTargets,
		SecurityContextID:                    SecConIdentBIBIOPHMACSHA,
		SecurityContextParametersPresentFlag: securityContextParametersPresentFlag,
		SecuritySource:                       securitySource,
		SecurityContextParameters:            securityContextParameters,
		SecurityResults:                      securityResults,
	}}

}

// prepareIPPT constructs the "Integrity Protected Plain Text" using the process defined in bpsec-default-sc-11 3.7.
func (bib *BIBIOPHMACSHA2) prepareIPPT(b Bundle, securityTargetBlockNumber uint64, bibBlockNumber uint64) (ippt *bytes.Buffer, err error) {
	ippt = &bytes.Buffer{}

	// Default Value for IntegrityScopeFlag, used if the optional security parameter is not present.
	integrityScopeFlag := BIBIOPHMACDefaultIntegrityScopeFlags

	securityTargetBlock, err := b.GetExtensionBlockByBlockNumber(securityTargetBlockNumber)
	if err != nil {
		return nil, err
	}

	// Find the integrity scope flag security parameter if present.
	if bib.Asb.HasSecurityContextParametersPresentContextFlag() {
		for _, scp := range bib.Asb.SecurityContextParameters {
			if scp.ID() == SecParIdBIBIOPHMACSHA2IntegrityScopeFlags {
				integrityScopeFlag = uint16(scp.Value().(uint64))
			}
		}
	}

	// 1. The canonical form of the IPPT starts as the CBOR encoding of the integrity scope flag.
	if err = cboring.WriteUInt(uint64(integrityScopeFlag), ippt); err != nil {
		return nil, err
	}

	// 2. If the primary block flag of the integrity scope flags is set to 1,
	// then a canonical form of the bundle's primary block MUST be calculated
	// and the result appended to the IPPT.
	if integrityScopeFlag&PrimaryBlockFlagBIBIOPHMAC == PrimaryBlockFlagBIBIOPHMAC {
		if err = b.PrimaryBlock.MarshalCbor(ippt); err != nil {
			return nil, err
		}
	}

	// 3. If the target header flag of the integrity scope flags is set to 1,
	// then the canonical form of the block type code, block number,
	// and block processing control flags associated with the security
	// target MUST be calculated and, in that order, appended to the IPPT.
	if integrityScopeFlag&TargetHeaderFlagBIBIOPHMAC == TargetHeaderFlagBIBIOPHMAC {
		if err = cboring.WriteUInt(securityTargetBlock.TypeCode(), ippt); err != nil {
			return nil, err
		}

		if err = cboring.WriteUInt(securityTargetBlock.BlockNumber, ippt); err != nil {
			return nil, err
		}

		if err = cboring.WriteUInt(uint64(securityTargetBlock.BlockControlFlags), ippt); err != nil {
			return nil, err
		}
	}

	// 4. If the security header flag of the integrity scope flags is set to 1,
	// then the canonical form of the block type code, block number,
	// and block processing control flags associated with the  BIB MUST be calculated and,
	// in that order, appended to the IPPT.
	if integrityScopeFlag&SecurityHeaderFlagBIBIOPHMAC == SecurityHeaderFlagBIBIOPHMAC {

		if err = cboring.WriteUInt(bib.BlockTypeCode(), ippt); err != nil {
			return nil, err
		}
		if err = cboring.WriteUInt(bibBlockNumber, ippt); err != nil {
			return nil, err
		}

		var bibCanonicalBlock *CanonicalBlock
		bibCanonicalBlock, err = b.ExtensionBlock(bib.BlockTypeCode())
		if err != nil {
			return nil, err
		}

		if err = cboring.WriteUInt(bibCanonicalBlock.BlockNumber, ippt); err != nil {
			return nil, err
		}

		if err = cboring.WriteUInt(uint64(bibCanonicalBlock.BlockControlFlags), ippt); err != nil {
			return nil, err
		}

	}

	// 5. The canonical form of the security target block-type-specific
	// data MUST be calculated and appended to the IPPT.
	if err = GetExtensionBlockManager().WriteBlock(securityTargetBlock.Value, ippt); err != nil {
		return nil, err
	}

	return ippt, nil
}

// calculateSecurityResultValues computes the results of the BIP-IOP-HMAC-SHA2 security operation for all Security Targets of the BIP, depending on the SHAVariant SecurityContextParameter
func (bib *BIBIOPHMACSHA2) calculateSecurityResultValues(b Bundle, bibBlockNumber uint64, privateKey []byte) (securityResults *[]*[]byte, err error) {
	shaVariantParameter := func() *uint64 {
		for _, scp := range bib.Asb.SecurityContextParameters {
			if scp.ID() == SecParIdBIBIOPHMACSHA2ShaVariant {
				scpValue := scp.Value().(uint64)
				return &scpValue
			}
		}
		return nil
	}()

	var shaVariant func() hash.Hash

	switch *shaVariantParameter {
	case HMAC384SHA384:
		shaVariant = sha512.New384

	case HMAC512SHA512:
		shaVariant = sha512.New
	default:
		// Use default is shaVariantParameter is not present e.g. nil or set to HMAC256SHA256
		shaVariant = sha256.New
	}

	h := hmac.New(shaVariant, privateKey)

	results := make([]*[]byte, len(bib.Asb.SecurityTargets))

	for i, securityTargetBlockNumber := range bib.Asb.SecurityTargets {
		ippt, err := bib.prepareIPPT(b, securityTargetBlockNumber, bibBlockNumber)
		if err != nil {
			return nil, err
		}

		_, err = h.Write(ippt.Bytes())
		if err != nil {
			return nil, err
		}

		targetResult := h.Sum(nil)

		results[i] = &targetResult

		h.Reset()

	}
	return &results, nil
}

func (bib *BIBIOPHMACSHA2) SignTargets(b Bundle, bibBlockNumber uint64, privateKey []byte) (err error) {
	securityResultValues, err := bib.calculateSecurityResultValues(b, bibBlockNumber, privateKey)
	if err != nil {
		return err
	}

	for i, resultValue := range *securityResultValues {
		bib.Asb.SecurityResults[i].results = append(bib.Asb.SecurityResults[i].results, &IDValueTupleByteString{
			id:    SecConResultIDBIBIOPHMACSHA2ExpectedHMAC,
			value: *resultValue,
		})
	}

	return
}

func (bib *BIBIOPHMACSHA2) VerifyTargets(b Bundle, bibBlockNumber uint64, privateKey []byte) (err error) {
	securityResultValues, err := bib.calculateSecurityResultValues(b, bibBlockNumber, privateKey)
	if err != nil {
		return err
	}

	for i, resultValue := range *securityResultValues {
		var resultToVerify []byte

		// Probably this is not necessary to check for the Result ID, because BIB-IOP-HMAC-SHA2 only has one specified Result ID.
		// But it makes sense as a check.
		for _, targetResults := range bib.Asb.SecurityResults[i].results {
			if targetResults.ID() == SecConResultIDBIBIOPHMACSHA2ExpectedHMAC {
				resultToVerify = targetResults.Value().([]byte)
			}
		}

		if resultToVerify == nil {
			return fmt.Errorf("could not find SecurityResult with ResultID %d, for SecurityTarget with Blocknumber %d in BIB-IOP-HMAC-SHA2 with Blocknumber %d", SecConResultIDBIBIOPHMACSHA2ExpectedHMAC, bib.Asb.SecurityTargets[i], bibBlockNumber)
		}

		if subtle.ConstantTimeCompare(*resultValue, resultToVerify) != 1 {
			return fmt.Errorf("Could not verify HMAC for Securitytarget with Blocknumber %d in BIB-IOP-HMAC-SHA2 with Blocknumber %d, Found MAC: %X  Expected MAC: %X", bib.Asb.SecurityTargets[i], bibBlockNumber, *resultValue, resultToVerify)
		}

	}
	return
}
