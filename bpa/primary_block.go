package bpa

import (
	"fmt"
	"strings"
)

const DTNVersion uint = 7

// PrimaryBlock is a representation of a Primary Bundle Block as defined in
// section 4.2.2.
type PrimaryBlock struct {
	Version            uint
	BundleControlFlags BundleControlFlags
	CRCType            CRCType
	Destination        EndpointID
	SourceNode         EndpointID
	ReportTo           EndpointID
	CreationTimestamp  CreationTimestamp
	Lifetime           uint
	FragmentOffset     uint
	TotalDataLength    uint
	CRC                uint
}

// NewPrimaryBlock creates a new PrimaryBlock with the given parameters. All
// other fields are set to default values.
func NewPrimaryBlock(bundleControlFlags BundleControlFlags,
	destination EndpointID, sourceNode EndpointID,
	creationTimestamp CreationTimestamp, lifetime uint) PrimaryBlock {
	return PrimaryBlock{
		Version:            DTNVersion,
		BundleControlFlags: bundleControlFlags,
		CRCType:            CRCNo,
		Destination:        destination,
		SourceNode:         sourceNode,
		ReportTo:           *DtnNone,
		CreationTimestamp:  creationTimestamp,
		Lifetime:           lifetime,
		FragmentOffset:     0,
		TotalDataLength:    0,
		CRC:                0,
	}
}

// HasFragmentation returns if the Bundle Processing Control Flags indicates a
// fragmented bundle. In this case the FragmentOffset and TotalDataLength fields
// of this struct should become relevant.
func (pb PrimaryBlock) HasFragmentation() bool {
	return pb.BundleControlFlags.Has(BndlCFBundleIsAFragment)
}

// HasCRC retruns if the CRCType indicates a CRC present for this bundle. In
// this case the CRC field of this struct should become relevant.
func (pb PrimaryBlock) HasCRC() bool {
	return pb.CRCType != CRCNo
}

func (pb PrimaryBlock) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "version: %d, ", pb.Version)
	fmt.Fprintf(&b, "bundle processing control flags: %b, ", pb.BundleControlFlags)
	fmt.Fprintf(&b, "crc type: %v, ", pb.CRCType)
	fmt.Fprintf(&b, "destination: %v, ", pb.Destination)
	fmt.Fprintf(&b, "source node: %v, ", pb.SourceNode)
	fmt.Fprintf(&b, "report to: %v, ", pb.ReportTo)
	fmt.Fprintf(&b, "creation timestamp: %v, ", pb.CreationTimestamp)
	fmt.Fprintf(&b, "lifetime: %d", pb.Lifetime)

	if pb.HasFragmentation() {
		fmt.Fprintf(&b, " , ")
		fmt.Fprintf(&b, "fragment offset: %d, ", pb.FragmentOffset)
		fmt.Fprintf(&b, "total data length: %d", pb.TotalDataLength)
	}

	if pb.HasCRC() {
		fmt.Fprintf(&b, " , ")
		fmt.Fprintf(&b, "crc: %x", pb.CRC)
	}

	return b.String()
}
