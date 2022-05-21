// SPDX-FileCopyrightText: 2019, 2020, 2022 Alvar Penning
// SPDX-FileCopyrightText: 2019, 2021 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"fmt"
	"io"

	"github.com/dtn7/cboring"
)

// DTLSRPeerData contains a peer's connection data
// This struct is placed in this location to avoid an import-loop with the routing package.
type DTLSRPeerData struct {
	// ID is the sending node's endpoint ID
	ID EndpointID
	// Timestamp is the time of the last update of the sending node's connection data.
	// When a node receives another's connection data, it should only update its view of the network
	// if this data is newer than the present one.
	Timestamp DtnTime
	// Peers is a representation of the node's connections.
	// Keys are the EndpointIDs of node which are or were connected to the sending node.
	// If the peer was currently connected when this block was sent, then the value will be 0.
	// If the connection to the peer was lost, the value will be the timestamp of the connection loss.
	Peers map[EndpointID]DtnTime
}

// ShouldReplace checks if one set of connection data should replace a different one.
// Currently only checks the timestamps.
func (pd DTLSRPeerData) ShouldReplace(other DTLSRPeerData) bool {
	return pd.Timestamp > other.Timestamp
}

// DTLSRBlock contains metadata used by the "Delay-Tolerant Link State Routing"-algorithm.
// It is a basic transmission-encapsulation of the DTLSRPeerData type,.
//
// NOTE:
// This is a custom extension block, and not part of the original bpv7 specification.
// It is currently assigned the block type code 193,
// which the specification sets aside for "private and/or experimental use"
//
// TODO: Turn this into an administrative record
type DTLSRBlock DTLSRPeerData

func NewDTLSRBlock(data DTLSRPeerData) *DTLSRBlock {
	newBlock := DTLSRBlock(data)
	return &newBlock
}

func (dtlsrb *DTLSRBlock) GetPeerData() DTLSRPeerData {
	return DTLSRPeerData(*dtlsrb)
}

func (dtlsrb *DTLSRBlock) BlockTypeCode() uint64 {
	return ExtBlockTypeDTLSRBlock
}

func (dtlsrb *DTLSRBlock) BlockTypeName() string {
	return "DTLSR Block"
}

func (dtlsrb *DTLSRBlock) CheckValid() error {
	return nil
}

func (dtlsrb *DTLSRBlock) CheckContextValid(*Bundle) error {
	return nil
}

func (dtlsrb *DTLSRBlock) MarshalCbor(w io.Writer) error {
	// start with the (apparently) required outer array
	if err := cboring.WriteArrayLength(3, w); err != nil {
		return err
	}

	// write our own endpoint ID
	if err := cboring.Marshal(&dtlsrb.ID, w); err != nil {
		return err
	}

	// write the Timestamp
	if err := cboring.WriteUInt(uint64(dtlsrb.Timestamp), w); err != nil {
		return err
	}

	// write the peer data array header
	if err := cboring.WriteMapPairLength(uint64(len(dtlsrb.Peers)), w); err != nil {
		return err
	}

	// write the actual data
	for peerID, timestamp := range dtlsrb.Peers {
		if err := cboring.Marshal(&peerID, w); err != nil {
			return err
		}
		if err := cboring.WriteUInt(uint64(timestamp), w); err != nil {
			return err
		}
	}

	return nil
}

func (dtlsrb *DTLSRBlock) UnmarshalCbor(r io.Reader) error {
	// read the (apparently) required outer array
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 3 {
		return fmt.Errorf("expected 3 fields, got %d", l)
	}

	// read endpoint ID
	id := EndpointID{}
	if err := cboring.Unmarshal(&id, r); err != nil {
		return err
	} else {
		dtlsrb.ID = id
	}

	// read the Timestamp
	if timestamp, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		dtlsrb.Timestamp = DtnTime(timestamp)
	}

	var lenData uint64

	// read length of data array
	lenData, err := cboring.ReadMapPairLength(r)
	if err != nil {
		return err
	}

	// read the actual data
	peers := make(map[EndpointID]DtnTime)
	var i uint64
	for i = 0; i < lenData; i++ {
		peerID := EndpointID{}
		if err := cboring.Unmarshal(&peerID, r); err != nil {
			return err
		}

		timestamp, err := cboring.ReadUInt(r)
		if err != nil {
			return err
		}

		peers[peerID] = DtnTime(timestamp)
	}

	dtlsrb.Peers = peers

	return nil
}
