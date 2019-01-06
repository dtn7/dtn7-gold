package bpa

// block is an interface for the blocks present in a bundle. The PrimaryBlock
// and each kind of CanonicalBlock have the CRC-fields in common.
type block interface {
	// block extends valid, "checkValid() error" method is required
	valid

	// HasCRC retruns true if the CRCType indicates a CRC present for this block.
	// In this case the CRC value should become relevant.
	HasCRC() bool

	// GetCRCType returns the CRCType of this Block.
	GetCRCType() CRCType

	// getCRC retruns the CRC value.
	getCRC() []byte

	// SetCRCType sets the CRC type.
	SetCRCType(CRCType)

	// CalculateCRC calculates and writes the CRC-value for this block.
	// This method changes the block's CRC value temporary and is not thread safe.
	CalculateCRC()

	// CheckCRC returns true if the CRC value matches to its CRCType or the
	// CRCType is CRCNo.
	// This method changes the block's CRC value temporary and is not thread safe.
	CheckCRC() bool

	// resetCRC resets the CRC value to zero. This should be called before
	// calculating the CRC value of this Block.
	resetCRC()

	// setCRC sets the CRC value to the given value.
	setCRC([]byte)
}
