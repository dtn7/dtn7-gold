package stcp

import (
	"fmt"

	"github.com/geistesk/dtn7/bundle"
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

// toBundle returns the encapsulated bundle.
func (du DataUnit) toBundle() (b bundle.Bundle, err error) {
	if du.Length != uint(len(du.EncBundle)) {
		err = fmt.Errorf("Length variable and bundle's length mismatch: %d != %d",
			du.Length, len(du.EncBundle))
		return
	}

	b, err = bundle.NewBundleFromCbor(du.EncBundle)
	return
}
