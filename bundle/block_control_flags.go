package bundle

import (
	"fmt"
	"strings"
)

// BlockControlFlags is an uint which represents the Block Processing Control
// Flags as specified in 4.1.4.
type BlockControlFlags uint64

const (
	// DeleteBundle: Bundle must be deleted if this block can't be processed.
	DeleteBundle BlockControlFlags = 0x08

	// StatusReportBlock: Transmission of a status report is requested if this
	// block can't be processed.
	StatusReportBlock BlockControlFlags = 0x04

	// RemoveBlock: Block must be removed from the bundle if it can't be processed.
	RemoveBlock BlockControlFlags = 0x02

	// ReplicateBlock: This block must be replicated in every fragment.
	ReplicateBlock BlockControlFlags = 0x01

	blckCFReservedFields BlockControlFlags = 0xF0
)

// Has returns true if a given flag or mask of flags is set.
func (bcf BlockControlFlags) Has(flag BlockControlFlags) bool {
	return (bcf & flag) != 0
}

func (bcf BlockControlFlags) CheckValid() error {
	if bcf.Has(blckCFReservedFields) {
		return fmt.Errorf("BlockControlFlags: Given flag %x contains reserved bits", bcf)
	}

	return nil
}

func (bcf BlockControlFlags) String() string {
	var fields []string

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

	return strings.Join(fields, ",")
}
