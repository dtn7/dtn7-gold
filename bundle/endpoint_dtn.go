package bundle

import (
	"fmt"
	"io"
	"net/url"
	"regexp"

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
	// As defined in dtn-bpbis, a "dtn" URI might be the null endpoint "dtn:none" or something URI/IRI like.
	// Thus, at first we are going after the null endpoint and inspect a more generic URI afterwards.

	if uri == DtnNone().String() {
		return DtnNone().EndpointType, nil
	}

	re := regexp.MustCompile("^" + dtnEndpointSchemeName + "://(.+)/(.*)$")
	if !re.MatchString(uri) {
		err = fmt.Errorf("uri does not match a dtn endpoint")
		return
	}

	switch submatches := re.FindStringSubmatch(uri); len(submatches) {
	case 2:
		e = DtnEndpoint{Ssp: fmt.Sprintf("//%s", submatches[1])}

	case 3:
		e = DtnEndpoint{Ssp: fmt.Sprintf("//%s/%s", submatches[1], submatches[2])}

	default:
		err = fmt.Errorf("invalid amount of submatches: %d", len(submatches))
		return
	}

	err = e.CheckValid()
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
	// The null endpoint requires some specific behaviour because it does not comply with the URI schema.
	if e.Ssp == dtnEndpointDtnNoneSsp {
		return "none", "/"
	}

	u, err := url.Parse(e.String())
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
func (e DtnEndpoint) CheckValid() (err error) {
	re := regexp.MustCompile("^" + dtnEndpointSchemeName + ":(none|//(.+)/(.*))$")
	if !re.MatchString(e.String()) {
		err = fmt.Errorf("dtn URI does not match regexp")
	}
	return
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
			// dtn://whatever/
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
