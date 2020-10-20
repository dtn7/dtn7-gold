// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

// Constraint is a retention constraint as defined in the subsections of the fifth chapter of dtn-bpbis.
type Constraint int

const (
	// DispatchPending is assigned to a bpv7 if its dispatching is pending.
	DispatchPending Constraint = iota

	// ForwardPending is assigned to a bundle if its forwarding is pending.
	ForwardPending Constraint = iota

	// ReassemblyPending is assigned to a fragmented bundle if its reassembly is
	// pending.
	ReassemblyPending Constraint = iota

	// Contraindicated is assigned to a bundle if it could not be delivered and
	// was moved to the contraindicated stage. This Constraint was not defined
	// in dtn-bpbis, but seemed reasonable for this implementation.
	Contraindicated Constraint = iota

	// LocalEndpoint is assigned to a bundle after delivery to a local endpoint.
	// This constraint demands storage until the endpoint removes this constraint.
	LocalEndpoint Constraint = iota
)

func (c Constraint) String() string {
	switch c {
	case DispatchPending:
		return "dispatch pending"

	case ForwardPending:
		return "forwarding pending"

	case ReassemblyPending:
		return "reassembly pending"

	case Contraindicated:
		return "contraindicated"

	case LocalEndpoint:
		return "local endpoint"

	default:
		return "unknown"
	}
}
