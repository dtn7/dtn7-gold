package bpa

import (
	"bytes"
	"fmt"
	"testing"
)

func TestBundleApplyCRC(t *testing.T) {
	var epPrim, _ = NewEndpointID("dtn", "foo/bar")
	var creationTs = NewCreationTimestamp(DTNTimeNow(), 23)

	var primary = NewPrimaryBlock(
		BndlCFBundleDeliveryStatusReportsAreRequested,
		*epPrim, *epPrim, creationTs, 42000)

	var epPrev, _ = NewEndpointID("ipn", "23.42")
	var prevNode = NewPreviousNodeBlock(1, 0, *epPrev)

	var payload = NewPayloadBlock(
		BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed, []byte("GuMo"))

	var bundle = NewBundle(
		primary, []CanonicalBlock{prevNode, payload})

	for _, crcTest := range []CRCType{CRCNo, CRC16, CRC32, CRCNo} {
		bundle.ApplyCRC(crcTest)

		if ty := bundle.PrimaryBlock.GetCRCType(); ty != crcTest {
			t.Errorf("Bundle's primary block has wrong CRCType, %v instead of %v",
				ty, crcTest)
		}

		if !CheckCRC(&bundle.PrimaryBlock) {
			t.Errorf("For %v the primary block's CRC mismatchs", crcTest)
		}

		for _, cb := range bundle.CanonicalBlocks {
			if !CheckCRC(&cb) {
				t.Errorf("For %v a canonical block's CRC mismatchs", crcTest)
			}
		}
	}
}

func TestBundleCbor(t *testing.T) {
	var epDest, _ = NewEndpointID("dtn", "desty")
	var epSource, _ = NewEndpointID("dtn", "gumo")
	var creationTs = NewCreationTimestamp(DTNTimeNow(), 23)

	var primary = NewPrimaryBlock(
		BndlCFBundleDeliveryStatusReportsAreRequested,
		*epDest, *epSource, creationTs, 42000)

	var epPrev, _ = NewEndpointID("ipn", "23.42")
	var prevNode = NewPreviousNodeBlock(23, 0, *epPrev)

	var payload = NewPayloadBlock(
		BlckCFBundleMustBeDeletedIfBlockCannotBeProcessed,
		[]byte("GuMo meine Kernel"))

	bundle1 := NewBundle(
		primary, []CanonicalBlock{prevNode, payload})
	bundle1.ApplyCRC(CRC32)

	bundle1Cbor := bundle1.ToCbor()

	bundle2 := NewBundleFromCbor(bundle1Cbor)
	bundle2Cbor := bundle2.ToCbor()

	if !bytes.Equal(bundle1Cbor, bundle2Cbor) {
		t.Errorf("Cbor-Representations do not match:\n- %v\n- %v",
			bundle1Cbor, bundle2Cbor)
	}

	s1 := fmt.Sprintf("%v", bundle1)
	s2 := fmt.Sprintf("%v", bundle2)

	if s1 != s2 {
		t.Errorf("String representations do not match:%v and %v", s1, s2)
	}
}
