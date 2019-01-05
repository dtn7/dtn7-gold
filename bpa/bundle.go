package bpa

import (
	"bytes"

	"github.com/ugorji/go/codec"
)

// Bundle represents a Bundle as defined in section 4.2.1. Each Bundle contains
// one Primary Block and multiple Canonical Blocks.
type Bundle struct {
	PrimaryBlock    PrimaryBlock
	CanonicalBlocks []CanonicalBlock
}

// NewBundle creates a new Bundle.
func NewBundle(primary PrimaryBlock, canonicals []CanonicalBlock) Bundle {
	return Bundle{
		PrimaryBlock:    primary,
		CanonicalBlocks: canonicals,
	}
}

// forEachBlock applies the given function for each of this Bundle's blocks.
func (b *Bundle) forEachBlock(f func(block)) {
	f(&b.PrimaryBlock)
	for i := 0; i < len(b.CanonicalBlocks); i++ {
		f(&b.CanonicalBlocks[i])
	}
}

// ApplyCRC sets the given CRCType to each block and calculates the CRC values.
func (b *Bundle) ApplyCRC(crcType CRCType) {
	b.forEachBlock(func(blck block) {
		blck.setCRCType(crcType)
		setCRC(blck)
	})
}

// CheckCRC checks the CRC value of each block.
func (b *Bundle) CheckCRC() bool {
	var flag = true

	b.forEachBlock(func(blck block) {
		if !checkCRC(blck) {
			flag = false
		}
	})

	return flag
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
