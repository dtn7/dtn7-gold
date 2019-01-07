package bundle

import "github.com/hashicorp/go-multierror"

// BundleControlFlags is an uint16 which represents the Bundle Processing
// Control Flags as specified in section 4.1.3.
type BundleControlFlags uint16

const (
	// StatusRequestDeletion: Request reporting of bundle deletion.
	StatusRequestDeletion BundleControlFlags = 0x1000

	// StatusRequestDelivery: Request reporting of bundle delivery.
	StatusRequestDelivery BundleControlFlags = 0x0800

	// StatusRequestForward: Request reporting of bundle forwarding.
	StatusRequestForward BundleControlFlags = 0x0400

	// StatusRequestReception: Request reporting of bundle reception.
	StatusRequestReception BundleControlFlags = 0x0100

	// ContainsManifest: The bundle contains a "manifest" extension block.
	ContainsManifest BundleControlFlags = 0x0080

	// RequestStatusTime: Status time is requested in all status reports.
	RequestStatusTime BundleControlFlags = 0x0040

	// RequestUserApplicationAck: Acknowledgment by the user application
	// is requested.
	RequestUserApplicationAck BundleControlFlags = 0x0020

	// MustNotFragmented: The bundle must not be fragmented.
	MustNotFragmented BundleControlFlags = 0x0004

	// AdministrativeRecordPayload: The bundle's payload is an
	// administrative record.
	AdministrativeRecordPayload BundleControlFlags = 0x0002

	// IsFragment: The bundle is a fragment.
	IsFragment BundleControlFlags = 0x0001

	bndlCFReservedFields BundleControlFlags = 0xE218
)

// Has returns true if a given flag or mask of flags is set.
func (bcf BundleControlFlags) Has(flag BundleControlFlags) bool {
	return (bcf & flag) != 0
}

func (bcf BundleControlFlags) checkValid() (errs error) {
	if bcf.Has(bndlCFReservedFields) {
		errs = multierror.Append(
			errs, newBundleError(
				"BundleControlFlags: Given flag contains reserved bits"))
	}

	if bcf.Has(IsFragment) && bcf.Has(MustNotFragmented) {
		errs = multierror.Append(errs,
			newBundleError("BundleControlFlags: both 'bundle is a fragment' and "+
				"'bundle must not be fragmented' flags are set"))
	}

	// payload is administrative record => no status report request flags
	adminRecCheck := !bcf.Has(AdministrativeRecordPayload) ||
		(!bcf.Has(StatusRequestReception) &&
			!bcf.Has(StatusRequestForward) &&
			!bcf.Has(StatusRequestDelivery) &&
			!bcf.Has(StatusRequestDeletion))
	if !adminRecCheck {
		errs = multierror.Append(errs, newBundleError(
			"BundleControlFlags: \"payload is administrative record => "+
				"no status report request flags\" failed"))
	}

	return
}
