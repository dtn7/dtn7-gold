package bundle

import "github.com/hashicorp/go-multierror"

// BundleControlFlags is an uint16 which represents the Bundle Processing
// Control Flags as specified in section 4.1.3.
type BundleControlFlags uint16

const (
	BndlCFBundleDeletionStatusReportsAreRequested   BundleControlFlags = 0x1000
	BndlCFBundleDeliveryStatusReportsAreRequested   BundleControlFlags = 0x0800
	BndlCFBundleForwardingStatusReportsAreRequested BundleControlFlags = 0x0400
	BndlCFBundleReceptionStatusReportsAreRequested  BundleControlFlags = 0x0100
	BndlCFBundleContainsAManifestBlock              BundleControlFlags = 0x0080
	BndlCFStatusTimeIsRequestedInAllStatusReports   BundleControlFlags = 0x0040
	BndlCFUserApplicationAcknowledgementIsRequested BundleControlFlags = 0x0020
	BndlCFBundleMustNotBeFragmented                 BundleControlFlags = 0x0004
	BndlCFPayloadIsAnAdministrativeRecord           BundleControlFlags = 0x0002
	BndlCFBundleIsAFragment                         BundleControlFlags = 0x0001
	bndlCFReservedFields                            BundleControlFlags = 0xE218
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

	if bcf.Has(BndlCFBundleIsAFragment) && bcf.Has(BndlCFBundleMustNotBeFragmented) {
		errs = multierror.Append(errs,
			newBundleError("BundleControlFlags: both 'bundle is a fragment' and "+
				"'bundle must not be fragmented' flags are set"))
	}

	// payload is administrative record => no status report request flags
	adminRecCheck := !bcf.Has(BndlCFPayloadIsAnAdministrativeRecord) ||
		(!bcf.Has(BndlCFBundleReceptionStatusReportsAreRequested) &&
			!bcf.Has(BndlCFBundleForwardingStatusReportsAreRequested) &&
			!bcf.Has(BndlCFBundleDeliveryStatusReportsAreRequested) &&
			!bcf.Has(BndlCFBundleDeletionStatusReportsAreRequested))
	if !adminRecCheck {
		errs = multierror.Append(errs, newBundleError(
			"BundleControlFlags: \"payload is administrative record => "+
				"no status report request flags\" failed"))
	}

	return
}
