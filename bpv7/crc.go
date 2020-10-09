// SPDX-FileCopyrightText: 2018, 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"github.com/dtn7/cboring"
	"github.com/howeyc/crc16"
)

// CRCType indicates which CRC type is used. Only the three defined consts
// CRCNo, CRC16 and CRC32 are valid, as specified in section 4.1.1.
type CRCType uint64

const (
	// CRCNo means no CRC to be present at all.
	CRCNo CRCType = 0

	// CRC16 represents "a standard X-25 CRC-16".
	CRC16 CRCType = 1

	// CRC32 represents "a standard CRC32C (Castagnoli) CRC-32".
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

// calculateCRCBuff calculates a block's CRC value for serialization.
func calculateCRCBuff(buff *bytes.Buffer, crcType CRCType) ([]byte, error) {
	// Append CRC type's empty bytes
	data, typeErr := emptyCRC(crcType)
	if typeErr != nil {
		return nil, typeErr
	}

	if err := cboring.WriteByteString(data, buff); err != nil {
		return nil, err
	}

	// Write CRC value for buff's data into data
	switch crcType {
	case CRCNo:

	case CRC16:
		binary.BigEndian.PutUint16(data, crc16.Checksum(buff.Bytes(), crc16table))

	case CRC32:
		binary.BigEndian.PutUint32(data, crc32.Checksum(buff.Bytes(), crc32table))

	default:
		return nil, fmt.Errorf("unknown CRCType %d", crcType)
	}

	return data, nil
}

// emptyCRC returns the "default" CRC value for the given CRC Type.
func emptyCRC(crcType CRCType) (arr []byte, err error) {
	switch crcType {
	case CRCNo:
		arr = nil

	case CRC16:
		arr = make([]byte, 2)

	case CRC32:
		arr = make([]byte, 4)

	default:
		err = fmt.Errorf("unknown CRCType %d", crcType)
	}

	return
}
