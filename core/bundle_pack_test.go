package core

import (
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestBundlePackUpdateBundleAge(t *testing.T) {
	var bndl, err = bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented,
			bundle.MustNewEndpointID("dtn:dest"),
			bundle.MustNewEndpointID("dtn:src"),
			bundle.NewCreationTimestamp(bundle.DtnTimeEpoch, 0), 24*60*60),
		[]bundle.CanonicalBlock{
			bundle.NewBundleAgeBlock(1, bundle.DeleteBundle, 0),
			bundle.NewPayloadBlock(0, []byte("hello world!")),
		})
	if err != nil {
		t.Errorf("Bundle creation failed: %v", err)
	}

	var bp = NewBundlePack(&bndl)

	time.Sleep(50 * time.Millisecond)
	age, err := bp.UpdateBundleAge()
	if err != nil {
		t.Errorf("No age block was found")
	}

	ageBlock, _ := bp.Bundle.ExtensionBlock(bundle.BundleAgeBlock)
	ageFromBlock := ageBlock.Data.(uint64)

	if age != ageFromBlock {
		t.Errorf("Returning value differs from block")
	}

	if age < 35000 || age > 65000 {
		t.Errorf("Bundle's Age Block drifts to much: %v", ageBlock)
	}
}
