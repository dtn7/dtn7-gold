// SPDX-FileCopyrightText: 2018, 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import "github.com/dtn7/cboring"

// block is an interface for blocks present in a bundle. Both PrimaryBlock
// and the CanonicalBlock have the CRC-field in common.
type block interface {
	// Valid is extended, which requires the checkValid() method.
	Valid

	// CborMarshaler is extended for the MarshalCbor and UnmarshalCbor methods.
	cboring.CborMarshaler

	// HasCRC returns if the CRCType indicates a CRC present for this block.
	// In this case the CRC value should become relevant.
	HasCRC() bool

	// GetCRCType returns the CRCType of this block.
	GetCRCType() CRCType

	// SetCRCType sets the CRC type.
	SetCRCType(CRCType)
}
