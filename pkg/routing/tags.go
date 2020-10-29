// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

// Tag describes a bpv7.Bundle / BundleDescriptor.
type Tag uint64

const (
	_ Tag = iota

	// Incoming Bundle received from another node.
	Incoming

	// Outgoing Bundle originated at this node.
	Outgoing

	// Faulty Bundle, e.g., because some CheckFunc failed. This will lead to deletion.
	Faulty

	// ReassemblyPending is a fragmented, Incoming Bundle which needs to be reassembled before local delivery.
	ReassemblyPending

	// NoLocalAgent is an Incoming Bundle addressed to this node with an unregistered EndpointID.
	NoLocalAgent

	// Delivered Bundle to all recipients.
	Delivered
)
