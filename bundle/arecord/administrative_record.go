package arecord

import (
	"bytes"
	"fmt"

	"github.com/dtn7/cboring"
	"github.com/dtn7/dtn7-go/bundle"
)

// AdministrativeRecord describes a possible administrative record, like a
// status report, implemented in the StatusReport struct.
type AdministrativeRecord interface {
	cboring.CborMarshaler

	// TypeCode returns this AdministrativeRecord's type code.
	TypeCode() uint64
}

// NewAdministrativeRecordFromCbor creates a new AdministrativeRecord from
// a given byte array.
func NewAdministrativeRecordFromCbor(data []byte) (ar AdministrativeRecord, err error) {
	buff := bytes.NewBuffer(data)

	if n, cborErr := cboring.ReadArrayLength(buff); cborErr != nil {
		err = cborErr
		return
	} else if n != 2 {
		err = fmt.Errorf("Expected array of length 2, got %d", n)
		return
	}

	if n, cborErr := cboring.ReadUInt(buff); cborErr != nil {
		err = cborErr
		return
	} else {
		switch n {
		case ARTypeStatusReport:
			ar = &StatusReport{}

		default:
			err = fmt.Errorf("Unsupported type code %d", n)
			return
		}

		if cborErr := cboring.Unmarshal(ar, buff); cborErr != nil {
			err = fmt.Errorf("Unmarshalling Content failed: %v", cborErr)
		}
		return
	}
}

// ToCanonicalBlock creates a canonical block, containing this administrative
// record. The surrounding bundle _must_ have a set AdministrativeRecordPayload
// bundle processing control flag.
func AdministrativeRecordToCbor(ar AdministrativeRecord) (blk bundle.CanonicalBlock, err error) {
	buff := new(bytes.Buffer)

	if err = cboring.WriteArrayLength(2, buff); err != nil {
		return
	}

	if err = cboring.WriteUInt(uint64(ar.TypeCode()), buff); err != nil {
		return
	}

	if cborErr := cboring.Marshal(ar, buff); cborErr != nil {
		err = fmt.Errorf("Marshalling Content failed: %v", cborErr)
		return
	}

	blk = bundle.NewCanonicalBlock(1, 0, bundle.NewPayloadBlock(buff.Bytes()))
	return
}
