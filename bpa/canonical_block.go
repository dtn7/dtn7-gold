package bpa

import (
	"fmt"
	"strings"

	"github.com/ugorji/go/codec"
)

const (
	// BlockTypePayload is a BlockType for a payload block as defined in 4.2.3.
	BlockTypePayload uint = 1

	// BlockTypeIntegrity is a BlockType defined in the Bundle Security Protocol
	// specifiation.
	BlockTypeIntegrity uint = 2

	// BlockTypeConfidentiality is a BlockType defined in the Bundle Security
	// Protocol specifiation.
	BlockTypeConfidentiality uint = 3

	// BlockTypeManifest is a BlockType defined in the Manifest Extension Block
	// specifiation.
	BlockTypeManifest uint = 4

	// BlockTypeFlowLabel is a BlockType defined in the Flow Label Extension Block
	// specification.
	BlockTypeFlowLabel uint = 6

	// BlockTypePreviousNode is a BlockType for a Previous Node block as defined
	// in section 4.3.1.
	BlockTypePreviousNode uint = 7

	// BlockTypeBundleAge is a BlockType for a Bundle Age block as defined in
	// section 4.3.2.
	BlockTypeBundleAge uint = 8

	// BlockTypeHopCount is a BlockType for a Hop Count block as defined in
	// section 4.3.3.
	BlockTypeHopCount uint = 9
)

// CanonicalBlock represents the Canonical Bundle Block defined in section 4.2.3
type CanonicalBlock struct {
	BlockType         uint
	BlockNumber       uint
	BlockControlFlags BlockControlFlags
	CRCType           CRCType
	Data              interface{}
	CRC               []byte
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
		CRC:               nil,
	}
}

func (cb CanonicalBlock) HasCRC() bool {
	return cb.GetCRCType() != CRCNo
}

func (cb CanonicalBlock) GetCRCType() CRCType {
	return cb.CRCType
}

func (cb CanonicalBlock) GetCRC() []byte {
	return cb.CRC
}

func (cb *CanonicalBlock) SetCRCType(crcType CRCType) {
	cb.CRCType = crcType
}

func (cb *CanonicalBlock) ResetCRC() {
	cb.CRC = nil
}

func (cb *CanonicalBlock) SetCRC(crc []byte) {
	cb.CRC = crc
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

func (cb *CanonicalBlock) codecDecodeData(data interface{}) {
	switch cb.BlockType {
	case BlockTypePreviousNode:
		var ep *EndpointID = new(EndpointID)
		setEndpointIDFromCborArray(ep, data.([]interface{}))
		cb.Data = *ep

	case BlockTypeBundleAge:
		cb.Data = uint(data.(uint64))

	case BlockTypeHopCount:
		tuple := data.([]interface{})
		cb.Data = HopCount{
			Limit: uint(tuple[0].(uint64)),
			Count: uint(tuple[1].(uint64)),
		}

	// BlockTypePayload is also a byte array and can be treated like the default.
	default:
		cb.Data = data.([]byte)
	}
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

	cb.codecDecodeData(blockArr[4])

	if len(blockArr) == 6 {
		cb.CRC = blockArr[5].([]byte)
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

// HopCount represents the tuple of a hop limit and hop count defined in 4.3.3
// for the Hop Count block.
type HopCount struct {
	_struct struct{} `codec:",toarray"`

	Limit uint
	Count uint
}

func (hc HopCount) String() string {
	return fmt.Sprintf("(%d, %d)", hc.Limit, hc.Count)
}

// NewPayloadBlock creates a new payload block.
func NewPayloadBlock(blockControlFlags BlockControlFlags, data []byte) CanonicalBlock {
	// A payload block's block number is always 0 (4.2.3)
	return NewCanonicalBlock(BlockTypePayload, 0, blockControlFlags, data)
}

// NewPreviousNodeBlock creates a new Previous Node block.
func NewPreviousNodeBlock(blockNumber uint, blockControlFlags BlockControlFlags,
	prevNodeId EndpointID) CanonicalBlock {
	return NewCanonicalBlock(
		BlockTypePreviousNode, blockNumber, blockControlFlags, prevNodeId)
}

// NewBundleAgeBlock creates a new Bundle Age block to hold the bundle's lifetime
// in microseconds.
func NewBundleAgeBlock(blockNumber uint, blockControlFlags BlockControlFlags,
	time uint) CanonicalBlock {
	return NewCanonicalBlock(
		BlockTypeBundleAge, blockNumber, blockControlFlags, time)
}

// NewHopCountBlock creates a new Hop Count block.
func NewHopCountBlock(blockNumber uint, blockControlFlags BlockControlFlags,
	hopCount HopCount) CanonicalBlock {
	return NewCanonicalBlock(
		BlockTypeHopCount, blockNumber, blockControlFlags, hopCount)
}
