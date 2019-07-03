package arecord

import (
	"bytes"
	"fmt"
	"io"

	"github.com/dtn7/cboring"
	"github.com/dtn7/dtn7-go/bundle"
)

// AdministrativeRecordTypeCode specifies the type of an AdministrativeRecord.
// However, currently the Bundle Status Report is the only known type.
type AdministrativeRecordTypeCode uint64

const (
	// BundleStatusReportTypeCode is the Bundle Status Report's type code, used in
	// its parent Administrative Record.
	BundleStatusReportTypeCode AdministrativeRecordTypeCode = 1
)

func (artc AdministrativeRecordTypeCode) String() string {
	switch artc {
	case BundleStatusReportTypeCode:
		return "bundle status report"

	default:
		return "unknown"
	}
}

// AdministrativeRecord is a application data unit used for administrative
// records. There is only one kind of Administrative Record definded today: the
// Bundle Status Report. Therefore the Content field is currently of the
// StatusReport type. Otherwise the CBOR en- and decoding would have been done
// by hand. So this becomes work for future me.
type AdministrativeRecord struct {
	TypeCode AdministrativeRecordTypeCode
	Content  StatusReport
}

// NewAdministrativeRecord generates a new Administrative Record based on the
// given parameters.
func NewAdministrativeRecord(typeCode AdministrativeRecordTypeCode, content StatusReport) AdministrativeRecord {
	return AdministrativeRecord{
		TypeCode: typeCode,
		Content:  content,
	}
}

func (ar *AdministrativeRecord) MarshalCbor(w io.Writer) error {
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	if err := cboring.WriteUInt(uint64(ar.TypeCode), w); err != nil {
		return err
	}

	if err := cboring.Marshal(&ar.Content, w); err != nil {
		return fmt.Errorf("Marshalling Content failed: %v", err)
	}

	return nil
}

func (ar *AdministrativeRecord) UnmarshalCbor(r io.Reader) error {
	if n, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if n != 2 {
		return fmt.Errorf("Expected array of length 2, got %d", n)
	}

	if n, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		ar.TypeCode = AdministrativeRecordTypeCode(n)
	}

	if err := cboring.Unmarshal(&ar.Content, r); err != nil {
		return fmt.Errorf("Unmarshalling Content failed: %v", err)
	}

	return nil
}

// NewAdministrativeRecordFromCbor creates a new AdministrativeRecord from
// a given byte array.
func NewAdministrativeRecordFromCbor(data []byte) (ar AdministrativeRecord, err error) {
	err = cboring.Unmarshal(&ar, bytes.NewBuffer(data))
	return
}

// ToCanonicalBlock creates a canonical block, containing this administrative
// record. The surrounding bundle _must_ have a set AdministrativeRecordPayload
// bundle processing control flag.
func (ar AdministrativeRecord) ToCanonicalBlock() bundle.CanonicalBlock {
	buff := new(bytes.Buffer)
	cboring.Marshal(&ar, buff)

	return bundle.NewCanonicalBlock(1, 0, bundle.NewPayloadBlock(buff.Bytes()))
}

func (ar AdministrativeRecord) String() string {
	return fmt.Sprintf("AdministrativeRecord(%s, %v)", ar.TypeCode, ar.Content)
}
