package bundle

import (
	"encoding/hex"
	"testing"
)

func TestBundleBuilderSimple(t *testing.T) {
	bndl, err := Builder().
		CRC(CRC32).
		Source("dtn://myself/").
		Destination("dtn://dest/").
		CreationTimestampNow().
		Lifetime("10m").
		HopCountBlock(64).
		PayloadBlock("hello world!").
		Build()

	if err != nil {
		t.Fatalf("Builder errored: %v", err)
	}

	t.Log(hex.EncodeToString(bndl.ToCbor()))

	/*
		// Something is fishy..
		_, err = NewBundleFromCbor(bndl.ToCbor())
		if err != nil {
			t.Fatalf("Parsing CBOR encoded Bundle errored: %v", err)
		}
	*/
}
