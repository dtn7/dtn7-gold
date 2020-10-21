// SPDX-FileCopyrightText: 2018, 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"strings"
	"testing"

	"github.com/hashicorp/go-multierror"
)

func TestBundleControlFlagsHas(t *testing.T) {
	var cf = IsFragment | RequestStatusTime

	if !cf.Has(IsFragment) {
		t.Error("cf has no IsFragment-flag even when it was set")
	}

	if cf.Has(StatusRequestDeletion) {
		t.Error("cf has StatusRequestDeletion-flag which was not set")
	}
}

func TestBundleControlFlagsImplications(t *testing.T) {
	var (
		cf BundleControlFlags = 0

		reportReqs = []BundleControlFlags{
			StatusRequestReception,
			StatusRequestForward,
			StatusRequestDelivery,
			StatusRequestDeletion}
	)

	cf |= AdministrativeRecordPayload
	if errs := cf.CheckValid(); errs != nil {
		t.Errorf("Initial set resulted in an invalid state: %v", errs)
	}

	cf |= IsFragment
	if errs := cf.CheckValid(); errs != nil {
		t.Errorf("Unrelated set resulted in an invalid state: %v", errs)
	}

	for _, flg := range reportReqs {
		cf |= flg

		if errs := cf.CheckValid(); errs == nil {
			t.Errorf("Setting %d does not resulted in an failed state", flg)
		} else {
			errFlag := false
			for _, err := range errs.(*multierror.Error).WrappedErrors() {
				if strings.Contains(err.Error(), "administrative record") {
					errFlag = true
				}
			}

			if !errFlag {
				t.Errorf("No error contained a correct message")
			}
		}

		cf &^= flg
		if errs := cf.CheckValid(); errs != nil {
			t.Errorf("Resetting %d does not resolved in a valid state: %v", flg, errs)
		}
	}

	for _, flg := range reportReqs {
		cf |= flg
	}

	if errs := cf.CheckValid(); errs == nil {
		t.Errorf("Setting all report flags should result in an invalid state")
	}
}
