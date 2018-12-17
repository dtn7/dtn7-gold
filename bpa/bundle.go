package bpa

import (
	"bytes"
	_ "fmt"

	"github.com/ugorji/go/codec"
)

// Bundle represents a Bundle as defined in section 4.2.1. Each Bundle contains
// one Primary Block and multiple Canonical Blocks.
type Bundle struct {
	_struct struct{} `codec:",toarray"`

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
func (b *Bundle) forEachBlock(f func(Block)) {
	f(&b.PrimaryBlock)
	for i := 0; i < len(b.CanonicalBlocks); i++ {
		f(&b.CanonicalBlocks[i])
	}
}

// ApplyCRC sets the given CRCType to each block and calculates the CRC values.
func (b *Bundle) ApplyCRC(crcType CRCType) {
	b.forEachBlock(func(block Block) {
		block.SetCRCType(crcType)
		SetCRC(block)
	})
}

// ToCbor creates a byte array representing a CBOR indefinite-length array of
// this Bundle with all its blocks.
func (b Bundle) ToCbor() []byte {
	// It seems to be tricky using both definite-length and indefinite-length
	// arays with the codec library. However, an indefinite-length array is just
	// a byte array wrapped inside the start and "break" codes, which are
	// exported as consts from the codec library.

	var buf bytes.Buffer
	var cborHandle *codec.CborHandle = new(codec.CborHandle)

	buf.WriteByte(codec.CborStreamArray)

	b.forEachBlock(func(block Block) {
		codec.NewEncoder(&buf, cborHandle).MustEncode(block)
	})

	buf.WriteByte(codec.CborStreamBreak)

	return buf.Bytes()
}

/*
// TODO
func NewBundleFromCbor(data []byte) Bundle {
	var bundle Bundle
	var dataArr interface{}

	var cborHandle = codec.CborHandle{IndefiniteLength: true}
	var dec *codec.Decoder = codec.NewDecoderBytes(data, &cborHandle)

	dec.MustDecode(&dataArr)

	fmt.Printf("%v\n", dataArr)

	return bundle
}
*/
