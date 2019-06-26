package bundle

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/dtn7/cboring"
)

const (
	endpointURISchemeDTN uint64 = 1
	endpointURISchemeIPN uint64 = 2
)

// EndpointID represents an Endpoint ID as defined in section 4.1.5.1. The
// "scheme name" is represented by an uint and the "scheme-specific part"
// (SSP) by an interface{}. Based on the characteristic of the name, the
// underlying type of the SSP may vary.
type EndpointID struct {
	SchemeName         uint64
	SchemeSpecificPart interface{}
}

func newEndpointIDDTN(ssp string) (EndpointID, error) {
	var sspRaw interface{}
	if ssp == "none" {
		sspRaw = uint64(0)
	} else {
		sspRaw = string(ssp)
	}

	return EndpointID{
		SchemeName:         endpointURISchemeDTN,
		SchemeSpecificPart: sspRaw,
	}, nil
}

func newEndpointIDIPN(ssp string) (ep EndpointID, err error) {
	// As definied in RFC 6260, section 2.1:
	// - node number: ASCII numeric digits between 1 and (2^64-1)
	// - an ASCII dot
	// - service number: ASCII numeric digits between 1 and (2^64-1)

	re := regexp.MustCompile(`^(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(ssp)
	if len(matches) != 3 {
		err = newBundleError("IPN does not satisfy given regex")
		return
	}

	nodeNo, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return
	}

	serviceNo, err := strconv.ParseUint(matches[2], 10, 64)
	if err != nil {
		return
	}

	if nodeNo < 1 || serviceNo < 1 {
		err = newBundleError("IPN's node and service number must be >= 1")
		return
	}

	ep = EndpointID{
		SchemeName:         endpointURISchemeIPN,
		SchemeSpecificPart: [2]uint64{nodeNo, serviceNo},
	}
	return
}

// NewEndpointID creates a new EndpointID by a given "scheme name" and a
// "scheme-specific part" (SSP). Currently the "dtn" and "ipn"-scheme names
// are supported.
//
// Example: "dtn:foobar"
func NewEndpointID(eid string) (e EndpointID, err error) {
	re := regexp.MustCompile(`^([[:alnum:]]+):(.+)$`)
	matches := re.FindStringSubmatch(eid)

	if len(matches) != 3 {
		err = newBundleError("eid does not satisfy regex")
		return
	}

	name := matches[1]
	ssp := matches[2]

	switch name {
	case "dtn":
		return newEndpointIDDTN(ssp)
	case "ipn":
		return newEndpointIDIPN(ssp)
	default:
		return EndpointID{}, newBundleError("Unknown scheme type")
	}
}

// MustNewEndpointID returns a new EndpointID like NewEndpointID, but panics
// in case of an error.
func MustNewEndpointID(eid string) EndpointID {
	ep, err := NewEndpointID(eid)
	if err != nil {
		panic(err)
	}

	return ep
}

func (eid *EndpointID) MarshalCbor(w io.Writer) error {
	// Start an array with two elements
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	// URI code: scheme name
	if err := cboring.WriteUInt(eid.SchemeName, w); err != nil {
		return err
	}

	// SSP
	switch eid.SchemeSpecificPart.(type) {
	case uint64:
		// dtn:none
		if err := cboring.WriteUInt(0, w); err != nil {
			return err
		}

	case string:
		// dtn:whatsoever
		if err := cboring.WriteTextString(eid.SchemeSpecificPart.(string), w); err != nil {
			return err
		}

	case [2]uint64:
		// ipn:23.42
		var ssps [2]uint64 = eid.SchemeSpecificPart.([2]uint64)
		if err := cboring.WriteArrayLength(2, w); err != nil {
			return err
		}

		for _, ssp := range ssps {
			if err := cboring.WriteUInt(ssp, w); err != nil {
				return err
			}
		}
	}

	return nil
}

func (eid *EndpointID) UnmarshalCbor(r io.Reader) error {
	// Start of an array with two elements
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 2 {
		return fmt.Errorf("Expected array with length 2, got %d", l)
	}

	// URI code: scheme name
	if sn, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		eid.SchemeName = sn
	}

	// SSP
	if m, n, err := cboring.ReadMajors(r); err != nil {
		return err
	} else {
		switch m {
		case cboring.UInt:
			// dtn:none
			eid.SchemeSpecificPart = n

		case cboring.TextString:
			// dtn:whatsoever
			if tmp, err := cboring.ReadRawBytes(n, r); err != nil {
				return err
			} else {
				eid.SchemeSpecificPart = string(tmp)
			}

		case cboring.Array:
			// ipn:23.42
			if n != 2 {
				return fmt.Errorf("Expected array with length 2, got %d", n)
			}

			var ssps [2]uint64
			for i := 0; i < 2; i++ {
				if n, err := cboring.ReadUInt(r); err != nil {
					return err
				} else {
					ssps[i] = n
				}
			}

			eid.SchemeSpecificPart = ssps
		}
	}

	return nil
}

func (eid EndpointID) checkValidDtn() error {
	switch eid.SchemeSpecificPart.(type) {
	case string:
		if eid.SchemeSpecificPart.(string) == "none" {
			return newBundleError("EndpointID: equals dtn:none, with none as a string")
		}
	}

	return nil
}

func (eid EndpointID) checkValidIpn() error {
	ssp := eid.SchemeSpecificPart.([2]uint64)
	if ssp[0] < 1 || ssp[1] < 1 {
		return newBundleError("EndpointID: IPN's node and service number must be >= 1")
	}

	return nil
}

func (eid EndpointID) checkValid() error {
	switch eid.SchemeName {
	case endpointURISchemeDTN:
		return eid.checkValidDtn()

	case endpointURISchemeIPN:
		return eid.checkValidIpn()

	default:
		return newBundleError("EndpointID: unknown scheme name")
	}
}

func (eid EndpointID) String() string {
	var b strings.Builder

	switch eid.SchemeName {
	case endpointURISchemeDTN:
		b.WriteString("dtn")
	case endpointURISchemeIPN:
		b.WriteString("ipn")
	default:
		fmt.Fprintf(&b, "unknown_%d", eid.SchemeName)
	}
	b.WriteRune(':')

	switch t := eid.SchemeSpecificPart.(type) {
	case uint64:
		if eid.SchemeName == endpointURISchemeDTN && eid.SchemeSpecificPart.(uint64) == 0 {
			b.WriteString("none")
		} else {
			fmt.Fprintf(&b, "%d", eid.SchemeSpecificPart.(uint64))
		}

	case string:
		b.WriteString(eid.SchemeSpecificPart.(string))

	case [2]uint64:
		var ssp [2]uint64 = eid.SchemeSpecificPart.([2]uint64)
		if eid.SchemeName == endpointURISchemeIPN {
			fmt.Fprintf(&b, "%d.%d", ssp[0], ssp[1])
		} else {
			fmt.Fprintf(&b, "%v", ssp)
		}

	default:
		fmt.Fprintf(&b, "unknown %T: %v", t, eid.SchemeSpecificPart)
	}

	return b.String()
}

// DtnNone returns the "null endpoint", "dtn:none".
func DtnNone() EndpointID {
	return EndpointID{
		SchemeName:         endpointURISchemeDTN,
		SchemeSpecificPart: uint64(0),
	}
}
