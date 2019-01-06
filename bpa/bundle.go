package bpa

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/ugorji/go/codec"
)

// Bundle represents a Bundle as defined in section 4.2.1. Each Bundle contains
// one Primary Block and multiple Canonical Blocks.
type Bundle struct {
	PrimaryBlock    PrimaryBlock
	CanonicalBlocks []CanonicalBlock
}

// NewBundle creates a new Bundle.
func NewBundle(primary PrimaryBlock, canonicals []CanonicalBlock) (b Bundle, err error) {
	b = Bundle{
		PrimaryBlock:    primary,
		CanonicalBlocks: canonicals,
	}
	err = b.checkValid()

	return
}

// forEachBlock applies the given function for each of this Bundle's blocks.
func (b *Bundle) forEachBlock(f func(block)) {
	f(&b.PrimaryBlock)
	for i := 0; i < len(b.CanonicalBlocks); i++ {
		f(&b.CanonicalBlocks[i])
	}
}

// SetCRCType sets the given CRCType for each block.
func (b *Bundle) SetCRCType(crcType CRCType) {
	b.forEachBlock(func(blck block) {
		blck.SetCRCType(crcType)
	})
}

// CalculateCRC calculates and sets the CRC value for each block.
func (b *Bundle) CalculateCRC() {
	b.forEachBlock(func(blck block) {
		blck.CalculateCRC()
	})
}

// CheckCRC checks the CRC value of each block and returns false if some
// values do not match.
// This method changes the block's CRC value temporary and is not thread safe.
func (b *Bundle) CheckCRC() bool {
	var flag = true

	b.forEachBlock(func(blck block) {
		if !blck.CheckCRC() {
			flag = false
		}
	})

	return flag
}

func (b Bundle) checkValid() (errs error) {
	// Check blocks for errors
	b.forEachBlock(func(blck block) {
		if blckErr := blck.checkValid(); blckErr != nil {
			errs = multierror.Append(errs, blckErr)
		}
	})

	// Check CanonicalBlocks for errors
	if b.PrimaryBlock.BundleControlFlags.Has(BndlCFPayloadIsAnAdministrativeRecord) ||
		b.PrimaryBlock.SourceNode == DtnNone() {
		for _, cb := range b.CanonicalBlocks {
			if cb.BlockControlFlags.Has(BlckCFStatusReportMustBeTransmittedIfBlockCannotBeProcessed) {
				errs = multierror.Append(errs,
					newBPAError("Bundle: Bundle Processing Control Flags indicate that "+
						"this bundle's payload is an administrative record or the source "+
						"node is omitted, but the \"Transmit status report if block canot "+
						"be processed\" Block Processing Control Flag was set in a "+
						"Canonical Block"))
			}
		}
	}

	// Check uniqueness of block numbers
	var cbBlockNumbers = make(map[uint]bool)
	// Check max 1 occurrence of extension blocks
	var cbBlockTypes = make(map[uint]bool)

	for _, cb := range b.CanonicalBlocks {
		if _, ok := cbBlockNumbers[cb.BlockNumber]; ok {
			errs = multierror.Append(errs,
				newBPAError(fmt.Sprintf(
					"Bundle: Block number %d occured multiple times", cb.BlockNumber)))
		}
		cbBlockNumbers[cb.BlockNumber] = true

		switch cb.BlockType {
		case blockTypePreviousNode, blockTypeBundleAge, blockTypeHopCount:
			if _, ok := cbBlockTypes[cb.BlockType]; ok {
				errs = multierror.Append(errs,
					newBPAError(fmt.Sprintf(
						"Bundle: Block type %d occured multiple times", cb.BlockType)))
			}
			cbBlockTypes[cb.BlockType] = true
		}
	}

	if b.PrimaryBlock.CreationTimestamp[0] == 0 {
		if _, ok := cbBlockTypes[blockTypeBundleAge]; !ok {
			errs = multierror.Append(errs, newBPAError(
				"Bundle: Creation Timestamp is zero, but no Bundle Age block is present"))
		}
	}

	return
}

// ToCbor creates a byte array representing a CBOR indefinite-length array of
// this Bundle with all its blocks.
func (b Bundle) ToCbor() []byte {
	// It seems to be tricky using both definite-length and indefinite-length
	// arays with the codec library. However, an indefinite-length array is just
	// a byte array wrapped between the start and "break" code, which are
	// exported as consts from the codec library.

	var buf bytes.Buffer
	var cborHandle *codec.CborHandle = new(codec.CborHandle)

	buf.WriteByte(codec.CborStreamArray)

	b.forEachBlock(func(blck block) {
		codec.NewEncoder(&buf, cborHandle).MustEncode(blck)
	})

	buf.WriteByte(codec.CborStreamBreak)

	return buf.Bytes()
}

// decodeBundleBlock decodes an already generic decoded block to its
// determinated data structure.
// The NewBundleFromCbor function decodes an array of interface{} which results
// in an array of arrays, as codec tries to decode the whole data. This method
// will re-encode this "anonymous" array to CBOR and will decode it to its
// struct, which is referenced as the target pointer.
func decodeBundleBlock(data interface{}, target interface{}) {
	var b []byte = make([]byte, 0, 64)
	var cborHandle *codec.CborHandle = new(codec.CborHandle)

	codec.NewEncoderBytes(&b, cborHandle).MustEncode(data)
	codec.NewDecoderBytes(b, cborHandle).MustDecode(target)
}

// NewBundleFromCbor decodes the given data to a new Bundle.
func NewBundleFromCbor(data []byte) Bundle {
	var dataArr []interface{}
	codec.NewDecoderBytes(data, new(codec.CborHandle)).MustDecode(&dataArr)

	var pb PrimaryBlock
	decodeBundleBlock(dataArr[0], &pb)

	var cb []CanonicalBlock = make([]CanonicalBlock, len(dataArr)-1)
	for i := 0; i < len(cb); i++ {
		decodeBundleBlock(dataArr[i+1], &cb[i])
	}

	return Bundle{pb, cb}
}
