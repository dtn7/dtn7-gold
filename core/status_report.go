package core

import (
	"fmt"
	"strings"

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

		bsi.StatusRequested = false

	case 2:
		bsi.Asserted = arr[0].(bool)
		bsi.Time = bundle.DtnTime(arr[1].(uint64))

		bsi.StatusRequested = true

	default:
		panic("arr has wrong length, neither 1 nor 2")
	}
}

func (bsi BundleStatusItem) String() string {
	if !bsi.Asserted {
		return fmt.Sprintf("BundleStatusItem(%t)", bsi.Asserted)
	} else {
		return fmt.Sprintf("BundleStatusItem(%t, %v)", bsi.Asserted, bsi.Time)
	}
}

// NewBundleStatusItem returns a new BundleStatusItem, indicating an optional
// assertion - givenas asserted -, but no status time request.
func NewBundleStatusItem(asserted bool) BundleStatusItem {
	return BundleStatusItem{
		Asserted:        asserted,
		Time:            bundle.DtnTimeEpoch,
		StatusRequested: false,
	}
}

// NewTimeReportingBundleStatusItem returns a new BundleStatusItem, indicating
// both a positive assertion and a requested status time report.
func NewTimeReportingBundleStatusItem(time bundle.DtnTime) BundleStatusItem {
	return BundleStatusItem{
		Asserted:        true,
		Time:            time,
		StatusRequested: true,
	}
}

// StatusReportReason is the bundle status report reason code, which is used as
// the second element of the bundle status report array.
type StatusReportReason uint

const (
	// NoInformation is the "No additional information" bundle status report
	// reason code.
	NoInformation StatusReportReason = 0

	// LifetimeExpired is the "Lifetime expired" bundle status report reason code.
	LifetimeExpired StatusReportReason = 1

	// ForwardUnidirectionalLink is the "Forwarded over unidirectional link"
	// bundle status report reason code.
	ForwardUnidirectionalLink StatusReportReason = 2

	// TransmissionCanceled is the "Transmission canceled" bundle status report
	// reason code.
	TransmissionCanceled StatusReportReason = 3

	// DepletedStorage is the "Depleted storage" bundle status report reason code.
	DepletedStorage StatusReportReason = 4

	// DestEndpointUnintelligible is the "Destination endpoint ID unintelligible"
	// bundle status report reason code.
	DestEndpointUnintelligible StatusReportReason = 5

	// NoRouteToDestination is the "No known route to destination from here"
	// bundle status report reason code.
	NoRouteToDestination StatusReportReason = 6

	// NoNextNodeContact is the "No timely contact with next node on route" bundle
	// status report reason code.
	NoNextNodeContact StatusReportReason = 7

	// BlockUnintelligible is the "Block unintelligible" bundle status report
	// reason code.
	BlockUnintelligible StatusReportReason = 8

	// HopLimitExceeded is the "Hop limit exceeded" bundle status report reason
	// code.
	HopLimitExceeded StatusReportReason = 9
)

func (srr StatusReportReason) String() string {
	switch srr {
	case NoInformation:
		return "No additional information"

	case LifetimeExpired:
		return "Lifetime expired"

	case ForwardUnidirectionalLink:
		return "Forward over unidirectional link"

	case TransmissionCanceled:
		return "Transmission canceled"

	case DepletedStorage:
		return "Depleted storage"

	case DestEndpointUnintelligible:
		return "Destination endpoint ID unintelligible"

	case NoRouteToDestination:
		return "No known route to destination from here"

	case NoNextNodeContact:
		return "No timely contact with next node on route"

	case BlockUnintelligible:
		return "Block unintelligible"

	case HopLimitExceeded:
		return "Hop limit exceeded"

	default:
		return "unknown"
	}
}

// StatusInformationPos describes the different bundle status information
// entries. Each bundle status report must contain at least the following
// bundle status items.
type StatusInformationPos int

const (
	// maxStatusInformationPos is the amount of different StatusInformationPos.
	maxStatusInformationPos int = 4

	// ReceivedBundle is the first bundle status information entry, indicating
	// the reporting node received this bundle.
	ReceivedBundle StatusInformationPos = 0

	// ForwardedBundle is the second bundle status information entry, indicating
	// the reporting node forwarded this bundle.
	ForwardedBundle StatusInformationPos = 1

	// DeliveredBundle is the third bundle status information entry, indicating
	// the reporting node delivered this bundle.
	DeliveredBundle StatusInformationPos = 2

	// DeletedBundle is the fourth bundle status information entry, indicating
	// the reporting node deleted this bundle.
	DeletedBundle StatusInformationPos = 3
)

func (sip StatusInformationPos) String() string {
	switch sip {
	case ReceivedBundle:
		return "received bundle"

	case ForwardedBundle:
		return "forwarded bundle"

	case DeliveredBundle:
		return "delivered bundle"

	case DeletedBundle:
		return "deleted bundle"

	default:
		return "unknown"
	}
}

// StatusReport is the bundle status report, used in an administrative record.
type StatusReport struct {
	_struct struct{} `codec:",toarray"`

	StatusInformation []BundleStatusItem
	ReportReason      StatusReportReason
	SourceNode        bundle.EndpointID
	Timestamp         bundle.CreationTimestamp
}

// NewStatusReport creates a bundle status report for the given bundle and
// StatusInformationPos, which creates the right bundle status item. The
// bundle status report reason code will be used and the bundle status item
// gets the given timestamp.
func NewStatusReport(bndl bundle.Bundle, statusItem StatusInformationPos,
	reason StatusReportReason, time bundle.DtnTime) StatusReport {
	var sr = StatusReport{
		StatusInformation: make([]BundleStatusItem, maxStatusInformationPos),
		ReportReason:      reason,
		SourceNode:        bndl.PrimaryBlock.SourceNode,
		Timestamp:         bndl.PrimaryBlock.CreationTimestamp,
	}

	for i := 0; i < maxStatusInformationPos; i++ {
		sip := StatusInformationPos(i)

		switch {
		case sip == statusItem && bndl.PrimaryBlock.BundleControlFlags.Has(bundle.RequestStatusTime):
			sr.StatusInformation[i] = NewTimeReportingBundleStatusItem(time)

		case sip == statusItem:
			sr.StatusInformation[i] = NewBundleStatusItem(true)

		default:
			sr.StatusInformation[i] = NewBundleStatusItem(false)
		}
	}

	return sr
}

func (sr StatusReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "StatusReport([")

	for i := 0; i < len(sr.StatusInformation); i++ {
		si := sr.StatusInformation[i]
		sip := StatusInformationPos(i)

		if !si.Asserted {
			continue
		}

		if si.Time == bundle.DtnTimeEpoch {
			fmt.Fprintf(&b, "%v,", sip)
		} else {
			fmt.Fprintf(&b, "%v %v,", sip, si.Time)
		}
	}
	fmt.Fprintf(&b, "], ")

	fmt.Fprintf(&b, "%v, %v, %v", sr.ReportReason, sr.SourceNode, sr.Timestamp)

	return b.String()
}
