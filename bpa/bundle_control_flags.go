package bpa

import "math/bits"

// BundleControlFlags is an integer of 16 bits which represents the Bundle
// Processing Control Flags as specified in section 4.1.3. Those are part of
// the primary block of each bundle.
type BundleControlFlags uint16

const (
	BndlCFBundleDeletionStatusReportsAreRequested   BundleControlFlags = 0x1000
	BndlCFBundleDeliveryStatusReportsAreRequested   BundleControlFlags = 0x0800
	BndlCFBundleForwardingStatusReportsAreRequested BundleControlFlags = 0x0400
	BndlCFBundleReceptionStatusReportsAreRequested  BundleControlFlags = 0x0100
	BndlCFBundleContainsAManifestBlock              BundleControlFlags = 0x0080
	BndlCFStatusTimeIsRequestedInAllStatusReports   BundleControlFlags = 0x0040
	BndlCFUserApplicationAcknowledgementIsRequested BundleControlFlags = 0x0020
	BndlCFBundleMustNotBeFragmented                 BundleControlFlags = 0x0004
	BndlCFPayloadIsAnAdministrativeRecord           BundleControlFlags = 0x0002
	BndlCFBundleIsAFragment                         BundleControlFlags = 0x0001
	BndlCFReservedFields                            BundleControlFlags = 0xE218
)

// NewBundleControlFlags creates a new and empty BundleControlFlags.
func NewBundleControlFlags() BundleControlFlags {
	return 0
}

func bundleControlFlagsCheck(flag BundleControlFlags) error {
	if (flag & BndlCFReservedFields) != 0 {
		return NewBPAError("Given flag contains reserved bits")
	}

	if bits.OnesCount16(uint16(flag)) != 1 {
		return NewBPAError("Given flag does not contain one bit")
	}

	return nil
}

// Set sets one of the available flags to one. The flags are present as const
// starting with BndlCF in the bpa-package (see bpa/bundle_control_flags.go).
// An error is returned if more or less than one flag is altered or changes were
// made to a reserved field.
func (bcf *BundleControlFlags) Set(flag BundleControlFlags) error {
	if err := bundleControlFlagsCheck(flag); err != nil {
		return err
	}

	*bcf |= flag
	return nil
}

// Unset sets one of the available flags to zero. The flags are present as const
// starting with BndlCF in the bpa-package (see bpa/bundle_control_flags.go). An
// error is returned if more or less than one flag is altered or changes were
// made to a reserved field.
func (bcf *BundleControlFlags) Unset(flag BundleControlFlags) error {
	if err := bundleControlFlagsCheck(flag); err != nil {
		return err
	}

	*bcf &^= flag
	return nil
}

// Has returns true if a given flag or mask of flags is set.
func (bcf *BundleControlFlags) Has(flag BundleControlFlags) bool {
	return (*bcf & flag) != 0
}
