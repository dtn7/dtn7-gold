package bpa

import (
	"strings"
	"testing"
)

func TestBlockControlFlagsNew(t *testing.T) {
	var cf BlockControlFlags = NewBlockControlFlags()
	if cf != 0 {
		t.Errorf("NewBlockControlFlags returned %d instead of 0", uint16(cf))
	}
}

func TestBlockControlFlagsSet(t *testing.T) {
	var (
		cf  BlockControlFlags = NewBlockControlFlags()
		err error
	)

	err = cf.Set(BlckCFBlockMustBeReplicatedInEveryFragment)
	if err != nil {
		t.Error(err)
	}
	if cf != 0x01 {
		t.Errorf("cf is %d after setting BlckCFBlockMustBeReplicatedInEveryFragment", cf)
	}

	err = cf.Set(BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed)
	if err != nil {
		t.Error(err)
	}
	if cf != (1 | (1 << 3)) {
		t.Errorf("cf is %d instead of %d", cf, 1|(1<<3))
	}

	err = cf.Set(1 << 4)
	if err == nil {
		t.Error("Setting reserved bit returned no error")
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

	err = cf.Set(BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed | BlckCFBlockMustBeReplicatedInEveryFragment)
	if err == nil {
		t.Error("Setting multiple bits returned no error")
	}
	if !strings.Contains(err.Error(), "one bit") {
		t.Errorf("Error does not mention \"one bit\" only: %v", err)
	}
}

func TestBlockControlFlagsUnset(t *testing.T) {
	var (
		cf  BlockControlFlags = NewBlockControlFlags()
		err error
	)

	// Tested in TestBlockControlFlagsSet
	cf.Set(BlckCFBlockMustBeReplicatedInEveryFragment)
	cf.Set(BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed)

	err = cf.Unset(BlckCFStatusReportMustBeTransmittedIfBlockCannotBeProcessed)
	if err != nil {
		t.Error(err)
	}
	if cf != (1<<0)|(1<<3) {
		t.Errorf("cf was changed even if an unset flag was altered: %d", cf)
	}

	err = cf.Unset(BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed)
	if err != nil {
		t.Error(err)
	}
	if cf != BlckCFBlockMustBeReplicatedInEveryFragment {
		t.Errorf("cf is %d instead of %d", cf, BlckCFBlockMustBeReplicatedInEveryFragment)
	}
}

func TestBlockControlFlagsHas(t *testing.T) {
	var cf BlockControlFlags = NewBlockControlFlags()

	// Tested in TestBlockControlFlagsSet
	cf.Set(BlckCFBlockMustBeReplicatedInEveryFragment)
	cf.Set(BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed)

	if !cf.Has(BlckCFBlockMustBeReplicatedInEveryFragment) {
		t.Error("cf has no BlckCFBlockMustBeReplicatedInEveryFragment-flag even when it was set")
	}

	if cf.Has(BlckCFBlockMustBeRemovedFromBundleIfItCannotBeProcessed) {
		t.Error("cf has BlckCFBlockMustBeRemovedFromBundleIfItCannotBeProcessed-flag which was not set")
	}
}
