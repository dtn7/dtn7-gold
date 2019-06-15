package bundle

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/ugorji/go/codec"
)

const dtnVersion uint64 = 7

// PrimaryBlock is a representation of the primary bundle block as defined in
// section 4.2.2.
type PrimaryBlock struct {
	Version            uint64
	BundleControlFlags BundleControlFlags
	CRCType            CRCType
	Destination        EndpointID
	SourceNode         EndpointID
	ReportTo           EndpointID
	CreationTimestamp  CreationTimestamp
	Lifetime           uint64
	FragmentOffset     uint64
	TotalDataLength    uint64
	CRC                []byte
}

// NewPrimaryBlock creates a new primary block with the given parameters. All
// other fields are set to default values. The lifetime is taken in
// microseconds.
func NewPrimaryBlock(bundleControlFlags BundleControlFlags,
	destination EndpointID, sourceNode EndpointID,
	creationTimestamp CreationTimestamp, lifetime uint64) PrimaryBlock {
	return PrimaryBlock{
		Version:            dtnVersion,
		BundleControlFlags: bundleControlFlags,
		CRCType:            CRCNo,
		Destination:        destination,
		SourceNode:         sourceNode,
		ReportTo:           sourceNode,
		CreationTimestamp:  creationTimestamp,
		Lifetime:           lifetime,
		FragmentOffset:     0,
		TotalDataLength:    0,
		CRC:                nil,
	}
}

// HasFragmentation returns true if the bundle processing control flags
// indicates a fragmented bundle. In this case the FragmentOffset and
// TotalDataLength fields should become relevant.
func (pb PrimaryBlock) HasFragmentation() bool {
	return pb.BundleControlFlags.Has(IsFragment)
}

// HasCRC retruns true if the CRCType indicates a CRC present for this block.
// In this case the CRC value should become relevant.
func (pb PrimaryBlock) HasCRC() bool {
	return pb.GetCRCType() != CRCNo
}

// GetCRCType returns the CRCType of this block.
func (pb PrimaryBlock) GetCRCType() CRCType {
	return pb.CRCType
}

// getCRC retruns the CRC value.
func (pb PrimaryBlock) getCRC() []byte {
	return pb.CRC
}

// SetCRCType sets the CRC type.
func (pb *PrimaryBlock) SetCRCType(crcType CRCType) {
	pb.CRCType = crcType
}

// CalculateCRC calculates and writes the CRC-value for this block.
func (pb *PrimaryBlock) CalculateCRC() {
	pb.setCRC(calculateCRC(pb))
}

// CheckCRC returns true if the CRC value matches to its CRCType or the
// CRCType is CRCNo.
//
// This method changes the block's CRC value temporary and is not thread safe.
func (pb *PrimaryBlock) CheckCRC() bool {
	return checkCRC(pb)
}

// resetCRC resets the CRC value to zero. This should be called before
// calculating the CRC value of this Block.
func (pb *PrimaryBlock) resetCRC() {
	pb.CRC = emptyCRC(pb.GetCRCType())
}

// setCRC sets the CRC value to the given value.
func (pb *PrimaryBlock) setCRC(crc []byte) {
	pb.CRC = crc
}

func (pb *PrimaryBlock) CodecEncodeSelf(enc *codec.Encoder) {
	if pb.HasFragmentation() && pb.HasCRC() {
		enc.MustEncode(primaryBlock11{
			Version:            pb.Version,
			BundleControlFlags: pb.BundleControlFlags,
			CRCType:            pb.CRCType,
			Destination:        pb.Destination,
			SourceNode:         pb.SourceNode,
			ReportTo:           pb.ReportTo,
			CreationTimestamp:  pb.CreationTimestamp,
			Lifetime:           pb.Lifetime,
			FragmentOffset:     pb.FragmentOffset,
			TotalDataLength:    pb.TotalDataLength,
			CRC:                pb.CRC,
		})
	} else if pb.HasFragmentation() {
		enc.MustEncode(primaryBlock10{
			Version:            pb.Version,
			BundleControlFlags: pb.BundleControlFlags,
			CRCType:            pb.CRCType,
			Destination:        pb.Destination,
			SourceNode:         pb.SourceNode,
			ReportTo:           pb.ReportTo,
			CreationTimestamp:  pb.CreationTimestamp,
			Lifetime:           pb.Lifetime,
			FragmentOffset:     pb.FragmentOffset,
			TotalDataLength:    pb.TotalDataLength,
		})
	} else if pb.HasCRC() {
		enc.MustEncode(primaryBlock09{
			Version:            pb.Version,
			BundleControlFlags: pb.BundleControlFlags,
			CRCType:            pb.CRCType,
			Destination:        pb.Destination,
			SourceNode:         pb.SourceNode,
			ReportTo:           pb.ReportTo,
			CreationTimestamp:  pb.CreationTimestamp,
			Lifetime:           pb.Lifetime,
			CRC:                pb.CRC,
		})
	} else {
		enc.MustEncode(primaryBlock08{
			Version:            pb.Version,
			BundleControlFlags: pb.BundleControlFlags,
			CRCType:            pb.CRCType,
			Destination:        pb.Destination,
			SourceNode:         pb.SourceNode,
			ReportTo:           pb.ReportTo,
			CreationTimestamp:  pb.CreationTimestamp,
			Lifetime:           pb.Lifetime,
		})
	}
}

func (pb *PrimaryBlock) CodecDecodeSelf(dec *codec.Decoder) {
	// The implementation of the deserialization still sucks. I don't get codec
	// to decode a PrimaryBlock into primaryBlock{08-11}, because reasons.

	var pbx []interface{}
	dec.MustDecode(&pbx)

	pb.Version = dtnVersion
	pb.BundleControlFlags = BundleControlFlags(pbx[1].(uint64))
	pb.CRCType = CRCType(pbx[2].(uint64))
	pb.Lifetime = pbx[7].(uint64)

	setEndpointIDFromCborArray(&pb.Destination, pbx[3].([]interface{}))
	setEndpointIDFromCborArray(&pb.SourceNode, pbx[4].([]interface{}))
	setEndpointIDFromCborArray(&pb.ReportTo, pbx[5].([]interface{}))

	ct := pbx[6].([]interface{})
	pb.CreationTimestamp[0] = ct[0].(uint64)
	pb.CreationTimestamp[1] = ct[1].(uint64)

	if l := len(pbx); l == 11 {
		pb.FragmentOffset = pbx[8].(uint64)
		pb.TotalDataLength = pbx[9].(uint64)
		pb.CRC = pbx[10].([]byte)
	} else if l == 10 {
		pb.FragmentOffset = pbx[8].(uint64)
		pb.TotalDataLength = pbx[9].(uint64)
	} else if l == 9 {
		pb.CRC = pbx[8].([]byte)
	}
}

func (pb PrimaryBlock) checkValid() (errs error) {
	if pb.Version != dtnVersion {
		errs = multierror.Append(errs,
			newBundleError(fmt.Sprintf("PrimaryBlock: Wrong Version, %d instead of %d",
				pb.Version, dtnVersion)))
	}

	if bcfErr := pb.BundleControlFlags.checkValid(); bcfErr != nil {
		errs = multierror.Append(errs, bcfErr)
	}

	if destErr := pb.Destination.checkValid(); destErr != nil {
		errs = multierror.Append(errs, destErr)
	}

	if srcErr := pb.SourceNode.checkValid(); srcErr != nil {
		errs = multierror.Append(errs, srcErr)
	}

	if rprtToErr := pb.ReportTo.checkValid(); rprtToErr != nil {
		errs = multierror.Append(errs, rprtToErr)
	}

	// 4.1.3 says that "if the bundle's source node is omitted [src = dtn:none]
	// [...] the "Bundle must not be fragmented" flag value must be 1 and all
	// status report request flag values must be zero.
	// SourceNode == dtn:none => (
	//    MustNotFragmented
	//  & !"all status report flags")
	bpcfImpl := !(pb.SourceNode == DtnNone()) ||
		(pb.BundleControlFlags.Has(MustNotFragmented) &&
			!pb.BundleControlFlags.Has(StatusRequestReception) &&
			!pb.BundleControlFlags.Has(StatusRequestForward) &&
			!pb.BundleControlFlags.Has(StatusRequestDelivery) &&
			!pb.BundleControlFlags.Has(StatusRequestDeletion))
	if !bpcfImpl {
		errs = multierror.Append(errs,
			newBundleError("PrimaryBlock: Source Node is dtn:none, but Bundle could "+
				"be fragmented or status report flags are not zero"))
	}

	return
}

// IsLifetimeExceeded returns true if this PrimaryBlock's lifetime is exceeded.
// This method only compares the tuple of the CreationTimestamp and Lifetime
// against the current time.
//
// The hop count block and the bundle age block are not inspected by this method
// and should also be checked.
func (pb PrimaryBlock) IsLifetimeExceeded() bool {
	currentTs := time.Now()
	supremumTs := pb.CreationTimestamp.DtnTime().Time().Add(
		time.Duration(pb.Lifetime) * time.Microsecond)

	return currentTs.After(supremumTs)
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
