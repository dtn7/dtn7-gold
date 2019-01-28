package core

import (
	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

// BundleStatusItem represents the a bundle status item, as used as an element
// in the bundle status information array of each Bundle Status Report.
type BundleStatusItem struct {
	Asserted        bool
	Time            bundle.DtnTime
	StatusRequested bool
}

func (bsi BundleStatusItem) CodecEncodeSelf(enc *codec.Encoder) {
	var arr = []interface{}{bsi.Asserted}

	if bsi.Asserted && bsi.StatusRequested {
		arr = append(arr, bsi.Time)
	}

	enc.MustEncode(arr)
}

func (bsi *BundleStatusItem) CodecDecodeSelf(dec *codec.Decoder) {
	var arrPt = new([]interface{})
	dec.MustDecode(arrPt)

	var arr = *arrPt

	switch len(arr) {
	case 1:
		bsi.Asserted = arr[0].(bool)

	case 2:
		bsi.Asserted = arr[0].(bool)
		bsi.Time = bundle.DtnTime(arr[1].(uint64))

	default:
		panic("arr has wrong length, neither 1 nor 2")
	}
}

// NewNegativeBundleStatusItem returns a new BundleStatusItem, indicating a
// negative assertion and/or no status time requested.
func NewNegativeBundleStatusItem() BundleStatusItem {
	return BundleStatusItem{
		Asserted:        false,
		Time:            bundle.DtnTimeEpoch,
		StatusRequested: false,
	}
}

// NewReportingBundleStatusItem returns a new BundleStatusItem, indicating both
// a positive assertion and a requested status time report.
func NewReportingBundleStatusItem(time bundle.DtnTime) BundleStatusItem {
	return BundleStatusItem{
		Asserted:        true,
		Time:            time,
		StatusRequested: true,
	}
}

type StatusReport struct {
	// TODO
}
