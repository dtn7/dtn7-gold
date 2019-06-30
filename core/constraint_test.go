package core

import (
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestBundlePackConstraints(t *testing.T) {
	var bndl, err = bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeEpoch, 0), 24*60*60),
		[]bundle.CanonicalBlock{
			bundle.NewCanonicalBlock(2, bundle.DeleteBundle, bundle.NewBundleAgeBlock(0)),
			bundle.NewCanonicalBlock(1, 0, bundle.NewPayloadBlock([]byte("hello world!"))),
		})
	if err != nil {
		t.Errorf("Bundle creation failed: %v", err)
	}

	var bp = NewBundlePack(&bndl)

	if len(bp.Constraints) != 0 {
		t.Errorf("Initial constraints map is not empty")
	}
	if bp.HasConstraints() {
		t.Errorf("Initial bundle pack has constraints")
	}
	if bp.HasConstraint(DispatchPending) {
		t.Errorf("Initial bundle pack has specific constraint")
	}

	bp.AddConstraint(DispatchPending)

	if len(bp.Constraints) != 1 {
		t.Errorf("Bundle pack has wrong length")
	}
	if !bp.HasConstraints() {
		t.Errorf("Bundle pack has no constraints after adding one")
	}
	if !bp.HasConstraint(DispatchPending) {
		t.Errorf("Bundle pack does not have set constraint")
	}
	if bp.HasConstraint(ForwardPending) {
		t.Errorf("Bundle pack has unknown constraint")
	}

	bp.RemoveConstraint(ForwardPending)

	if len(bp.Constraints) != 1 {
		t.Errorf("Bundle pack's length after removing unknown constraint")
	}

	bp.RemoveConstraint(DispatchPending)

	if bp.HasConstraints() {
		t.Errorf("Bundle pack has constraints after deletion")
	}
}
