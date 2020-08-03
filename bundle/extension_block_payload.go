// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bundle

// ExtBlockTypePayloadBlock is the block type code for a Payload Block.
const ExtBlockTypePayloadBlock uint64 = 1

// PayloadBlock implements the Bundle Protocol's Payload Block.
type PayloadBlock []byte

// BlockTypeCode must return a constant integer, indicating the block type code.
func (pb *PayloadBlock) BlockTypeCode() uint64 {
	return ExtBlockTypePayloadBlock
}

// NewPayloadBlock creates a new PayloadBlock with the given payload.
func NewPayloadBlock(data []byte) *PayloadBlock {
	pb := PayloadBlock(data)
	return &pb
}

// Data returns this PayloadBlock's payload.
func (pb *PayloadBlock) Data() []byte {
	return *pb
}

// MarshalBinary writes the binary representation of a PayloadBlock.
func (pb *PayloadBlock) MarshalBinary() ([]byte, error) {
	return *pb, nil
}

// UnmarshalBinary reads a binary PayloadBlock.
func (pb *PayloadBlock) UnmarshalBinary(data []byte) error {
	*pb = data
	return nil
}

// CheckValid returns an array of errors for incorrect data.
func (pb *PayloadBlock) CheckValid() error {
	return nil
}
