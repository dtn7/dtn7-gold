package bpa

import (
	"fmt"
	"strings"

	"github.com/ugorji/go/codec"
)

// CanonicalBlock represents the Canonical Bundle Block defined in section 4.2.3
type CanonicalBlock struct {
	BlockType         uint
	BlockNumber       uint
	BlockControlFlags BlockControlFlags
	CRCType           CRCType
	Data              interface{}
	CRC               uint
}

// NewCanonicalBlock creates a new CanonicalBlock with the given parameters.
func NewCanonicalBlock(blockType uint, blockNumber uint,
	blockControlFlags BlockControlFlags, data interface{}) CanonicalBlock {
	return CanonicalBlock{
		BlockType:         blockType,
		BlockNumber:       blockNumber,
		BlockControlFlags: blockControlFlags,
		CRCType:           CRCNo,
		Data:              data,
		CRC:               0,
	}
}

// NewPayloadBlock creates a new payload block based on the CanonicalBlock.
func NewPayloadBlock(blockControlFlags BlockControlFlags, data interface{}) CanonicalBlock {
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

func (cb CanonicalBlock) CodecEncodeSelf(enc *codec.Encoder) {
	var blockArr = []interface{}{
		cb.BlockType,
		cb.BlockNumber,
		cb.BlockControlFlags,
		cb.CRCType,
		cb.Data}

	if cb.HasCRC() {
		blockArr = append(blockArr, cb.CRC)
	}

	enc.MustEncode(blockArr)
}

func (cb *CanonicalBlock) CodecDecodeSelf(dec *codec.Decoder) {
	var blockArrPt = new([]interface{})
	dec.MustDecode(blockArrPt)

	var blockArr = *blockArrPt

	if len(blockArr) != 5 && len(blockArr) != 6 {
		panic("blockArr has wrong length (!= 5, 6)")
	}

	cb.BlockType = uint(blockArr[0].(uint64))
	cb.BlockNumber = uint(blockArr[1].(uint64))
	cb.BlockControlFlags = BlockControlFlags(blockArr[2].(uint64))
	cb.CRCType = CRCType(blockArr[3].(uint64))
	cb.Data = blockArr[4]

	if len(blockArr) == 6 {
		cb.CRC = uint(blockArr[5].(uint64))
	}
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
