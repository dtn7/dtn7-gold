package bpa

// CRCType indicates which CRC type is used. Only the three defined consts
// CRCNo, CRC16 and CRC32 are valid, as specified in section 4.1.1.
type CRCType uint

const (
	CRCNo CRCType = 0
	CRC16 CRCType = 1
	CRC32 CRCType = 2
)
