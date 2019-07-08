package bundle

import (
	"fmt"
	"io"
	"strings"

	"github.com/dtn7/cboring"
)

// BundleID identifies a bundle by its source node, creation timestamp and
// fragmentation offset paired the total data length. The last two fields are
// only available if and only if the referenced bundle is a fragment.
//
// Furthermore, a BundleID can be serialized and deserialized with the cboring
// library. Therefore, all required fields will be written in series. For
// deserialization, the IsFragment field MUST be set beforehand. This will
// determine if two or four values will be read.
type BundleID struct {
	SourceNode EndpointID
	Timestamp  CreationTimestamp

	IsFragment      bool
	FragmentOffset  uint64
	TotalDataLength uint64
}

func (bid BundleID) String() string {
	var bldr strings.Builder

	fmt.Fprintf(&bldr, "%v-%d-%d",
		bid.SourceNode, bid.Timestamp[0], bid.Timestamp[1])

	if bid.IsFragment {
		fmt.Fprintf(&bldr, "-%d-%d", bid.FragmentOffset, bid.TotalDataLength)
	}

	return bldr.String()
}

// Len returns the amount of fields, dependent on the fragmentation.
func (bid BundleID) Len() uint64 {
	if bid.IsFragment {
		return 4
	} else {
		return 2
	}
}

func (bid *BundleID) MarshalCbor(w io.Writer) error {
	if err := cboring.Marshal(&bid.SourceNode, w); err != nil {
		return fmt.Errorf("Marshalling source node failed: %v", err)
	}

	if err := cboring.Marshal(&bid.Timestamp, w); err != nil {
		return fmt.Errorf("Marshalling timestamp failed: %v", err)
	}

	if bid.IsFragment {
		flds := []uint64{bid.FragmentOffset, bid.TotalDataLength}
		for _, fld := range flds {
			if err := cboring.WriteUInt(fld, w); err != nil {
				return err
			}
		}
	}

	return nil
}

func (bid *BundleID) UnmarshalCbor(r io.Reader) error {
	if err := cboring.Unmarshal(&bid.SourceNode, r); err != nil {
		return fmt.Errorf("Unmarshalling source node failed: %v", err)
	}

	if err := cboring.Unmarshal(&bid.Timestamp, r); err != nil {
		return fmt.Errorf("Unmarshalling timestamp failed: %v", err)
	}

	// MUST be set beforehand
	if bid.IsFragment {
		flds := []*uint64{&bid.FragmentOffset, &bid.TotalDataLength}
		for _, fld := range flds {
			if n, err := cboring.ReadUInt(r); err != nil {
				return err
			} else {
				*fld = n
			}
		}
	}

	return nil
}
