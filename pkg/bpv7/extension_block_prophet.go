// SPDX-FileCopyrightText: 2019, 2021 Markus Sommer
// SPDX-FileCopyrightText: 2020, 2022 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"io"

	"github.com/dtn7/cboring"
)

// ProphetBlock contains metadata used by the "PRoPHET" routing algorithm.
//
// Each key-value pair represents the encounter probability between the sending node and other nodes in the network.
//
// NOTE:
// This is a custom extension block, and not part of the original bpv7 specification.
// It is currently assigned the block type code 194,
// which the specification sets aside for "private and/or experimental use"
//
// TODO: Turn this into an administrative record
type ProphetBlock map[EndpointID]float64

func NewProphetBlock(data map[EndpointID]float64) *ProphetBlock {
	newBlock := ProphetBlock(data)
	return &newBlock
}

func (pBlock *ProphetBlock) GetPredictabilities() map[EndpointID]float64 {
	return *pBlock
}

func (pBlock *ProphetBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeProphetBlock
}

func (pBlock *ProphetBlock) BlockTypeName() string {
	return "Prophet Routing Block"
}

func (pBlock ProphetBlock) CheckValid() error {
	return nil
}

func (pBlock ProphetBlock) CheckContextValid(*Bundle) error {
	return nil
}

func (pBlock *ProphetBlock) MarshalCbor(w io.Writer) error {
	// write the peer data array header
	if err := cboring.WriteMapPairLength(uint64(len(*pBlock)), w); err != nil {
		return err
	}

	// write the actual data
	for peerID, pred := range *pBlock {
		if err := cboring.Marshal(&peerID, w); err != nil {
			return err
		}
		if err := cboring.WriteFloat64(pred, w); err != nil {
			return err
		}
	}

	return nil
}

func (pBlock *ProphetBlock) UnmarshalCbor(r io.Reader) error {
	var lenData uint64

	// read length of data array
	lenData, err := cboring.ReadMapPairLength(r)
	if err != nil {
		return err
	}

	// read the actual data
	predictability := make(map[EndpointID]float64)
	var i uint64
	for i = 0; i < lenData; i++ {
		peerID := EndpointID{}
		if err := cboring.Unmarshal(&peerID, r); err != nil {
			return err
		}

		pred, err := cboring.ReadFloat64(r)
		if err != nil {
			return err
		}

		predictability[peerID] = pred
	}

	*pBlock = predictability

	return nil
}
