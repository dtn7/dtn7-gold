// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package soclp

import (
	"fmt"
	"io"

	"github.com/dtn7/cboring"

	"github.com/dtn7/dtn7-go/bundle"
)

// IdentityMessage is the type specific message for peer identification based on a bundle.EndpointID.
type IdentityMessage struct {
	NodeID bundle.EndpointID
}

// NewIdentityMessage based on a bundle.EndpointID.
func NewIdentityMessage(nodeId bundle.EndpointID) *IdentityMessage {
	return &IdentityMessage{NodeID: nodeId}
}

// Type code of an IdentityMessage is always 0.
func (im *IdentityMessage) Type() uint64 {
	return MsgIdentity
}

func (im *IdentityMessage) String() string {
	return fmt.Sprintf("IdentityMessage(%v)", im.NodeID.String())
}

// MarshalCbor serializes the EndpointID's CBOR representation.
func (im *IdentityMessage) MarshalCbor(w io.Writer) error {
	return cboring.Marshal(&im.NodeID, w)
}

// UnmarshalCbor a CBOR-represented EndpointID back to an IdentityMessage.
func (im *IdentityMessage) UnmarshalCbor(r io.Reader) error {
	return cboring.Unmarshal(&im.NodeID, r)
}
