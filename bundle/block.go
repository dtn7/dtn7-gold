package bundle

import "github.com/dtn7/cboring"

// block is an interface for blocks present in a bundle. Both PrimaryBlock
// and the CanonicalBlock have the CRC-field in common.
type block interface {
	// block extends valid, "checkValid() error" method is required
	Valid

	// block extends cboring's CborMarshaler for MarshalCbor, UnmarshalCbor
	cboring.CborMarshaler

	// HasCRC returns if the CRCType indicates a CRC present for this block.
	// In this case the CRC value should become relevant.
	HasCRC() bool

	// GetCRCType returns the CRCType of this block.
	GetCRCType() CRCType

	// SetCRCType sets the CRC type.
	SetCRCType(CRCType)
}
