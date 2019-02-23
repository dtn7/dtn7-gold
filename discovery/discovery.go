package discovery

import (
	"fmt"
	"net"
	"strings"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

const (
	// DiscoveryAddress4 is the default multicast IPv4 address used for discovery.
	DiscoveryAddress4 = "224.23.23.23"

	// DiscoveryAddress6 is the default multicast IPv4 add6ess used for discovery.
	DiscoveryAddress6 = "ff00::23:23:23"

	// DiscoveryPort is the default multicast port used for discovery.
	DiscoveryPort = 35039
)

// CLAType is the first field of a DiscoveryMessage, specifying a CLA.
type CLAType uint

const (
	// TCPCLV4 is the "Delay-Tolerant Networking TCP Convergence Layer Protocol
	// Version 4" as specified in draft-ietf-dtn-tcpclv4-10 or newer documents.
	TCPCLV4 CLAType = 0

	// STCP is the "Simple TCP Convergence-Layer Protocol" as specified in
	// draft-burleigh-dtn-stcp-00 or newer documents.
	STCP CLAType = 1
)

// DiscoveryMessage is the kind of message used by this peer/neighbor discovery.
type DiscoveryMessage struct {
	_struct struct{} `codec:",toarray"`

	Type        CLAType
	Endpoint    bundle.EndpointID
	Address     net.IP
	Port        uint
	Additionals []byte
}

// NewDiscoveryMessageFromCbor creates a new DiscoveryMessage based on the given
// CBOR byte string.
func NewDiscoveryMessageFromCbor(buff []byte) (dm DiscoveryMessage, err error) {
	var dec = codec.NewDecoderBytes(buff, new(codec.CborHandle))
	err = dec.Decode(&dm)

	return
}

// Cbor returns a CBOR byte string representation of this DiscoveryMessage.
func (dm DiscoveryMessage) Cbor() (buff []byte, err error) {
	var enc = codec.NewEncoderBytes(&buff, new(codec.CborHandle))
	err = enc.Encode(dm)

	return
}

func (dm DiscoveryMessage) String() string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "DiscoveryMessage(")

	switch dm.Type {
	case TCPCLV4:
		fmt.Fprintf(&builder, "TCPCLv4")
	case STCP:
		fmt.Fprintf(&builder, "STCP")
	default:
		fmt.Fprintf(&builder, "Unknown CLA")
	}

	fmt.Fprintf(&builder, ",%v,%v,%d,%v)",
		dm.Endpoint, dm.Address, dm.Port, dm.Additionals)

	return builder.String()
}
