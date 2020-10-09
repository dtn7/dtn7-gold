// SPDX-FileCopyrightText: 2018, 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"encoding/json"
	"strings"
)

// BlockControlFlags is an uint which represents the Block Processing Control
// Flags as specified in 4.1.4.
type BlockControlFlags uint64

const (
	// ReplicateBlock requires this block to be replicated in every fragment.
	ReplicateBlock BlockControlFlags = 0x01

	// StatusReportBlock requires transmission of a status report if this block cannot be processed.
	StatusReportBlock BlockControlFlags = 0x02

	// DeleteBundle requires bundle deletion if this block cannot be processed.
	DeleteBundle BlockControlFlags = 0x04

	// RemoveBlock requires the block to be removed from the bundle if it cannot be processed.
	RemoveBlock BlockControlFlags = 0x10
)

// Has returns true if a given flag or mask of flags is set.
func (bcf BlockControlFlags) Has(flag BlockControlFlags) bool {
	return (bcf & flag) != 0
}

// CheckValid returns an array of errors for incorrect data.
func (bcf BlockControlFlags) CheckValid() error {
	// There is currently nothing to check here.
	// Especially since dtn-bpbpis-24 no longer defines unknown bits as faults.
	return nil
}

// Strings returns an array of all flags as a string representation.
func (bcf BlockControlFlags) Strings() (fields []string) {
	checks := []struct {
		field BlockControlFlags
		text  string
	}{
		{DeleteBundle, "DELETE_BUNDLE"},
		{StatusReportBlock, "REQUEST_STATUS_REPORT"},
		{RemoveBlock, "REMOVE_BLOCK"},
		{ReplicateBlock, "REPLICATE_BLOCK"},
	}

	for _, check := range checks {
		if bcf.Has(check.field) {
			fields = append(fields, check.text)
		}
	}

	return
}

// MarshalJSON returns a JSON array of control flags.
func (bcf BlockControlFlags) MarshalJSON() ([]byte, error) {
	return json.Marshal(bcf.Strings())
}

func (bcf BlockControlFlags) String() string {
	return strings.Join(bcf.Strings(), ",")
}
