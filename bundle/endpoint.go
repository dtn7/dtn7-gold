// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bundle

import (
	"encoding/gob"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sync"

	"github.com/dtn7/cboring"
)

// EndpointType describes a discrete EndpointID.
// Because of Go's type system, the MarshalCbor function from the cboring library must be implemented as a
// value receiver in this interface. In addition, the UnmarshalCbor function MUST be implemented as a pointer
// receiver. Afaik, this is not possible to describe with a Golang interface..
type EndpointType interface {
	// SchemeName must return the static URI scheme type for this endpoint, e.g., "dtn" or "ipn".
	SchemeName() string

	// SchemeNo must return the static URI scheme type number for this endpoint, e.g., 1 for "dtn".
	SchemeNo() uint64

	// Authority is the authority part of the Endpoint URI, e.g., "foo" for "dtn://foo/bar".
	Authority() string

	// Path is the path part of the Endpoint URI, e.g., "/bar" for "dtn://foo/bar".
	Path() string

	// IsSingleton checks if this Endpoint represents a singleton.
	IsSingleton() bool

	// MarshalCbor is the marshalling CBOR function from the cboring library.
	MarshalCbor(io.Writer) error

	Valid
	fmt.Stringer
}

type endpointManager struct {
	typeMap map[uint64]reflect.Type
	newMap  map[string]func(string) (EndpointType, error)
}

var (
	endpointMngr  *endpointManager
	endpointMutex sync.Mutex
)

func getEndpointManager() *endpointManager {
	endpointMutex.Lock()
	defer endpointMutex.Unlock()

	if endpointMngr == nil {
		endpointMngr = &endpointManager{
			typeMap: make(map[uint64]reflect.Type),
			newMap:  make(map[string]func(string) (EndpointType, error)),
		}

		epTypes := []struct {
			schemeNo   uint64
			schemeName string
			impl       interface{}
			newFunc    func(string) (EndpointType, error)
		}{
			{dtnEndpointSchemeNo, dtnEndpointSchemeName, DtnEndpoint{}, NewDtnEndpoint},
			{ipnEndpointSchemeNo, ipnEndpointSchemeName, IpnEndpoint{}, NewIpnEndpoint},
		}

		for _, epType := range epTypes {
			endpointMngr.typeMap[epType.schemeNo] = reflect.TypeOf(epType.impl)
			endpointMngr.newMap[epType.schemeName] = epType.newFunc
			gob.Register(epType.impl)
		}
	}

	return endpointMngr
}

// EndpointID represents an Endpoint ID as defined in section 4.1.5.1.
// Its form is specified in an EndpointType, e.g., DtnEndpoint.
type EndpointID struct {
	EndpointType EndpointType
}

// NewEndpointID based on an URI, e.g., "dtn://seven/".
func NewEndpointID(uri string) (e EndpointID, err error) {
	re := regexp.MustCompile("^([[:alnum:]]+):.+$")
	matches := re.FindStringSubmatch(uri)

	if len(matches) == 0 {
		err = fmt.Errorf("given URI does not match URI regexp")
		return
	}

	scheme := matches[1]
	if f, ok := getEndpointManager().newMap[scheme]; !ok {
		err = fmt.Errorf("no handler registered for URI scheme %s", scheme)
	} else if et, etErr := f(uri); etErr != nil {
		err = etErr
	} else {
		e = EndpointID{et}
	}
	return
}

// MustNewEndpointID based on an URI like NewEndpointID, but panics on an error.
func MustNewEndpointID(uri string) EndpointID {
	if ep, err := NewEndpointID(uri); err != nil {
		panic(err)
	} else {
		return ep
	}
}

// MarshalCbor writes the CBOR representation of this Endpoint ID.
func (eid *EndpointID) MarshalCbor(w io.Writer) error {
	if err := cboring.WriteArrayLength(2, w); err != nil {
		return err
	}

	// URI scheme name code
	if err := cboring.WriteUInt(eid.EndpointType.SchemeNo(), w); err != nil {
		return err
	}

	// SSP
	if err := eid.EndpointType.MarshalCbor(w); err != nil {
		return err
	}

	return nil
}

// UnmarshalCbor creates this Endpoint ID based on a CBOR representation.
func (eid *EndpointID) UnmarshalCbor(r io.Reader) error {
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 2 {
		return fmt.Errorf("EndpointID expects array of 2 elements, not %d", l)
	}

	var epType reflect.Type

	// URI scheme name code
	if scheme, err := cboring.ReadUInt(r); err != nil {
		return err
	} else if ept, ok := getEndpointManager().typeMap[scheme]; !ok {
		return fmt.Errorf("no URI scheme registered for scheme number %d", scheme)
	} else {
		epType = ept
	}

	// SSP
	tmpEt := reflect.New(epType)
	tmpEtUnmarshalCbor := tmpEt.MethodByName("UnmarshalCbor")
	if err := tmpEtUnmarshalCbor.Call([]reflect.Value{reflect.ValueOf(r)})[0].Interface(); err != nil {
		return err.(error)
	} else {
		eid.EndpointType = tmpEt.Elem().Interface().(EndpointType)
	}

	return nil
}

// Authority is the authority part of the Endpoint URI, e.g., "foo" for "dtn://foo/bar".
func (eid EndpointID) Authority() string {
	return eid.EndpointType.Authority()
}

// Path is the path part of the Endpoint URI, e.g., "/bar" for "dtn://foo/bar".
func (eid EndpointID) Path() string {
	return eid.EndpointType.Path()
}

// IsSingleton checks if this Endpoint represents a singleton.
func (eid EndpointID) IsSingleton() bool {
	return eid.EndpointType.IsSingleton()
}

// SameNode checks if two Endpoints contain to the same Node, based on the scheme and authority part.
func (eid EndpointID) SameNode(other EndpointID) bool {
	return eid.EndpointType.SchemeName() == other.EndpointType.SchemeName() &&
		eid.EndpointType.Authority() == other.EndpointType.Authority()
}

// CheckValid returns an array of errors for incorrect data.
func (eid EndpointID) CheckValid() error {
	return eid.EndpointType.CheckValid()
}

func (eid EndpointID) String() string {
	if eid.EndpointType == nil {
		return DtnNone().String()
	} else {
		return eid.EndpointType.String()
	}
}
