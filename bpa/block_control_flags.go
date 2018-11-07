package bpa

import "math/bits"

// BlockControlFlags is an integer of 8 bits which represents the Block
// Processing Control Flags as specified in 4.1.4. Those are part of the
// canonical bundle blocks.
type BlockControlFlags uint8

// TODO: There should be a check against the block's Bundle Processing Control
// Flags if the data is an administrative record or the endpoint is the null
// endpoint.

const (
	BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed           BlockControlFlags = 0x08
	BlckCFStatusReportMustBeTransmittedIfBlockCannotBeProcessed BlockControlFlags = 0x04
	BlckCFBlockMustBeRemovedFromBundleIfItCannotBeProcessed     BlockControlFlags = 0x02
	BlckCFBlockMustBeReplicatedInEveryFragment                  BlockControlFlags = 0x01
	BlckCFReservedFields                                        BlockControlFlags = 0xF0
)

// NewBlockControlFlags returns a new and empty BlockControlFlags.
func NewBlockControlFlags() BlockControlFlags {
	return 0
}

func blockControlFlagsCheck(flag BlockControlFlags) error {
	if (flag & BlckCFReservedFields) != 0 {
		return NewBPAError("Given flag contains reserved bits")
	}

	if bits.OnesCount8(uint8(flag)) != 1 {
		return NewBPAError("Given flag does not contain one bit")
	}

	return nil
}

// Set sets one of the available flags to one. The flags are present as const
// starting with BlckCF in the bpa-package (see bpa/block_control_flags.go).
// An error is returned if more or less than one flag is altered or changes were
// made to a reserved field.
func (bcf *BlockControlFlags) Set(flag BlockControlFlags) error {
	if err := blockControlFlagsCheck(flag); err != nil {
		return err
	}

	*bcf |= flag
	return nil
}

// Unset sets one of the available flags to zero. The flags are present as const
// starting with BlckCF in the bpa-package (see bpa/block_control_flags.go). An
// error is returned if more or less than one flag is altered or changes were
// made to a reserved field.
func (bcf *BlockControlFlags) Unset(flag BlockControlFlags) error {
	if err := blockControlFlagsCheck(flag); err != nil {
		return err
	}

	*bcf &^= flag
	return nil
}

// Has returns true if a given flag or mask of flags is set.
func (bcf *BlockControlFlags) Has(flag BlockControlFlags) bool {
	return (*bcf & flag) != 0
}
