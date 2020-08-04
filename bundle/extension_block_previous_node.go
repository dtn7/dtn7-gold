// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bundle

import (
	"io"

	"github.com/dtn7/cboring"
)

// PreviousNodeBlock implements the Bundle Protocol's Previous Node Block.
type PreviousNodeBlock EndpointID

// BlockTypeCode must return a constant integer, indicating the block type code.
func (pnb *PreviousNodeBlock) BlockTypeCode() uint64 {
	return ExtBlockTypePreviousNodeBlock
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

// CheckValid returns an array of errors for incorrect data.
func (pnb *PreviousNodeBlock) CheckValid() error {
	return EndpointID(*pnb).CheckValid()
}
