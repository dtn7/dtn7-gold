package bundle

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/ugorji/go/codec"
)

// CanonicalBlockType is an uint which is used as "block type code" for the
// canonical block. The BlockType-consts may be used.
type CanonicalBlockType uint

const (
	// PayloadBlock is a BlockType for a payload block as defined in 4.2.3.
	PayloadBlock CanonicalBlockType = 1

	// IntegrityBlock is a BlockType defined in the Bundle Security Protocol
	// specifiation.
	IntegrityBlock CanonicalBlockType = 2

	// ConfidentialityBlock is a BlockType defined in the Bundle Security
	// Protocol specifiation.
	ConfidentialityBlock CanonicalBlockType = 3

	// ManifestBlock is a BlockType defined in the Manifest Extension Block
	// specifiation.
	ManifestBlock CanonicalBlockType = 4

	// FlowLabelBlock is a BlockType defined in the Flow Label Extension Block
	// specification.
	FlowLabelBlock CanonicalBlockType = 6

	// PreviousNodeBlock is a BlockType for a Previous Node block as defined
	// in section 4.3.1.
	PreviousNodeBlock CanonicalBlockType = 7

	// BundleAgeBlock is a BlockType for a Bundle Age block as defined in
	// section 4.3.2.
	BundleAgeBlock CanonicalBlockType = 8

	// HopCountBlock is a BlockType for a Hop Count block as defined in
	// section 4.3.3.
	HopCountBlock CanonicalBlockType = 9
)

// CanonicalBlock represents the canonical bundle block defined
// in section 4.2.3.
type CanonicalBlock struct {
	BlockType         CanonicalBlockType
	BlockNumber       uint
	BlockControlFlags BlockControlFlags
	CRCType           CRCType
	Data              interface{}
	CRC               []byte
}

// NewCanonicalBlock creates a new canonical block with the given parameters.
func NewCanonicalBlock(blockType CanonicalBlockType, blockNumber uint,
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

// HasCRC retruns true if the CRCType indicates a CRC present for this block.
// In this case the CRC value should become relevant.
func (cb CanonicalBlock) HasCRC() bool {
	return cb.GetCRCType() != CRCNo
}

// GetCRCType returns the CRCType of this block.
func (cb CanonicalBlock) GetCRCType() CRCType {
	return cb.CRCType
}

// getCRC retruns the CRC value.
func (cb CanonicalBlock) getCRC() []byte {
	return cb.CRC
}

// SetCRCType sets the CRC type.
func (cb *CanonicalBlock) SetCRCType(crcType CRCType) {
	cb.CRCType = crcType
}

// CalculateCRC calculates and writes the CRC-value for this block.
func (cb *CanonicalBlock) CalculateCRC() {
	cb.setCRC(calculateCRC(cb))
}

// CheckCRC returns true if the CRC value matches to its CRCType or the
// CRCType is CRCNo.
//
// This method changes the block's CRC value temporary and is not thread safe.
func (cb *CanonicalBlock) CheckCRC() bool {
	return checkCRC(cb)
}

// resetCRC resets the CRC value to zero. This should be called before
// calculating the CRC value of this Block.
func (cb *CanonicalBlock) resetCRC() {
	cb.CRC = emptyCRC(cb.GetCRCType())
}

// setCRC sets the CRC value to the given value.
func (cb *CanonicalBlock) setCRC(crc []byte) {
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
	case PreviousNodeBlock:
		var ep *EndpointID = new(EndpointID)
		setEndpointIDFromCborArray(ep, data.([]interface{}))
		cb.Data = *ep

	case BundleAgeBlock:
		cb.Data = uint(data.(uint64))

	case HopCountBlock:
		tuple := data.([]interface{})
		cb.Data = HopCount{
			Limit: uint(tuple[0].(uint64)),
			Count: uint(tuple[1].(uint64)),
		}

	// blockTypePayload is also a byte array and can be treated like the default.
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

	cb.BlockType = CanonicalBlockType(blockArr[0].(uint64))
	cb.BlockNumber = uint(blockArr[1].(uint64))
	cb.BlockControlFlags = BlockControlFlags(blockArr[2].(uint64))
	cb.CRCType = CRCType(blockArr[3].(uint64))

	cb.codecDecodeData(blockArr[4])

	if len(blockArr) == 6 {
		cb.CRC = blockArr[5].([]byte)
	}
}

func (cb CanonicalBlock) checkValidExtensionBlocks() error {
	switch cb.BlockType {
	case PayloadBlock:
		if cb.BlockNumber != 0 {
			return newBundleError(
				"CanonicalBlock: Payload Block's block number is not zero")
		}

		return nil

	case IntegrityBlock, ConfidentialityBlock, ManifestBlock, FlowLabelBlock:
		// These extension blocks are defined in other specifications
		return nil

	case PreviousNodeBlock:
		return cb.Data.(EndpointID).checkValid()

	case BundleAgeBlock, HopCountBlock:
		// Nothing to check here
		return nil

	default:
		// "Block type codes 192 through 255 are not reserved and are available for
		// private and/or experimental use.", draft-ietf-dtn-bpbis-12#section-4.2.3
		if !(192 <= cb.BlockType && cb.BlockType <= 255) {
			return newBundleError("CanonicalBlock: Unknown block type")
		}
	}

	return nil
}

func (cb CanonicalBlock) checkValid() (errs error) {
	if bcfErr := cb.BlockControlFlags.checkValid(); bcfErr != nil {
		errs = multierror.Append(errs, bcfErr)
	}

	if extErr := cb.checkValidExtensionBlocks(); extErr != nil {
		errs = multierror.Append(errs, extErr)
	}

	return
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

// IsExceeded returns true if the hop limit exceeded.
func (hc HopCount) IsExceeded() bool {
	return hc.Count >= hc.Limit
}

// NewHopCount returns a new Hop Count block as defined in section 4.3.3.
func NewHopCount(limit, count uint) HopCount {
	return HopCount{
		Limit: limit,
		Count: count,
	}
}

func (hc HopCount) String() string {
	return fmt.Sprintf("(%d, %d)", hc.Limit, hc.Count)
}

// NewPayloadBlock creates a new payload block.
func NewPayloadBlock(blockControlFlags BlockControlFlags, data []byte) CanonicalBlock {
	// A payload block's block number is always 0 (4.2.3)
	return NewCanonicalBlock(PayloadBlock, 0, blockControlFlags, data)
}

// NewPreviousNodeBlock creates a new Previous Node block.
func NewPreviousNodeBlock(blockNumber uint, blockControlFlags BlockControlFlags,
	prevNodeId EndpointID) CanonicalBlock {
	return NewCanonicalBlock(
		PreviousNodeBlock, blockNumber, blockControlFlags, prevNodeId)
}

// NewBundleAgeBlock creates a new Bundle Age block to hold the bundle's lifetime
// in microseconds.
func NewBundleAgeBlock(blockNumber uint, blockControlFlags BlockControlFlags,
	time uint) CanonicalBlock {
	return NewCanonicalBlock(
		BundleAgeBlock, blockNumber, blockControlFlags, time)
}

// NewHopCountBlock creates a new Hop Count block.
func NewHopCountBlock(blockNumber uint, blockControlFlags BlockControlFlags,
	hopCount HopCount) CanonicalBlock {
	return NewCanonicalBlock(
		HopCountBlock, blockNumber, blockControlFlags, hopCount)
}
