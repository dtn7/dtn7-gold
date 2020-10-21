// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"fmt"
	"sort"
	"testing"
)

func TestCanonicalBlockNumberSortLess(t *testing.T) {
	var canonicals canonicalBlockNumberSort = []CanonicalBlock{
		NewCanonicalBlock(2, 0, nil), // 0
		NewCanonicalBlock(3, 0, nil), // 1
		NewCanonicalBlock(4, 0, nil), // 2
		NewCanonicalBlock(5, 0, nil), // 3
		NewCanonicalBlock(6, 0, nil), // 4
		NewCanonicalBlock(1, 0, nil), // 5
		NewCanonicalBlock(9, 0, nil), // 6
	}

	tests := []struct {
		i, j int
		want bool
	}{
		{0, 1, true},
		{0, 2, true},
		{3, 4, true},
		{5, 0, false},
		{0, 5, true},
		{5, 6, false},
		{6, 5, true},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%d,%d", test.i, test.j), func(t *testing.T) {
			if got := canonicals.Less(test.i, test.j); got != test.want {
				t.Fatalf("Less() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCanonicalBlockNumberSort(t *testing.T) {
	// Shuffled array of CanonicalBlocks with block numbers from 1 to 7.
	// Thus, it should result in 2, 3, ..., 7, 1.
	var canonicals = []CanonicalBlock{
		NewCanonicalBlock(5, 0, nil),
		NewCanonicalBlock(3, 0, nil),
		NewCanonicalBlock(6, 0, nil),
		NewCanonicalBlock(7, 0, nil),
		NewCanonicalBlock(4, 0, nil),
		NewCanonicalBlock(1, 0, nil),
		NewCanonicalBlock(2, 0, nil),
	}

	sort.Sort(canonicalBlockNumberSort(canonicals))

	for i := 0; i < len(canonicals)-1; i++ {
		if blockNumber := canonicals[i].BlockNumber; blockNumber != uint64(i+2) {
			t.Fatalf("index %d contains block number %d", i, blockNumber)
		}
	}

	if blockNumber := canonicals[len(canonicals)-1].BlockNumber; blockNumber != ExtBlockTypePayloadBlock {
		t.Fatalf("last block's block number is %d", blockNumber)
	}
}
