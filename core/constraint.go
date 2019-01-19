package core

// Constraint is a retention constraint as defined in the subsections of the
// fifth chapter of draft-ietf-dtn-bpbis-12.
type Constraint int

const (
	// DispatchPending is assigned to a bundle if its dispatching is pending.
	DispatchPending Constraint = iota

	// ForwardPending is assigned to a bundle, if its forwarding is pending.
	ForwardPending Constraint = iota

	// ReassemblyPending is assigned to a fragmented bundle, if its reassembly is
	// pending.
	ReassemblyPending Constraint = iota
)
