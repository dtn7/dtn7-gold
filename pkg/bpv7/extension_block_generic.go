// SPDX-FileCopyrightText: 2019, 2020, 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

// GenericExtensionBlock is a dummy ExtensionBlock to cover for unknown or unregistered ExtensionBlocks.
type GenericExtensionBlock struct {
	data     []byte
	typeCode uint64
}

// NewGenericExtensionBlock creates a new GenericExtensionBlock from some payload and a block type code.
func NewGenericExtensionBlock(data []byte, typeCode uint64) *GenericExtensionBlock {
	return &GenericExtensionBlock{
		data:     data,
		typeCode: typeCode,
	}
}

// MarshalBinary writes a binary representation of this block.
func (geb *GenericExtensionBlock) MarshalBinary() ([]byte, error) {
	return geb.data, nil
}

// UnmarshalBinary reads a binary representation of a generic block.
func (geb *GenericExtensionBlock) UnmarshalBinary(data []byte) error {
	geb.data = data
	return nil
}

// CheckValid returns an array of errors for incorrect data.
func (geb *GenericExtensionBlock) CheckValid() error {
	// We have zero knowledge about this block.
	// Thus, who are we to judge someone else's block?
	return nil
}

// CheckContextValid has no implementation for a GenericExtensionBlock.
func (geb *GenericExtensionBlock) CheckContextValid(*Bundle) error {
	return nil
}

// BlockTypeCode must return a constant integer, indicating the block type code.
func (geb *GenericExtensionBlock) BlockTypeCode() uint64 {
	return geb.typeCode
}

// BlockTypeName must return a constant string, this block's name.
func (geb *GenericExtensionBlock) BlockTypeName() string {
	return "N/A"
}
