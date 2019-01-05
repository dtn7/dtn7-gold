package bpa

import (
	"testing"
)

func TestBlockControlFlagsHas(t *testing.T) {
	var cf BlockControlFlags = BlckCFBlockMustBeReplicatedInEveryFragment |
		BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed

	if !cf.Has(BlckCFBlockMustBeReplicatedInEveryFragment) {
		t.Error("cf has no BlckCFBlockMustBeReplicatedInEveryFragment-flag even when it was set")
	}

	if cf.Has(BlckCFBlockMustBeRemovedFromBundleIfItCannotBeProcessed) {
		t.Error("cf has BlckCFBlockMustBeRemovedFromBundleIfItCannotBeProcessed-flag which was not set")
	}
}

func TestBlockControlFlagsCheckValid(t *testing.T) {
	tests := []struct {
		cf    BlockControlFlags
		valid bool
	}{
		{0, true},
		{BlckCFBlockMustBeReplicatedInEveryFragment, true},
		{BlckCFBlockMustBeReplicatedInEveryFragment | BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed, true},
		{BlckCFBlockMustBeReplicatedInEveryFragment | 0x80, false},
		{0x40 | 0x20, false},
	}

	for _, test := range tests {
		if err := test.cf.checkValid(); (err == nil) != test.valid {
			t.Errorf("BlockControlFlags validation failed: %v resulted in %v",
				test.cf, err)
		}
	}
}
