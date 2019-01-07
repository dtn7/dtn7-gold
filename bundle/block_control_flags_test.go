package bundle

import (
	"testing"
)

func TestBlockControlFlagsHas(t *testing.T) {
	var cf BlockControlFlags = ReplicateBlock |
		DeleteBundle

	if !cf.Has(ReplicateBlock) {
		t.Error("cf has no ReplicateBlock-flag even when it was set")
	}

	if cf.Has(RemoveBlock) {
		t.Error("cf has RemoveBlock-flag which was not set")
	}
}

func TestBlockControlFlagsCheckValid(t *testing.T) {
	tests := []struct {
		cf    BlockControlFlags
		valid bool
	}{
		{0, true},
		{ReplicateBlock, true},
		{ReplicateBlock | DeleteBundle, true},
		{ReplicateBlock | 0x80, false},
		{0x40 | 0x20, false},
	}

	for _, test := range tests {
		if err := test.cf.checkValid(); (err == nil) != test.valid {
			t.Errorf("BlockControlFlags validation failed: %v resulted in %v",
				test.cf, err)
		}
	}
}
