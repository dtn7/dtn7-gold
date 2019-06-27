package bundle

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/dtn7/cboring"
	"github.com/hashicorp/go-multierror"
)

// CanonicalBlock represents the canonical bundle block defined in section 4.2.3.
type CanonicalBlock struct {
	BlockNumber       uint64
	BlockControlFlags BlockControlFlags
	CRCType           CRCType
	CRC               []byte
	Value             ExtensionBlock
}

func NewCanonicalBlock(no uint64, bcf BlockControlFlags, value ExtensionBlock) CanonicalBlock {
	return CanonicalBlock{
		BlockNumber:       no,
		BlockControlFlags: bcf,
		CRCType:           CRCNo,
		CRC:               nil,
		Value:             value,
	}
}

// BlockTypeCode returns the block type code.
func (cb CanonicalBlock) BlockTypeCode() uint64 {
	return cb.Value.BlockTypeCode()
}

// HasCRC retruns true if the CRCType indicates a CRC present for this block.
func (cb CanonicalBlock) HasCRC() bool {
	return cb.GetCRCType() != CRCNo
}

// GetCRCType returns the CRCType of this block.
func (cb CanonicalBlock) GetCRCType() CRCType {
	return cb.CRCType
}

// SetCRCType sets the CRC type.
func (cb *CanonicalBlock) SetCRCType(crcType CRCType) {
	cb.CRCType = crcType
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

	fields := []uint64{cb.BlockTypeCode(), cb.BlockNumber,
		uint64(cb.BlockControlFlags), uint64(cb.CRCType)}
	for _, f := range fields {
		if err := cboring.WriteUInt(f, w); err != nil {
			return err
		}
	}

	if err := cboring.Marshal(cb.Value, w); err != nil {
		return fmt.Errorf("Marshalling value failed: %v", err)
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

	var blockType uint64
	if bt, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		blockType = bt
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

	// TODO: generic factory or the like
	switch blockType {
	case ExtBlockTypePayloadBlock:
		var pb PayloadBlock
		if err := cboring.Unmarshal(&pb, r); err != nil {
			return fmt.Errorf("Unmarshalling PayloadBlock failed: %v", err)
		}
		cb.Value = &pb

	case ExtBlockTypePreviousNodeBlock:
		var pnb PreviousNodeBlock
		if err := cboring.Unmarshal(&pnb, r); err != nil {
			return fmt.Errorf("Unmarshalling PreviousNodeBlock failed: %v", err)
		}
		cb.Value = &pnb

	case ExtBlockTypeBundleAgeBlock:
		var bab BundleAgeBlock
		if err := cboring.Unmarshal(&bab, r); err != nil {
			return fmt.Errorf("Unmarshalling BundleAgeBlock failed: %v", err)
		}
		cb.Value = &bab

	case ExtBlockTypeHopCountBlock:
		var hcb HopCountBlock
		if err := cboring.Unmarshal(&hcb, r); err != nil {
			return fmt.Errorf("Unmarshalling HopCountBlock failed: %v", err)
		}
		cb.Value = &hcb

	default:
		return fmt.Errorf("Unsupported block type code: %d", blockType)
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
	// TODO
	/*
		switch cb.BlockType {
		case PayloadBlock:
			if cb.BlockNumber != 0 {
				return fmt.Errorf("CanonicalBlock: Payload Block's block number is not zero")
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
				return fmt.Errorf("CanonicalBlock: Unknown block type %d", cb.BlockType)
			}
		}
	*/

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

	fmt.Fprintf(&b, "block type code: %d, ", cb.Value.BlockTypeCode())
	fmt.Fprintf(&b, "block number: %d, ", cb.BlockNumber)
	fmt.Fprintf(&b, "block processing control flags: %b, ", cb.BlockControlFlags)
	fmt.Fprintf(&b, "crc type: %v, ", cb.CRCType)
	fmt.Fprintf(&b, "data: %v", cb.Value)

	if cb.HasCRC() {
		fmt.Fprintf(&b, ", crc: %x", cb.CRC)
	}

	return b.String()
}
