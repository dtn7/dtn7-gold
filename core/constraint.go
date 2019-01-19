package core

import "github.com/geistesk/dtn7/bundle"

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

// BundlePack is a tuple of a bundle and a set of constraints used in the
// process of delivering this bundle.
type BundlePack struct {
	Bundle      bundle.Bundle
	Constraints map[Constraint]bool
}

// NewBundlePack returns a BundlePack for the given bundle.
func NewBundlePack(b bundle.Bundle) BundlePack {
	return BundlePack{
		Bundle:      b,
		Constraints: make(map[Constraint]bool),
	}
}

// HasConstraint returns true if the given constraint contains.
func (bp BundlePack) HasConstraint(c Constraint) bool {
	_, ok := bp.Constraints[c]
	return ok
}

// HasConstraints returns true if any constraint exists.
func (bp BundlePack) HasConstraints() bool {
	return len(bp.Constraints) != 0
}

// AddConstraint adds the given constraint.
func (bp *BundlePack) AddConstraint(c Constraint) {
	bp.Constraints[c] = true
}

// RemoveConstraint removes the given constraint.
func (bp *BundlePack) RemoveConstraint(c Constraint) {
	delete(bp.Constraints, c)
}
