package core

import (
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestBundlePackUpdateBundleAge(t *testing.T) {
	var bndl, err = bundle.Builder().
		Source("dtn:src").
		Destination("dtn:dest").
		CreationTimestampEpoch().
		Lifetime("60m").
		BundleCtrlFlags(bundle.MustNotFragmented).
		BundleAgeBlock(0).
		PayloadBlock([]byte("hello world!")).
		Build()
	if err != nil {
		t.Errorf("Bundle creation failed: %v", err)
	}

	var bp = NewBundlePack(&bndl)

	time.Sleep(50 * time.Millisecond)
	age, err := bp.UpdateBundleAge()
	if err != nil {
		t.Errorf("No age block was found")
	}

	ageBlock, _ := bp.Bundle.ExtensionBlock(bundle.ExtBlockTypeBundleAgeBlock)
	ageFromBlock := ageBlock.Value.(*bundle.BundleAgeBlock)

	if age != ageFromBlock.Age() {
		t.Errorf("Returning value differs from block")
	}

	if age < 35000 || age > 65000 {
		t.Errorf("Bundle's Age Block drifts to much: %v", ageBlock)
	}
}
