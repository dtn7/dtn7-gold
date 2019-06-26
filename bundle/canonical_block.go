package bundle

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/dtn7/cboring"
	"github.com/hashicorp/go-multierror"
)

// CanonicalBlockType is an uint which is used as "block type code" for the
// canonical block. The BlockType-consts may be used.
type CanonicalBlockType uint64

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
	BlockNumber       uint64
	BlockControlFlags BlockControlFlags
	CRCType           CRCType
	Data              interface{}
	CRC               []byte
}

// NewCanonicalBlock creates a new canonical block with the given parameters.
func NewCanonicalBlock(blockType CanonicalBlockType, blockNumber uint64,
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
	return true
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

func (cb *CanonicalBlock) MarshalCbor(w io.Writer) error {
	var blockLen uint64 = 5
	if cb.HasCRC() {
		blockLen = 6
	}

	crcBuff := new(bytes.Buffer)
	if cb.HasCRC() {
		w = io.MultiWriter(w, crcBuff)
	}

	if err := cboring.WriteArrayLength(blockLen, w); err != nil {
		return err
	}

	fields := []uint64{uint64(cb.BlockType), cb.BlockNumber,
		uint64(cb.BlockControlFlags), uint64(cb.CRCType)}
	for _, f := range fields {
		if err := cboring.WriteUInt(f, w); err != nil {
			return err
		}
	}

	switch cb.BlockType {
	case PayloadBlock:
		// byte array
		if err := cboring.WriteByteString(cb.Data.([]byte), w); err != nil {
			return err
		}

	case BundleAgeBlock:
		// uint
		if err := cboring.WriteUInt(cb.Data.(uint64), w); err != nil {
			return err
		}

	case PreviousNodeBlock:
		// endpoint
		ep := cb.Data.(EndpointID)
		if err := cboring.Marshal(&ep, w); err != nil {
			return fmt.Errorf("EndpointID failed: %v", err)
		}

	case HopCountBlock:
		// hop count
		hc := cb.Data.(HopCount)
		if err := cboring.Marshal(&hc, w); err != nil {
			return fmt.Errorf("HopCount failed: %v", err)
		}

	default:
		return fmt.Errorf("Unsupported block type code: %d", cb.BlockType)
	}

	if cb.HasCRC() {
		if crcVal, crcErr := calculateCRCBuff(crcBuff, cb.CRCType); crcErr != nil {
			return crcErr
		} else if err := cboring.WriteByteString(crcVal, w); err != nil {
			return err
		} else {
			cb.CRC = crcVal
		}
	}

	return nil
}

func (cb *CanonicalBlock) UnmarshalCbor(r io.Reader) error {
	var blockLen uint64
	if bl, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if bl != 5 && bl != 6 {
		return fmt.Errorf("Expected array with length 5 or 6, got %d", bl)
	} else {
		blockLen = bl
	}

	// Pipe incoming bytes into a separate CRC buffer
	crcBuff := new(bytes.Buffer)
	if blockLen == 6 {
		// Replay array's start
		cboring.WriteArrayLength(blockLen, crcBuff)
		r = io.TeeReader(r, crcBuff)
	}

	if bt, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		cb.BlockType = CanonicalBlockType(bt)
	}

	if bn, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		cb.BlockNumber = bn
	}

	if bcf, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		cb.BlockControlFlags = BlockControlFlags(bcf)
	}

	if crcT, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		cb.CRCType = CRCType(crcT)
	}

	switch cb.BlockType {
	case PayloadBlock:
		// byte array
		if pl, err := cboring.ReadByteString(r); err != nil {
			return err
		} else {
			cb.Data = pl
		}

	case BundleAgeBlock:
		// uint
		if ba, err := cboring.ReadUInt(r); err != nil {
			return err
		} else {
			cb.Data = ba
		}

	case PreviousNodeBlock:
		// endpoint
		ep := EndpointID{}
		if err := cboring.Unmarshal(&ep, r); err != nil {
			return fmt.Errorf("EndpointID failed: %v", err)
		} else {
			cb.Data = ep
		}

	case HopCountBlock:
		// hop count
		hc := HopCount{}
		if err := cboring.Unmarshal(&hc, r); err != nil {
			return fmt.Errorf("HopCount failed: %v", err)
		} else {
			cb.Data = hc
		}

	default:
		return fmt.Errorf("Unsupported block type code: %d", cb.BlockType)
	}

	if blockLen == 6 {
		if crcCalc, crcErr := calculateCRCBuff(crcBuff, cb.CRCType); crcErr != nil {
			return crcErr
		} else if crcVal, err := cboring.ReadByteString(r); err != nil {
			return err
		} else if !bytes.Equal(crcCalc, crcVal) {
			return fmt.Errorf("Invalid CRC value: %x instead of expected %x", crcVal, crcCalc)
		} else {
			cb.CRC = crcVal
		}
	}

	return nil
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
		// private and/or experimental use.", draft-ietf-dtn-bpbis-13#section-4.2.3
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
	Limit uint64
	Count uint64
}

// IsExceeded returns true if the hop limit exceeded.
func (hc HopCount) IsExceeded() bool {
	return hc.Count > hc.Limit
}

// Increment increments the hop counter and returns false, if the hop limit is
// exceeded after incrementing the counter.
func (hc *HopCount) Increment() bool {
	hc.Count++

	return hc.IsExceeded()
}

// Decrement decrements the hop counter. This could be usefull if you want to
// reset the HopCount's state after sending a modified bundle.
func (hc *HopCount) Decrement() {
	hc.Count--
}

// NewHopCount returns a new Hop Count block as defined in section 4.3.3. The
// hop count will be set to zero, as specified for new blocks.
func NewHopCount(limit uint64) HopCount {
	return HopCount{
		Limit: limit,
		Count: 0,
	}
}

func (hc HopCount) String() string {
	return fmt.Sprintf("(%d, %d)", hc.Limit, hc.Count)
}

func (hc *HopCount) MarshalCbor(w io.Writer) error {
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	fields := []uint64{hc.Limit, hc.Count}
	for _, f := range fields {
		if err := cboring.WriteUInt(f, w); err != nil {
			return err
		}
	}

	return nil
}

func (hc *HopCount) UnmarshalCbor(r io.Reader) error {
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 2 {
		return fmt.Errorf("Expected array with length 2, got %d", l)
	}

	fields := []*uint64{&hc.Limit, &hc.Count}
	for _, f := range fields {
		if x, err := cboring.ReadUInt(r); err != nil {
			return err
		} else {
			*f = x
		}
	}

	return nil
}

// NewPayloadBlock creates a new payload block.
func NewPayloadBlock(blockControlFlags BlockControlFlags, data []byte) CanonicalBlock {
	// A payload block's block number is always 0 (4.2.3)
	return NewCanonicalBlock(PayloadBlock, 0, blockControlFlags, data)
}

// NewPreviousNodeBlock creates a new Previous Node block.
func NewPreviousNodeBlock(blockNumber uint64, blockControlFlags BlockControlFlags,
	prevNodeId EndpointID) CanonicalBlock {
	return NewCanonicalBlock(
		PreviousNodeBlock, blockNumber, blockControlFlags, prevNodeId)
}

// NewBundleAgeBlock creates a new Bundle Age block to hold the bundle's lifetime
// in microseconds.
func NewBundleAgeBlock(blockNumber uint64, blockControlFlags BlockControlFlags,
	time uint64) CanonicalBlock {
	return NewCanonicalBlock(
		BundleAgeBlock, blockNumber, blockControlFlags, time)
}

// NewHopCountBlock creates a new Hop Count block.
func NewHopCountBlock(blockNumber uint64, blockControlFlags BlockControlFlags,
	hopCount HopCount) CanonicalBlock {
	return NewCanonicalBlock(
		HopCountBlock, blockNumber, blockControlFlags, hopCount)
}
