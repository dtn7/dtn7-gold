// SPDX-FileCopyrightText: 2018, 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// BundleControlFlags is an uint which represents the Bundle Processing
// Control Flags as specified in section 4.1.3.
type BundleControlFlags uint64

const (
	// IsFragment indicates this bundle is a fragment.
	IsFragment BundleControlFlags = 0x000001

	// AdministrativeRecordPayload indicates the payload is an administrative record.
	AdministrativeRecordPayload BundleControlFlags = 0x000002

	// MustNotFragmented forbids bundle fragmentation.
	MustNotFragmented BundleControlFlags = 0x000004

	// RequestUserApplicationAck requests an acknowledgement from the application agent.
	RequestUserApplicationAck BundleControlFlags = 0x000020

	// RequestStatusTime requests a status time in all status reports.
	RequestStatusTime BundleControlFlags = 0x000040

	// StatusRequestReception requests a bundle reception status report.
	StatusRequestReception BundleControlFlags = 0x004000

	// StatusRequestForward requests a bundle forwarding status report.
	StatusRequestForward BundleControlFlags = 0x010000

	// StatusRequestDelivery requests a bundle delivery status report.
	StatusRequestDelivery BundleControlFlags = 0x020000

	// StatusRequestDeletion requests a bundle deletion status report.
	StatusRequestDeletion BundleControlFlags = 0x040000
)

// Has returns true if a given flag or mask of flags is set.
func (bcf BundleControlFlags) Has(flag BundleControlFlags) bool {
	return (bcf & flag) != 0
}

// CheckValid returns an array of errors for incorrect data.
func (bcf BundleControlFlags) CheckValid() (errs error) {
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

// Strings returns an array of all flags as a string representation.
func (bcf BundleControlFlags) Strings() (fields []string) {
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
		{MustNotFragmented, "MUST_NOT_BE_FRAGMENTED"},
		{AdministrativeRecordPayload, "ADMINISTRATIVE_PAYLOAD"},
		{IsFragment, "IS_FRAGMENT"},
	}

	for _, check := range checks {
		if bcf.Has(check.field) {
			fields = append(fields, check.text)
		}
	}

	return
}

// MarshalJSON creates a JSON array of control flags.
func (bcf BundleControlFlags) MarshalJSON() ([]byte, error) {
	return json.Marshal(bcf.Strings())
}

func (bcf BundleControlFlags) String() string {
	return strings.Join(bcf.Strings(), ",")
}
