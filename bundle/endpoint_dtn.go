package bundle

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/dtn7/cboring"
)

const (
	dtnEndpointSchemeName string = "dtn"
	dtnEndpointSchemeNo   uint64 = 1
	dtnEndpointDtnNoneSsp string = "none"
)

// DtnEndpoint describes the dtn URI for EndpointIDs, as defined in ietf-dtn-bpbis.
type DtnEndpoint struct {
	Ssp string
}

// NewDtnEndpoint from an URI with the dtn scheme.
func NewDtnEndpoint(uri string) (e EndpointType, err error) {
	re := regexp.MustCompile("^" + dtnEndpointSchemeName + ":(.+)$")
	if !re.MatchString(uri) {
		err = fmt.Errorf("uri does not match a dtn endpoint")
		return
	}

	e = DtnEndpoint{Ssp: re.FindStringSubmatch(uri)[1]}
	return
}

// SchemeName is "dtn" for DtnEndpoints.
func (_ DtnEndpoint) SchemeName() string {
	return dtnEndpointSchemeName
}

// SchemeNo is 1 for DtnEndpoints.
func (_ DtnEndpoint) SchemeNo() uint64 {
	return dtnEndpointSchemeNo
}

func (e DtnEndpoint) parseUri() (authority, path string) {
	// net.url.URL requires two leading slashes.
	var tmpEndpoint string
	if !strings.HasPrefix(e.Ssp, "//") {
		tmpEndpoint = DtnEndpoint{"//" + e.Ssp}.String()
	} else {
		tmpEndpoint = e.String()
	}

	u, err := url.Parse(tmpEndpoint)
	if err != nil {
		return
	}

	authority = u.Hostname()
	path = u.RequestURI()
	return
}

// Authority is the authority part of the Endpoint URI, e.g., "foo" for "dtn://foo/bar".
func (e DtnEndpoint) Authority() string {
	authority, _ := e.parseUri()
	return authority
}

// Path is the path part of the Endpoint URI, e.g., "/bar" for "dtn://foo/bar".
func (e DtnEndpoint) Path() string {
	_, path := e.parseUri()
	return path
}

// CheckValid returns an array of errors for incorrect data.
func (_ DtnEndpoint) CheckValid() error {
	return nil
}

func (e DtnEndpoint) String() string {
	return fmt.Sprintf("%s:%s", dtnEndpointSchemeName, e.Ssp)
}

// MarshalCbor writes this DtnEndpoint's CBOR representation.
func (e DtnEndpoint) MarshalCbor(w io.Writer) error {
	var isDtnNone = e.Ssp == dtnEndpointDtnNoneSsp
	if isDtnNone {
		return cboring.WriteUInt(0, w)
	} else {
		return cboring.WriteTextString(e.Ssp, w)
	}
}

// UnmarshalCbor reads a CBOR representation.
func (e *DtnEndpoint) UnmarshalCbor(r io.Reader) error {
	if m, n, err := cboring.ReadMajors(r); err != nil {
		return err
	} else {
		switch m {
		case cboring.UInt:
			// dtn:none
			e.Ssp = dtnEndpointDtnNoneSsp

		case cboring.TextString:
			// dtn:whatever
			if tmp, err := cboring.ReadRawBytes(n, r); err != nil {
				return err
			} else {
				e.Ssp = string(tmp)
			}

		default:
			return fmt.Errorf("DtnEndpoint: wrong major type 0x%X for unmarshalling", m)
		}
	}

	return nil
}

// DtnNone returns the null endpoint "dtn:none".
func DtnNone() EndpointID {
	return EndpointID{DtnEndpoint{Ssp: dtnEndpointDtnNoneSsp}}
}
