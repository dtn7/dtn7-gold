package bpa

// Block is an interface for the blocks present in a bundle. The PrimaryBlock
// and each kind of CanonicalBlock have the CRC-fields in common.
type Block interface {
	// HasCRC retruns if the CRCType indicates a CRC present for this block. In
	// this case the CRC value should become relevant.
	HasCRC() bool

	// CRCType returns the CRCType of this Block.
	GetCRCType() CRCType

	// Retruns the CRC value.
	GetCRC() []byte

	// SetCRCType sets the CRC type.
	SetCRCType(CRCType)

	// ResetCRC resets the CRC value to zero. This should be called before
	// calculating the CRC value of this Block.
	ResetCRC()

	// SetCRC sets the CRC value to the given value.
	SetCRC([]byte)
}
