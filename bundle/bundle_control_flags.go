package bundle

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// BundleControlFlags is an uint which represents the Bundle Processing
// Control Flags as specified in section 4.1.3.
type BundleControlFlags uint64

const (
	// IsFragment: bundle is a fragment.
	IsFragment BundleControlFlags = 0x000001

	// AdministrativeRecordPayload: payload is an administrative record.
	AdministrativeRecordPayload BundleControlFlags = 0x000002

	// MustNotFragmented: bundle must not be fragmented.
	MustNotFragmented BundleControlFlags = 0x000004

	// RequestUserApplicationAck: user application acknowledgement is requested.
	RequestUserApplicationAck BundleControlFlags = 0x000020

	// RequestStatusTime: status time is requested in all status reports.
	RequestStatusTime BundleControlFlags = 0x000040

	// StatusRequestReception: bundle reception status reports are requested.
	StatusRequestReception BundleControlFlags = 0x004000

	// StatusRequestForward: bundle forwarding status reports are requested.
	StatusRequestForward BundleControlFlags = 0x010000

	// StatusRequestDelivery: bundle delivery status reports are requested.
	StatusRequestDelivery BundleControlFlags = 0x020000

	// StatusRequestDeletion: bundle deletion status reports are requested.
	StatusRequestDeletion BundleControlFlags = 0x040000

	// bndlCFReservedFields are both reserved and unassigned flags.
	bndlCFReservedFields BundleControlFlags = 0xFFFFFFFFFFF8BF98
)

// Has returns true if a given flag or mask of flags is set.
func (bcf BundleControlFlags) Has(flag BundleControlFlags) bool {
	return (bcf & flag) != 0
}

func (bcf BundleControlFlags) CheckValid() (errs error) {
	if bcf.Has(bndlCFReservedFields) {
		errs = multierror.Append(
			errs, fmt.Errorf("BundleControlFlags: Given flag contains reserved bits"))
	}

	if bcf.Has(IsFragment) && bcf.Has(MustNotFragmented) {
		errs = multierror.Append(errs,
			fmt.Errorf("BundleControlFlags: both 'bundle is a fragment' and "+
				"'bundle must not be fragmented' flags are set"))
	}

	// payload is administrative record => no status report request flags
	adminRecCheck := !bcf.Has(AdministrativeRecordPayload) ||
		(!bcf.Has(StatusRequestReception) &&
			!bcf.Has(StatusRequestForward) &&
			!bcf.Has(StatusRequestDelivery) &&
			!bcf.Has(StatusRequestDeletion))
	if !adminRecCheck {
		errs = multierror.Append(errs, fmt.Errorf(
			"BundleControlFlags: \"payload is administrative record => "+
				"no status report request flags\" failed"))
	}

	return
}

func (bcf BundleControlFlags) String() string {
	var fields []string

	checks := []struct {
		field BundleControlFlags
		text  string
	}{
		{StatusRequestDeletion, "REQUESTED_DELETION_STATUS_REPORT"},
		{StatusRequestDelivery, "REQUESTED_DELIVERY_STATUS_REPORT"},
		{StatusRequestForward, "REQUESTED_FORWARD_STATUS_REPORT"},
		{StatusRequestReception, "REQUESTED_RECEPTION_STATUS_REPORT"},
		{RequestStatusTime, "REQUESTED_TIME_IN_STATUS_REPORT"},
		{RequestUserApplicationAck, "REQUESTED_APPLICATION_ACK"},
		{MustNotFragmented, "MUST_NO_BE_FRAGMENTED"},
		{AdministrativeRecordPayload, "ADMINISTRATIVE_PAYLOAD"},
		{IsFragment, "IS_FRAGMENT"},
	}

	for _, check := range checks {
		if bcf.Has(check.field) {
			fields = append(fields, check.text)
		}
	}

	return strings.Join(fields, ",")
}
