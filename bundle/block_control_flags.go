package bundle

// BlockControlFlags is an uint8 which represents the Block Processing Control
// Flags as specified in 4.1.4.
type BlockControlFlags uint8

const (
	// DeleteBundle: Bundle must be deleted if this block can't be processed.
	DeleteBundle BlockControlFlags = 0x08

	// StatusReportBlock: Transmission of a status report is requested if this
	// block can't be processed.
	StatusReportBlock BlockControlFlags = 0x04

	// Block must be removed from the bundle if it can't be processed.
	RemoveBlock BlockControlFlags = 0x02

	// ReplicateBlock: This block must be replicated in every fragment.
	ReplicateBlock BlockControlFlags = 0x01

	blckCFReservedFields BlockControlFlags = 0xF0
)

func blockControlFlagsCheck(flag BlockControlFlags) error {

	return nil
}

// Has returns true if a given flag or mask of flags is set.
func (bcf BlockControlFlags) Has(flag BlockControlFlags) bool {
	return (bcf & flag) != 0
}

func (bcf BlockControlFlags) checkValid() error {
	if bcf.Has(blckCFReservedFields) {
		return newBundleError("BlockControlFlags: Given flag contains reserved bits")
	}

	return nil
}
