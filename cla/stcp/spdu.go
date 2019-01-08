package stcp

import (
	"fmt"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

// Data Unit represents a STCP Data Unit, which will be decoded as a CBOR
// array of the serialized bundle's length and the serialized bundle.
type DataUnit struct {
	_struct struct{} `codec:",toarray"`

	Length    uint
	EncBundle []byte
}

// newDataUnit creates a STCP Data Unit (SPDU) based on a given bundle.
func newDataUnit(bndl bundle.Bundle) DataUnit {
	encBundle := bndl.ToCbor()

	return DataUnit{
		Length:    uint(len(encBundle)),
		EncBundle: encBundle,
	}
}

// newDataUnitFromCbor tries to create a new STCP Data Unit (SPDU) from the
// given CBOR array, represented as a byte string.
func newDataUnitFromCbor(data []byte) (s DataUnit, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	err = codec.NewDecoderBytes(data, new(codec.CborHandle)).Decode(&s)
	return
}

// toCbor converts the STCP Data Unit (SPDU) to a CBOR array.
func (du DataUnit) toCbor() []byte {
	var b = make([]byte, 0, 64)
	codec.NewEncoderBytes(&b, new(codec.CborHandle)).MustEncode(du)

	return b
}

// toBundle returns the encapsulated bundle.
func (du DataUnit) toBundle() (b bundle.Bundle, err error) {
	if du.Length != uint(len(du.EncBundle)) {
		err = fmt.Errorf("Length variable and bundle's length mismatch: %d != %d",
			du.Length, len(du.EncBundle))
	}

	b, err = bundle.NewBundleFromCbor(du.EncBundle)
	return
}
