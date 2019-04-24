package mtcp

import (
	"github.com/geistesk/dtn7/bundle"
)

// DataUnit represents a MTCP Data Unit, which is an encapsulated bundle inside
// a byte string (byte array) with an definite length.
type DataUnit []byte

// newDataUnit creates a MTCP Data Unit (MPDU) based on a given bundle.
func newDataUnit(bndl bundle.Bundle) DataUnit {
	return bndl.ToCbor()
}

// toBundle returns the encapsulated bundle.
func (du DataUnit) toBundle() (bundle.Bundle, error) {
	return bundle.NewBundleFromCbor(du)
}
