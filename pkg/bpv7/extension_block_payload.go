// SPDX-FileCopyrightText: 2019, 2020, 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import "encoding/json"

// PayloadBlock implements the Bundle Protocol's Payload Block.
type PayloadBlock []byte

// BlockTypeCode must return a constant integer, indicating the block type code.
func (pb *PayloadBlock) BlockTypeCode() uint64 {
	return ExtBlockTypePayloadBlock
}

// BlockTypeName must return a constant string, this block's name.
func (pb *PayloadBlock) BlockTypeName() string {
	return "Payload Block"
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

// MarshalJSON writes the binary representation of a PayloadBlock.
//
// If this type does not implement the json.Marshaler, the CBOR encoding would be returned which might be misleading.
func (pb *PayloadBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(pb.Data())
}

// CheckValid returns an array of errors for incorrect data.
func (pb *PayloadBlock) CheckValid() error {
	return nil
}

// CheckContextValid has no implementation for a Payload Block.
func (pb *PayloadBlock) CheckContextValid(*Bundle) error {
	return nil
}
