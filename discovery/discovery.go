// Package discovery contains code for peer/neighbor discovery of other DTN
// nodes through UDP multicast packages.
package discovery

import (
	"bytes"
	"fmt"
	"github.com/dtn7/dtn7-go/core"
	"io"
	"strings"

	"github.com/dtn7/cboring"
	"github.com/dtn7/dtn7-go/bundle"
)

const (
	// DiscoveryAddress4 is the default multicast IPv4 address used for discovery.
	DiscoveryAddress4 = "224.23.23.23"

	// DiscoveryAddress6 is the default multicast IPv4 add6ess used for discovery.
	DiscoveryAddress6 = "ff02::23"

	// DiscoveryPort is the default multicast port used for discovery.
	DiscoveryPort = 35039
)

// DiscoveryMessage is the kind of message used by this peer/neighbor discovery.
type DiscoveryMessage struct {
	Type     core.CLAType
	Endpoint bundle.EndpointID
	Port     uint
}

// NewDiscoveryMessagesFromCbor creates a new array of DiscoveryMessage based
// on the given CBOR byte string.
func NewDiscoveryMessagesFromCbor(data []byte) (dms []DiscoveryMessage, err error) {
	buff := bytes.NewBuffer(data)

	if l, cErr := cboring.ReadArrayLength(buff); cErr != nil {
		err = cErr
		return
	} else {
		dms = make([]DiscoveryMessage, l)
	}

	for i := 0; i < len(dms); i++ {
		if cErr := cboring.Unmarshal(&dms[i], buff); cErr != nil {
			err = fmt.Errorf("Unmarshalling DiscoveryMessage %d failed: %v", i, cErr)
			return
		}
	}

	return
}

// DiscoveryMessagesToCbor returns a CBOR byte string representation of this
// array of DiscoveryMessages.
func DiscoveryMessagesToCbor(dms []DiscoveryMessage) (data []byte, err error) {
	buff := new(bytes.Buffer)

	if cErr := cboring.WriteArrayLength(uint64(len(dms)), buff); cErr != nil {
		err = cErr
		return
	}

	for i, dm := range dms {
		discoveryMessage := dm
		if cErr := cboring.Marshal(&discoveryMessage, buff); cErr != nil {
			err = fmt.Errorf("Marshalling DiscoveryMessage %d failed: %v", i, cErr)
			return
		}
	}

	data = buff.Bytes()
	return
}

func (dm *DiscoveryMessage) MarshalCbor(w io.Writer) error {
	if err := cboring.WriteArrayLength(3, w); err != nil {
		return err
	}

	if err := cboring.WriteUInt(uint64(dm.Type), w); err != nil {
		return err
	}
	if err := cboring.Marshal(&dm.Endpoint, w); err != nil {
		return fmt.Errorf("Marshalling endpoint failed: %v", err)
	}
	if err := cboring.WriteUInt(uint64(dm.Port), w); err != nil {
		return err
	}

	return nil
}

func (dm *DiscoveryMessage) UnmarshalCbor(r io.Reader) error {
	if l, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if l != 3 {
		return fmt.Errorf("Wrong array length: %d instead of 3", l)
	}

	if n, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		dm.Type = core.CLAType(n)
	}
	if err := cboring.Unmarshal(&dm.Endpoint, r); err != nil {
		return fmt.Errorf("Unmarshalling endpoint failed: %v", err)
	}
	if n, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		dm.Port = uint(n)
	}

	return nil
}

func (dm DiscoveryMessage) String() string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "DiscoveryMessage(")

	switch dm.Type {
	case core.TCPCL:
		fmt.Fprintf(&builder, "TCPCL")
	case core.MTCP:
		fmt.Fprintf(&builder, "MTCP")
	default:
		fmt.Fprintf(&builder, "Unknown CLA")
	}

	fmt.Fprintf(&builder, ",%v,%d)", dm.Endpoint, dm.Port)

	return builder.String()
}
