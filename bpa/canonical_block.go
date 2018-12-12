package bpa

import (
	"fmt"
	"strings"
)

// CanonicalBlock represents the Canonical Bundle Block defined in section 4.2.3
type CanonicalBlock struct {
	BlockType         uint
	BlockNumber       uint
	BlockControlFlags BlockControlFlags
	CRCType           CRCType
	Data              []byte
	CRC               uint
}

// NewCanonicalBlock creates a new CanonicalBlock with the given parameters.
func NewCanonicalBlock(blockType uint, blockNumber uint,
	blockControlFlags BlockControlFlags, data []byte) CanonicalBlock {
	return CanonicalBlock{
		BlockType:         blockType,
		BlockControlFlags: blockControlFlags,
		CRCType:           CRCNo,
		Data:              data,
		CRC:               0,
	}
}

// NewPayloadBlock creates a new payload block based on the CanonicalBlock.
func NewPayloadBlock(blockControlFlags BlockControlFlags, data []byte) CanonicalBlock {
	// Payload block's values are defined in 4.2.3:
	// - block type: 1
	// - block number: 0
	return NewCanonicalBlock(1, 0, blockControlFlags, data)
}

// HasCRC retruns if the CRCType indicates a CRC present for this block. In
// this case the CRC field of this struct should become relevant.
func (cb CanonicalBlock) HasCRC() bool {
	return cb.CRCType != CRCNo
}

func (cb CanonicalBlock) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "block type code: %d, ", cb.BlockType)
	fmt.Fprintf(&b, "block number: %d, ", cb.BlockNumber)
	fmt.Fprintf(&b, "block processing control flags: %b, ", cb.BlockControlFlags)
	fmt.Fprintf(&b, "crc type: %v, ", cb.CRCType)
	fmt.Fprintf(&b, "data: %v", cb.Data)

	if cb.HasCRC() {
		fmt.Fprintf(&b, ", crc: %x", cb.CRC)
	}

	return b.String()
}
