package bpa

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/ugorji/go/codec"
)

const dtnVersion uint = 7

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
	CRC                []byte
}

// NewPrimaryBlock creates a new PrimaryBlock with the given parameters. All
// other fields are set to default values.
func NewPrimaryBlock(bundleControlFlags BundleControlFlags,
	destination EndpointID, sourceNode EndpointID,
	creationTimestamp CreationTimestamp, lifetime uint) PrimaryBlock {
	return PrimaryBlock{
		Version:            dtnVersion,
		BundleControlFlags: bundleControlFlags,
		CRCType:            CRCNo,
		Destination:        destination,
		SourceNode:         sourceNode,
		ReportTo:           DtnNone(),
		CreationTimestamp:  creationTimestamp,
		Lifetime:           lifetime,
		FragmentOffset:     0,
		TotalDataLength:    0,
		CRC:                nil,
	}
}

// HasFragmentation returns if the Bundle Processing Control Flags indicates a
// fragmented bundle. In this case the FragmentOffset and TotalDataLength fields
// of this struct should become relevant.
func (pb PrimaryBlock) HasFragmentation() bool {
	return pb.BundleControlFlags.Has(BndlCFBundleIsAFragment)
}

// HasCRC retruns true if the CRCType indicates a CRC present for this block.
// In this case the CRC value should become relevant.
func (pb PrimaryBlock) HasCRC() bool {
	return pb.GetCRCType() != CRCNo
}

// GetCRCType returns the CRCType of this Block.
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
// This method changes the block's CRC value temporary and is not thread safe.
func (pb *PrimaryBlock) CalculateCRC() {
	pb.setCRC(calculateCRC(pb))
}

// CheckCRC returns true if the CRC value matches to its CRCType or the
// CRCType is CRCNo.
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

func (pb PrimaryBlock) CodecEncodeSelf(enc *codec.Encoder) {
	var blockArr = []interface{}{
		pb.Version,
		pb.BundleControlFlags,
		pb.CRCType,
		pb.Destination,
		pb.SourceNode,
		pb.ReportTo,
		pb.CreationTimestamp,
		pb.Lifetime}

	if pb.HasFragmentation() {
		blockArr = append(blockArr, pb.FragmentOffset, pb.TotalDataLength)
	}

	if pb.HasCRC() {
		blockArr = append(blockArr, pb.CRC)
	}

	enc.MustEncode(blockArr)
}

// decodeEndpoints decodes the three defined EndpointIDs. This method is called
// from CodecDecodeSelf.
func (pb *PrimaryBlock) decodeEndpoints(blockArr []interface{}) {
	endpoints := []struct {
		pos     int
		pointer *EndpointID
	}{
		{3, &pb.Destination},
		{4, &pb.SourceNode},
		{5, &pb.ReportTo},
	}

	for _, ep := range endpoints {
		var arr []interface{} = blockArr[ep.pos].([]interface{})
		setEndpointIDFromCborArray(ep.pointer, arr)
	}
}

// decodeCreationTimestamp decodes the CreationTimestamp. This method is called
// from CodecDecodeSelf.
func (pb *PrimaryBlock) decodeCreationTimestamp(blockArr []interface{}) {
	for i := 0; i <= 1; i++ {
		pb.CreationTimestamp[i] = uint((blockArr[6].([]interface{}))[i].(uint64))
	}
}

func (pb *PrimaryBlock) CodecDecodeSelf(dec *codec.Decoder) {
	var blockArrPt = new([]interface{})
	dec.MustDecode(blockArrPt)

	var blockArr = *blockArrPt

	if len(blockArr) < 8 || len(blockArr) > 11 {
		panic("blockArr has wrong length (< 8 or > 10)")
	}

	pb.decodeEndpoints(blockArr)
	pb.decodeCreationTimestamp(blockArr)

	pb.Version = uint(blockArr[0].(uint64))
	pb.BundleControlFlags = BundleControlFlags(blockArr[1].(uint64))
	pb.CRCType = CRCType(blockArr[2].(uint64))
	pb.Lifetime = uint(blockArr[7].(uint64))

	switch len(blockArr) {
	case 9:
		// CRC, No Fragmentation
		pb.CRC = blockArr[8].([]byte)

	case 10:
		// No CRC, Fragmentation
		pb.FragmentOffset = uint(blockArr[8].(uint64))
		pb.TotalDataLength = uint(blockArr[9].(uint64))

	case 11:
		// CRC, Fragmentation
		pb.FragmentOffset = uint(blockArr[8].(uint64))
		pb.TotalDataLength = uint(blockArr[9].(uint64))
		pb.CRC = blockArr[10].([]byte)
	}
}

func (pb PrimaryBlock) checkValid() (errs error) {
	if pb.Version != dtnVersion {
		errs = multierror.Append(errs,
			newBPAError(fmt.Sprintf("PrimaryBlock: Wrong Version, %d instead of %d",
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
	//    BndlCFBundleMustNotBeFragmented
	//  & !"all status report flags")
	bpcfImpl := !(pb.SourceNode == DtnNone()) ||
		(pb.BundleControlFlags.Has(BndlCFBundleMustNotBeFragmented) &&
			!pb.BundleControlFlags.Has(BndlCFBundleReceptionStatusReportsAreRequested) &&
			!pb.BundleControlFlags.Has(BndlCFBundleForwardingStatusReportsAreRequested) &&
			!pb.BundleControlFlags.Has(BndlCFBundleDeliveryStatusReportsAreRequested) &&
			!pb.BundleControlFlags.Has(BndlCFBundleDeletionStatusReportsAreRequested))
	if !bpcfImpl {
		errs = multierror.Append(errs,
			newBPAError("PrimaryBlock: Source Node is dtn:none, but Bundle could be "+
				"fragmented or status report flags are not zero"))
	}

	return
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
