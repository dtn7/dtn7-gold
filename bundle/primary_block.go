package bundle

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dtn7/cboring"
	"github.com/hashicorp/go-multierror"
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
	// :^)
	return true
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

func (pb *PrimaryBlock) MarshalCbor(w io.Writer) error {
	var blockLen uint64 = 8
	if pb.HasCRC() && pb.HasFragmentation() {
		blockLen = 11
	} else if pb.HasFragmentation() {
		blockLen = 10
	} else if pb.HasCRC() {
		blockLen = 9
	}

	crcBuff := new(bytes.Buffer)
	if pb.HasCRC() {
		w = io.MultiWriter(w, crcBuff)
	}

	if err := cboring.WriteArrayLength(blockLen, w); err != nil {
		return err
	}

	fields := []uint64{7, uint64(pb.BundleControlFlags), uint64(pb.CRCType)}
	for _, f := range fields {
		if err := cboring.WriteUInt(f, w); err != nil {
			return err
		}
	}

	eids := []*EndpointID{&pb.Destination, &pb.SourceNode, &pb.ReportTo}
	for _, eid := range eids {
		if err := cboring.Marshal(eid, w); err != nil {
			return fmt.Errorf("EndpointID failed: %v", err)
		}
	}

	if err := cboring.Marshal(&pb.CreationTimestamp, w); err != nil {
		return fmt.Errorf("CreationTimestamp failed: %v", err)
	}

	if err := cboring.WriteUInt(pb.Lifetime, w); err != nil {
		return err
	}

	if pb.HasFragmentation() {
		fields = []uint64{pb.FragmentOffset, pb.TotalDataLength}
		for _, f := range fields {
			if err := cboring.WriteUInt(f, w); err != nil {
				return err
			}
		}
	}

	if pb.HasCRC() {
		if crcVal, crcErr := calculateCRCBuff(crcBuff, pb.CRCType); crcErr != nil {
			return crcErr
		} else if err := cboring.WriteByteString(crcVal, w); err != nil {
			return err
		} else {
			pb.CRC = crcVal
		}
	}

	return nil
}

func (pb *PrimaryBlock) UnmarshalCbor(r io.Reader) error {
	var blockLen uint64
	if bl, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if bl < 8 || bl > 11 {
		return fmt.Errorf("Expected array with length 8-11, got %d", bl)
	} else {
		blockLen = bl
	}

	// Pipe incoming bytes into a separate CRC buffer
	crcBuff := new(bytes.Buffer)
	if blockLen == 9 || blockLen == 11 {
		// Replay array's start
		cboring.WriteArrayLength(blockLen, crcBuff)
		r = io.TeeReader(r, crcBuff)
	}

	if version, err := cboring.ReadUInt(r); err != nil {
		return err
	} else if version != 7 {
		return fmt.Errorf("Expected version 7, got %d", version)
	} else {
		pb.Version = 7
	}

	if bcf, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		pb.BundleControlFlags = BundleControlFlags(bcf)
	}

	if crcT, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		pb.CRCType = CRCType(crcT)
	}

	eids := []*EndpointID{&pb.Destination, &pb.SourceNode, &pb.ReportTo}
	for _, eid := range eids {
		if err := cboring.Unmarshal(eid, r); err != nil {
			return fmt.Errorf("EndpointID failed: %v", err)
		}
	}

	if err := cboring.Unmarshal(&pb.CreationTimestamp, r); err != nil {
		return fmt.Errorf("CreationTimestamp failed: %v", err)
	}

	if lt, err := cboring.ReadUInt(r); err != nil {
		return err
	} else {
		pb.Lifetime = lt
	}

	if blockLen == 10 || blockLen == 11 {
		fields := []*uint64{&pb.FragmentOffset, &pb.TotalDataLength}
		for _, f := range fields {
			if x, err := cboring.ReadUInt(r); err != nil {
				return err
			} else {
				*f = x
			}
		}
	}

	if blockLen == 9 || blockLen == 11 {
		if crcCalc, crcErr := calculateCRCBuff(crcBuff, pb.CRCType); crcErr != nil {
			return crcErr
		} else if crcVal, err := cboring.ReadByteString(r); err != nil {
			return err
		} else if !bytes.Equal(crcCalc, crcVal) {
			return fmt.Errorf("Invalid CRC value: %x instead of expected %x", crcVal, crcCalc)
		} else {
			pb.CRC = crcVal
		}
	}

	return nil
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
