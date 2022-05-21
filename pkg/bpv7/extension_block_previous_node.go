// SPDX-FileCopyrightText: 2019, 2020, 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/dtn7/cboring"
)

// PreviousNodeBlock implements the Bundle Protocol's Previous Node Block.
type PreviousNodeBlock EndpointID

// BlockTypeCode must return a constant integer, indicating the block type code.
func (pnb *PreviousNodeBlock) BlockTypeCode() uint64 {
	return ExtBlockTypePreviousNodeBlock
}

// BlockTypeName must return a constant string, this block's name.
func (pnb *PreviousNodeBlock) BlockTypeName() string {
	return "Previous Node Block"
}

// NewPreviousNodeBlock creates a new Previous Node Block for an Endpoint ID.
func NewPreviousNodeBlock(prev EndpointID) *PreviousNodeBlock {
	pnb := PreviousNodeBlock(prev)
	return &pnb
}

// Endpoint returns this Previous Node Block's Endpoint ID.
func (pnb *PreviousNodeBlock) Endpoint() EndpointID {
	return EndpointID(*pnb)
}

// MarshalCbor writes the CBOR representation of a PreviousNodeBlock.
func (pnb *PreviousNodeBlock) MarshalCbor(w io.Writer) error {
	endpoint := EndpointID(*pnb)
	return cboring.Marshal(&endpoint, w)
}

// UnmarshalCbor reads a CBOR representation of a PreviousNodeBlock.
func (pnb *PreviousNodeBlock) UnmarshalCbor(r io.Reader) error {
	endpoint := EndpointID{}
	if err := cboring.Unmarshal(&endpoint, r); err != nil {
		return err
	} else {
		*pnb = PreviousNodeBlock(endpoint)
		return nil
	}
}

// MarshalJSON writes the JSON representation of a PreviousNodeBlock.
func (pnb *PreviousNodeBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(pnb.Endpoint())
}

// CheckValid returns an array of errors for incorrect data.
func (pnb *PreviousNodeBlock) CheckValid() error {
	return EndpointID(*pnb).CheckValid()
}

// CheckContextValid that there is at most one Previous Node Block.
func (pnb *PreviousNodeBlock) CheckContextValid(b *Bundle) error {
	cb, err := b.ExtensionBlock(ExtBlockTypePreviousNodeBlock)

	if err != nil {
		return err
	} else if cb.Value != pnb {
		return fmt.Errorf("PreviousNodeBlock's pointer differs, %p != %p", cb.Value, pnb)
	} else {
		return nil
	}
}
