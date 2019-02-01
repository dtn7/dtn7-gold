package core

import (
	"fmt"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

// AdministrativeRecordTypeCode specifies the type of an AdministrativeRecord.
// However, currently the Bundle Status Report is the only known type.
type AdministrativeRecordTypeCode uint

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
	_struct struct{} `codec:",toarray"`

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

// NewAdministrativeRecordFromCbor creates a new AdministrativeRecord from
// a given byte array.
func NewAdministrativeRecordFromCbor(data []byte) (ar AdministrativeRecord, err error) {
	var dec = codec.NewDecoderBytes(data, new(codec.CborHandle))
	err = dec.Decode(&ar)

	return
}

// ToCanonicalBlock creates a canonical block, containing this administrative
// record. The surrounding bundle _must_ have a set AdministrativeRecordPayload
// bundle processing control flag.
func (ar AdministrativeRecord) ToCanonicalBlock() bundle.CanonicalBlock {
	var data []byte
	codec.NewEncoderBytes(&data, new(codec.CborHandle)).Encode(ar)

	return bundle.NewCanonicalBlock(bundle.PayloadBlock, 0, 0, data)
}

func (ar AdministrativeRecord) String() string {
	return fmt.Sprintf("AdministrativeRecord(%s, %v)", ar.TypeCode, ar.Content)
}
