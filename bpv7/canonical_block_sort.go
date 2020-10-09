// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

// canonicalBlockNumberSort implements sort.Interface to sort []CanonicalBlock based on their block number.
// The sorting is in ascending order. An exception is the payload block, which occurs in the last position despite
// having the lowest block number of 1.
//
// This allows a deterministic sorting of CanonicalBlocks, e.g., necessary for the BundleBuilder.
type canonicalBlockNumberSort []CanonicalBlock

// Len of elements within the array.
func (cbns canonicalBlockNumberSort) Len() int {
	return len(cbns)
}

// Less is true iff element i should be sort before element j.
//
// Thus, if element i is a Payload Block, this function returns false because this block must always be the last one,
// et vice versa. Otherwise, the block numbers are compared in ascending order.
func (cbns canonicalBlockNumberSort) Less(i, j int) bool {
	if cbns[i].BlockNumber == ExtBlockTypePayloadBlock {
		return false
	} else if cbns[j].BlockNumber == ExtBlockTypePayloadBlock {
		return true
	} else {
		return cbns[i].BlockNumber < cbns[j].BlockNumber
	}
}

// Swap elements i and j.
func (cbns canonicalBlockNumberSort) Swap(i, j int) {
	cbns[i], cbns[j] = cbns[j], cbns[i]
}
