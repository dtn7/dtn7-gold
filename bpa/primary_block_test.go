package bpa

import "testing"

func setupPrimaryBlock() PrimaryBlock {
	bcf := BndlCFBundleDeletionStatusReportsAreRequested |
		BndlCFBundleDeliveryStatusReportsAreRequested |
		BndlCFBundleMustNotBeFragmented

	destination, _ := NewEndpointID("dtn", "foobar")
	source, _ := NewEndpointID("dtn", "me")

	creationTimestamp := NewCreationTimestamp(DTNTimeNow(), 0)
	lifetime := uint(10 * 60 * 1000)

	return NewPrimaryBlock(bcf, *destination, *source, creationTimestamp, lifetime)
}

func TestNewPrimaryBlock(t *testing.T) {
	pb := setupPrimaryBlock()

	if pb.HasCRC() {
		t.Error("Primary Block has no CRC, but says so")
	}

	if pb.HasFragmentation() {
		t.Error("Primary Block has no fragmentation, but says so")
	}
}

func TestPrimaryBlockCRC(t *testing.T) {
	pb := setupPrimaryBlock()
	pb.CRCType = CRC16

	if !pb.HasCRC() {
		t.Error("Primary Block should need a CRC")
	}
}

func TestPrimaryBlockFragmentation(t *testing.T) {
	pb := setupPrimaryBlock()
	pb.BundleControlFlags = BndlCFBundleIsAFragment

	if !pb.HasFragmentation() {
		t.Error("Primary Block should be fragmented")
	}
}
