package bpa

import (
	"hash/crc32"

	"github.com/howeyc/crc16"
	"github.com/ugorji/go/codec"
)

// CRCType indicates which CRC type is used. Only the three defined consts
// CRCNo, CRC16 and CRC32 are valid, as specified in section 4.1.1.
type CRCType uint

const (
	CRCNo CRCType = 0
	CRC16 CRCType = 1
	CRC32 CRCType = 2
)

func (c CRCType) String() string {
	switch c {
	case CRCNo:
		return "no"
	case CRC16:
		return "16"
	case CRC32:
		return "32"
	default:
		return "unknown"
	}
}

var (
	crc16table = crc16.MakeTable(crc16.CCITT)
	crc32table = crc32.MakeTable(crc32.Castagnoli)
)

// blockToBytes encodes a Block to a byte array based on the CBOR encoding. It
// temporary sets the present CRC value to zero. Therefore this function is not
// thread safe.
func blockToBytes(block Block) []byte {
	var b []byte = make([]byte, 0, 64)
	var enc *codec.Encoder = codec.NewEncoderBytes(&b, new(codec.CborHandle))

	var blockCRC = block.GetCRC()

	block.ResetCRC()
	enc.MustEncode(block)
	block.SetCRC(blockCRC)

	return b
}

// CalculateCRC calculates a Block's CRC value based on its CRCType. The CRC
// value will be set to zero temporary during calcuation. Thereforce this
// function is not thread safe.
func CalculateCRC(block Block) uint {
	var data = blockToBytes(block)

	switch block.GetCRCType() {
	case CRCNo:
		return 0

	case CRC16:
		return uint(crc16.Checksum(data, crc16table))

	case CRC32:
		return uint(crc32.Checksum(data, crc32table))

	default:
		panic("Unknown CRCType")
	}
}

// SetCRC sets the CRC value of the given block.
func SetCRC(block Block) {
	block.SetCRC(CalculateCRC(block))
}

// CheckCRC returns true if the stored CRC value matches the calculated one.
func CheckCRC(block Block) bool {
	return block.GetCRC() == CalculateCRC(block)
}
