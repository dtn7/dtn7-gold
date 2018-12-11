package bpa

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	URISchemeDTN uint = 1
	URISchemeIPN uint = 2
)

// DtnNone is a instance of the `dtn:none` endpoint id.
var DtnNone, _ = NewEndpointID("dtn", "none")

// EndpointID represents an Endpoint ID as defined in section 4.1.5.1. The
// "scheme name" is represented by an uint (vide supra) and the "scheme-specific
// part" (SSP) by an interface{}. Based on the characteristic of the name, the
// underlying type of the SSP may vary.
type EndpointID struct {
	SchemeName         uint
	SchemeSpecificPort interface{}
}

func newEndpointIDDTN(ssp string) (*EndpointID, error) {
	var sspRaw interface{}
	if ssp == "none" {
		sspRaw = uint(0)
	} else {
		sspRaw = string(ssp)
	}

	return &EndpointID{
		SchemeName:         URISchemeDTN,
		SchemeSpecificPort: sspRaw,
	}, nil
}

func newEndpointIDIPN(ssp string) (*EndpointID, error) {
	// As definied in RFC 6260, section 2.1:
	// - node number: ASCII numeric digits between 1 and (2^64-1)
	// - an ASCII dot
	// - service number: ASCII numeric digits between 1 and (2^64-1)

	re := regexp.MustCompile(`^(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(ssp)
	if len(matches) != 3 {
		return nil, NewBPAError("IPN does not satisfy given regex")
	}

	nodeNo, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return nil, err
	}

	serviceNo, err := strconv.ParseUint(matches[2], 10, 64)
	if err != nil {
		return nil, err
	}

	if nodeNo < 1 || serviceNo < 1 {
		return nil, NewBPAError("IPN's node and service number must be >= 1")
	}

	return &EndpointID{
		SchemeName:         URISchemeIPN,
		SchemeSpecificPort: [2]uint64{nodeNo, serviceNo},
	}, nil
}

// NewEndpointID creates a new EndpointID by a given "scheme name" and a
// "scheme-specific part" (SSP). Currently the "dtn" and "ipn"-scheme names
// are supported.
func NewEndpointID(name, ssp string) (*EndpointID, error) {
	switch name {
	case "dtn":
		return newEndpointIDDTN(ssp)
	case "ipn":
		return newEndpointIDIPN(ssp)
	default:
		return nil, NewBPAError("Unknown scheme type")
	}
}

func (eid EndpointID) String() string {
	var b strings.Builder

	switch eid.SchemeName {
	case URISchemeDTN:
		b.WriteString("dtn")
	case URISchemeIPN:
		b.WriteString("ipn")
	default:
		fmt.Fprintf(&b, "unknown_%d", eid.SchemeName)
	}
	b.WriteRune(':')

	switch t := eid.SchemeSpecificPort.(type) {
	case uint:
		if eid.SchemeName == URISchemeDTN && eid.SchemeSpecificPort.(uint) == 0 {
			b.WriteString("none")
		} else {
			fmt.Fprintf(&b, "%d", eid.SchemeSpecificPort.(uint))
		}

	case string:
		b.WriteString(eid.SchemeSpecificPort.(string))

	case [2]uint64:
		var ssp [2]uint64 = eid.SchemeSpecificPort.([2]uint64)
		if eid.SchemeName == URISchemeIPN {
			fmt.Fprintf(&b, "%d.%d", ssp[0], ssp[1])
		} else {
			fmt.Fprintf(&b, "%v", ssp)
		}

	default:
		fmt.Fprintf(&b, "unkown %T: %v", t, eid.SchemeSpecificPort)
	}

	return b.String()
}
