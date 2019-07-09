package bundle

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/dtn7/cboring"
)

func TestBundleBuilderSimple(t *testing.T) {
	bndl, err := Builder().
		CRC(CRC32).
		Source("dtn://myself/").
		Destination("dtn://dest/").
		CreationTimestampEpoch().
		Lifetime("10m").
		HopCountBlock(64).
		BundleAgeBlock(0).
		PayloadBlock([]byte("hello world!")).
		Build()

	if err != nil {
		t.Fatalf("Builder errored: %v", err)
	}

	buff := new(bytes.Buffer)
	if err := bndl.MarshalCbor(buff); err != nil {
		t.Fatal(err)
	}
	bndlCbor := buff.Bytes()

	bndl2 := Bundle{}
	if err = bndl2.UnmarshalCbor(buff); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(bndl, bndl2) {
		t.Fatalf("Bundle changed after serialization: %v, %v", bndl, bndl2)
	}

	bndl3, err := NewBundle(
		NewPrimaryBlock(
			0,
			MustNewEndpointID("dtn://dest/"),
			MustNewEndpointID("dtn://myself/"),
			NewCreationTimestamp(DtnTimeEpoch, 0),
			1000000*60*10),
		[]CanonicalBlock{
			NewCanonicalBlock(2, 0, NewHopCountBlock(64)),
			NewCanonicalBlock(3, 0, NewBundleAgeBlock(0)),
			NewCanonicalBlock(1, 0, NewPayloadBlock([]byte("hello world!")))})
	if err != nil {
		t.Fatal(err)
	}

	buff.Reset()
	bndl3.PrimaryBlock.ReportTo = bndl3.PrimaryBlock.SourceNode
	bndl3.SetCRCType(CRC32)

	if err := bndl3.MarshalCbor(buff); err != nil {
		t.Fatal(err)
	}
	bndl3Cbor := buff.Bytes()

	if !bytes.Equal(bndlCbor, bndl3Cbor) {
		t.Fatalf("CBOR has changed:\n%x\n%x", bndlCbor, bndl3Cbor)
	}

	if !reflect.DeepEqual(bndl, bndl3) {
		t.Fatalf("Bundles differ: %v, %v", bndl, bndl3)
	}
}

func TestBldrParseEndpoint(t *testing.T) {
	eidIn, _ := NewEndpointID("dtn://foo/bar/")
	if eidTmp, _ := bldrParseEndpoint(eidIn); eidTmp != eidIn {
		t.Fatalf("Endpoint does not match: %v != %v", eidTmp, eidIn)
	}

	if eidTmp, _ := bldrParseEndpoint("dtn://foo/bar/"); eidTmp != eidIn {
		t.Fatalf("Parsed endpoint does not match: %v != %v", eidTmp, eidIn)
	}

	if _, errTmp := bldrParseEndpoint(23.42); errTmp == nil {
		t.Fatalf("Invalid endpoint type does not resulted in an error")
	}
}

func TestBldrParseLifetime(t *testing.T) {
	tests := []struct {
		val interface{}
		us  uint64
		err bool
	}{
		{1000, 1000, false},
		{uint64(1000), 1000, false},
		{"1000µs", 1000, false},
		{"1000us", 1000, false},
		{-23, 0, true},
		{"-10m", 0, true},
		{true, 0, true},
	}

	for _, test := range tests {
		us, err := bldrParseLifetime(test.val)

		if test.err == (err == nil) {
			t.Fatalf("Error value for %v was unexpected: %v != %v",
				test.val, test.err, err)
		}

		if test.us != us {
			t.Fatalf("Value for %v was unexpected: %v != %v", test.val, test.us, us)
		}
	}
}

// This function tests the serialization and deserialization of Bundles to CBOR
// et vice versa by comparing this implementation against uPCN's Python
// implementation. Thanks for this code!
//
// uPCN: https://upcn.eu/
// modified implementation dtn-bpbis-13: https://github.com/geistesk/upcn-bundle7
func TestBundleBuilderUpcn(t *testing.T) {
	var upcnBytes = []byte{
		0x9f, 0x89, 0x07, 0x18, 0x84, 0x01, 0x82, 0x01, 0x63, 0x47, 0x53, 0x32,
		0x82, 0x01, 0x00, 0x82, 0x01, 0x00, 0x82, 0x00, 0x00, 0x1a, 0x00, 0x01,
		0x51, 0x80, 0x42, 0x45, 0x39, 0x86, 0x07, 0x02, 0x00, 0x02, 0x82, 0x01,
		0x63, 0x47, 0x53, 0x34, 0x44, 0x96, 0xcf, 0xec, 0xe0, 0x86, 0x09, 0x03,
		0x00, 0x02, 0x82, 0x18, 0x1e, 0x00, 0x44, 0x9f, 0x46, 0x74, 0xc7, 0x86,
		0x08, 0x04, 0x00, 0x02, 0x00, 0x44, 0xaf, 0x9b, 0xbf, 0x74, 0x86, 0x01,
		0x01, 0x00, 0x02, 0x4c, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f,
		0x72, 0x6c, 0x64, 0x21, 0x44, 0xce, 0xa4, 0xb8, 0xbf, 0xff}

	bndl, bndlErr := Builder().
		BundleCtrlFlags(MustNotFragmented | ContainsManifest).
		CRC(CRC32).
		Destination("dtn:GS2").
		Source("dtn:none").
		ReportTo("dtn:none").
		CreationTimestampEpoch().
		Lifetime(24 * 60 * 60).
		PreviousNodeBlock("dtn:GS4").
		HopCountBlock(30).
		BundleAgeBlock(0).
		PayloadBlock([]byte("Hello world!")).
		Build()
	if bndlErr != nil {
		t.Fatal(bndlErr)
	}

	// Different CRC types within a Bundle Builder is not a desired feature
	bndl.PrimaryBlock.SetCRCType(CRC16)

	buff := new(bytes.Buffer)
	if err := cboring.Marshal(&bndl, buff); err != nil {
		t.Fatal(err)
	}

	if bbBytes := buff.Bytes(); !bytes.Equal(upcnBytes, bbBytes) {
		t.Logf("%x", upcnBytes)
		t.Logf("%x", bbBytes)
		t.Fatal("Bundle Builder serialization mismatches")
	}
}
