// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bundle

import (
	"io"

	"github.com/dtn7/cboring"
)

// BundleAgeBlock implements the Bundle Protocol's Bundle Age Block.
type BundleAgeBlock uint64

// BlockTypeCode must return a constant integer, indicating the block type code.
func (bab *BundleAgeBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeBundleAgeBlock
}

// NewBundleAgeBlock creates a new BundleAgeBlock for the given milliseconds.
func NewBundleAgeBlock(ms uint64) *BundleAgeBlock {
	bab := BundleAgeBlock(ms)
	return &bab
}

// Age returns the age in milliseconds.
func (bab *BundleAgeBlock) Age() uint64 {
	return uint64(*bab)
}

// Increment with an offset in milliseconds and return the new time.
func (bab *BundleAgeBlock) Increment(offset uint64) uint64 {
	newBabVal := uint64(*bab) + offset
	*bab = BundleAgeBlock(newBabVal)
	return newBabVal
}

// MarshalCbor writes a CBOR representation for a Bundle Age Block.
func (bab *BundleAgeBlock) MarshalCbor(w io.Writer) error {
	return cboring.WriteUInt(uint64(*bab), w)
}

// UnmarshalCbor reads the CBOR representation for a Bundle Age Block.
func (bab *BundleAgeBlock) UnmarshalCbor(r io.Reader) error {
	if us, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		*bab = BundleAgeBlock(us)
		return nil
	}
}

// CheckValid returns an array of errors for incorrect data.
func (bab *BundleAgeBlock) CheckValid() error {
	return nil
}
