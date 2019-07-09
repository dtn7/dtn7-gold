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

// HasCRC returns if the CRCType indicates a CRC is present for this block.
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
		if err := cboring.WriteArrayLength(blockLen, crcBuff); err != nil {
			return err
		}
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

	if eb, ebErr := GetExtensionBlockManager().CreateBlock(blockType); ebErr != nil {
		return fmt.Errorf("Unsupported block type code: %d", blockType)
	} else {
		cb.Value = eb
		if err := cboring.Unmarshal(cb.Value, r); err != nil {
			return fmt.Errorf("Unmarshalling block type %d failed: %v", blockType, err)
		}
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

func (cb CanonicalBlock) CheckValid() (errs error) {
	if bcfErr := cb.BlockControlFlags.CheckValid(); bcfErr != nil {
		errs = multierror.Append(errs, bcfErr)
	}

	if extErr := cb.Value.CheckValid(); extErr != nil {
		errs = multierror.Append(errs, extErr)
	}

	if cb.Value.BlockTypeCode() == ExtBlockTypePayloadBlock && cb.BlockNumber != 1 {
		errs = multierror.Append(errs, fmt.Errorf(
			"CanonicalBlock is a PayloadBlock with a block number %d != 1", cb.BlockNumber))
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
