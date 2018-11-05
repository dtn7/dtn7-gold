package bpa

import (
	"strings"
	"testing"
)

func TestBundleControlFlagsNew(t *testing.T) {
	var cf BundleControlFlags = NewBundleControlFlags()
	if cf != 0 {
		t.Errorf("NewBundleControlFlags returned %d instead of 0", uint16(cf))
	}
}

func TestBundleControlFlagsSet(t *testing.T) {
	var (
		cf  BundleControlFlags = NewBundleControlFlags()
		err error
	)

	err = cf.Set(BndlCFBundleIsAFragment)
	if err != nil {
		t.Error(err)
	}
	if cf != 0x0001 {
		t.Errorf("cf is %d after first Set instead of 1", uint16(cf))
	}

	err = cf.Set(BndlCFStatusTimeIsRequestedInAllStatusReports)
	if err != nil {
		t.Error(err)
	}
	if cf != (1<<0)|(1<<6) {
		t.Errorf("cf is %d after second Set instead of 65", uint16(cf))
	}

	err = cf.Set(1 << 3)
	if err == nil {
		t.Error("Setting reserved bit 0x0008 returned no error")
	}
	if !strings.Contains(err.Error(), "reserved bits") {
		t.Errorf("Error does not mention \"reserved bits\": %v", err)
	}

	err = cf.Set(0)
	if err == nil {
		t.Error("Setting no bits returned no error")
	}
	if !strings.Contains(err.Error(), "one bit") {
		t.Errorf("Error does not mention \"one bit\" only: %v", err)
	}

	err = cf.Set(BndlCFBundleContainsAManifestBlock | BndlCFBundleIsAFragment)
	if err == nil {
		t.Error("Setting multiple bits returned no error")
	}
	if !strings.Contains(err.Error(), "one bit") {
		t.Errorf("Error does not mention \"one bit\" only: %v", err)
	}
}

func TestBundleControlFlagsUnset(t *testing.T) {
	var (
		cf  BundleControlFlags = NewBundleControlFlags()
		err error
	)

	// Tested in TestBundleControlFlagsSet
	cf.Set(BndlCFBundleIsAFragment)
	cf.Set(BndlCFStatusTimeIsRequestedInAllStatusReports)

	err = cf.Unset(BndlCFBundleDeletionStatusReportsAreRequested)
	if err != nil {
		t.Error(err)
	}
	if cf != (1<<0)|(1<<6) {
		t.Errorf("cf was changed even if an unset flag was altered: %d", cf)
	}

	err = cf.Unset(BndlCFStatusTimeIsRequestedInAllStatusReports)
	if err != nil {
		t.Error(err)
	}
	if cf != BndlCFBundleIsAFragment {
		t.Errorf("cf is %d instead of %d", cf, BndlCFBundleIsAFragment)
	}
}

func TestBundleControlFlagsHas(t *testing.T) {
	var cf BundleControlFlags = NewBundleControlFlags()

	// Tested in TestBundleControlFlagsSet
	cf.Set(BndlCFBundleIsAFragment)
	cf.Set(BndlCFStatusTimeIsRequestedInAllStatusReports)

	if !cf.Has(BndlCFBundleIsAFragment) {
		t.Error("cf has no BndlCFBundleIsAFragment-flag even when it was set")
	}

	if cf.Has(BndlCFBundleDeletionStatusReportsAreRequested) {
		t.Error("cf has BndlCFBundleDeletionStatusReportsAreRequested-flag which was not set")
	}
}
