package bundle

import (
	"bytes"
	"reflect"
	"testing"
)

func TestNewDtnEndpoint(t *testing.T) {
	tests := []struct {
		uri   string
		ssp   string
		valid bool
	}{
		{"dtn:none", dtnEndpointDtnNoneSsp, true},
		{"dtn://foo/", "//foo/", true},
		{"dtn://foo/bar", "//foo/bar", true},
		{"dtn://foo/bar/buz", "//foo/bar/buz", true},
		{"dtn:foo", "foo", false},     // missing slashes
		{"dtn:/foo/", "/foo/", false}, // only one leading slash
		{"dtn://foo", "//foo", false}, // missing trailing slash
		{"dtn:", "", false},           // missing SSP
		{"dtn", "", false},            // missing SSP and ":"
		{"uff:uff", "uff", false},     // just no
		{"", "", false},               // nothing
	}

	for _, test := range tests {
		ep, err := NewDtnEndpoint(test.uri)

		if err == nil != test.valid {
			t.Fatalf("%s: expected valid = %t, got err: %v", test.uri, test.valid, err)
		} else if err == nil {
			if ep.(DtnEndpoint).Ssp != test.ssp {
				t.Fatalf("Expected SSP %v, got %v", test.ssp, ep.(DtnEndpoint).Ssp)
			}
		}
	}
}

func TestDtnEndpointCbor(t *testing.T) {
	tests := []struct {
		ep   DtnEndpoint
		data []byte
	}{
		{DtnEndpoint{dtnEndpointDtnNoneSsp}, []byte{0x00}},
		{DtnEndpoint{"foo"}, []byte{0x63, 0x66, 0x6F, 0x6F}},
		{DtnEndpoint{"//foo/"}, []byte{0x66, 0x2F, 0x2F, 0x66, 0x6F, 0x6F, 0x2F}},
	}

	for _, test := range tests {
		var buf bytes.Buffer

		// Marshal
		if err := test.ep.MarshalCbor(&buf); err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(buf.Bytes(), test.data) {
			t.Fatalf("Expected %v, got %v", test.data, buf.Bytes())
		}

		// Unmarshal
		var ep DtnEndpoint
		if err := ep.UnmarshalCbor(&buf); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(ep, test.ep) {
			t.Fatalf("Expected %v, got %v", test.ep, ep)
		}
	}
}

func TestDtnEndpointUri(t *testing.T) {
	tests := []struct {
		ep        DtnEndpoint
		authority string
		path      string
	}{
		{DtnEndpoint{dtnEndpointDtnNoneSsp}, "none", "/"},
		{DtnEndpoint{"//foobar/"}, "foobar", "/"},
		{DtnEndpoint{"//foo/bar"}, "foo", "/bar"},
		{DtnEndpoint{"//foo/bar/"}, "foo", "/bar/"},
	}

	for _, test := range tests {
		if authority := test.ep.Authority(); test.authority != authority {
			t.Fatalf("Authority: expected %s, got %s", test.authority, authority)
		}
		if path := test.ep.Path(); test.path != path {
			t.Fatalf("Path: expected %s, got %s", test.path, path)
		}
	}
}
