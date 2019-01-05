package bpa

// block is an interface for the blocks present in a bundle. The PrimaryBlock
// and each kind of CanonicalBlock have the CRC-fields in common.
type block interface {
	// HasCRC retruns if the CRCType indicates a CRC present for this block. In
	// this case the CRC value should become relevant.
	HasCRC() bool

	// GetCRCType returns the CRCType of this Block.
	GetCRCType() CRCType

	// getCRC retruns the CRC value.
	getCRC() []byte

	// setCRCType sets the CRC type.
	setCRCType(CRCType)

	// resetCRC resets the CRC value to zero. This should be called before
	// calculating the CRC value of this Block.
	resetCRC()

	// setCRC sets the CRC value to the given value.
	setCRC([]byte)
}
