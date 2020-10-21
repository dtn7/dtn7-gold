// SPDX-FileCopyrightText: 2018, 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"testing"
)

func TestBlockControlFlagsHas(t *testing.T) {
	var cf = ReplicateBlock | DeleteBundle

	if !cf.Has(ReplicateBlock) {
		t.Error("cf has no ReplicateBlock-flag even when it was set")
	}

	if cf.Has(RemoveBlock) {
		t.Error("cf has RemoveBlock-flag which was not set")
	}
}

func TestBlockControlFlagsCheckValid(t *testing.T) {
	// Since dtn-bpbpis-24 _all_ bit masks are valid Block Processing Control Flags.
	// The `valid` check might become useful again.
	tests := []struct {
		cf    BlockControlFlags
		valid bool
	}{
		{0, true},
		{ReplicateBlock, true},
		{ReplicateBlock | DeleteBundle, true},
		{ReplicateBlock | 0x80, true},
		{0x40 | 0x20, true},
	}

	for _, test := range tests {
		if err := test.cf.CheckValid(); (err == nil) != test.valid {
			t.Errorf("BlockControlFlags validation failed: %v resulted in %v",
				test.cf, err)
		}
	}
}
