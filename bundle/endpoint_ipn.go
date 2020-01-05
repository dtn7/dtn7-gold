package bundle

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/dtn7/cboring"
)

const (
	ipnEndpointSchemeName string = "ipn"
	ipnEndpointSchemeNo   uint64 = 2
)

// IpnEndpoint describes the ipn URI for EndpointIDs, as defined in RFC 6260.
type IpnEndpoint struct {
	Node    uint64
	service uint64
}

// NewIpnEndpoint from an URI with the ipn scheme.
func NewIpnEndpoint(uri string) (e EndpointType, err error) {
	// As defined in RFC 6260, section 2.1:
	// - node number: ASCII numeric digits between 1 and (2^64-1)
	// - an ASCII dot
	// - service number: ASCII numeric digits between 1 and (2^64-1)

	re := regexp.MustCompile("^" + ipnEndpointSchemeName + ":(\\d+)\\.(\\d+)$")
	matches := re.FindStringSubmatch(uri)
	if len(matches) != 3 {
		err = fmt.Errorf("uri does not match an ipn endpoint")
		return
	}

	var node, service uint64
	if node, err = strconv.ParseUint(matches[1], 10, 64); err != nil {
		return
	}
	if service, err = strconv.ParseUint(matches[2], 10, 64); err != nil {
		return
	}

	e = IpnEndpoint{node, service}
	err = e.CheckValid()

	return
}

func (e IpnEndpoint) SchemeName() string {
	return ipnEndpointSchemeName
}

func (e IpnEndpoint) SchemeNo() uint64 {
	return ipnEndpointSchemeNo
}

func (e IpnEndpoint) CheckValid() error {
	if e.Node < 1 || e.service < 1 {
		return fmt.Errorf("ipn's node and service number must be >= 1")
	}

	return nil
}

func (e IpnEndpoint) String() string {
	return fmt.Sprintf("%s:%d.%d", ipnEndpointSchemeName, e.Node, e.service)
}

func (e IpnEndpoint) MarshalCbor(w io.Writer) error {
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	for _, n := range []uint64{e.Node, e.service} {
		if err := cboring.WriteUInt(n, w); err != nil {
			return err
		}
	}

	return nil
}

func (e *IpnEndpoint) UnmarshalCbor(r io.Reader) error {
	if n, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if n != 2 {
		return fmt.Errorf("ipn uri expected array of 2 elements, not %d", n)
	}

	for _, n := range []*uint64{&e.Node, &e.service} {
		if i, err := cboring.ReadUInt(r); err != nil {
			return err
		} else {
			*n = i
		}
	}

	return nil
}
