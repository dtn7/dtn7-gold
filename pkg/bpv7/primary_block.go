// SPDX-FileCopyrightText: 2018, 2019, 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/dtn7/cboring"
	"github.com/hashicorp/go-multierror"
)

const dtnVersion uint64 = 7

// PrimaryBlock is a representation of the primary bundle block as defined in section 4.3.1.
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
// other fields are set to default values. The lifetime is passed in milliseconds.
func NewPrimaryBlock(bundleControlFlags BundleControlFlags, destination EndpointID, sourceNode EndpointID, creationTimestamp CreationTimestamp, lifetime uint64) PrimaryBlock {
	pb := PrimaryBlock{
		Version:            dtnVersion,
		BundleControlFlags: bundleControlFlags,
		CRCType:            CRC32,
		Destination:        destination,
		SourceNode:         sourceNode,
		ReportTo:           sourceNode,
		CreationTimestamp:  creationTimestamp,
		Lifetime:           lifetime,
		FragmentOffset:     0,
		TotalDataLength:    0,
		CRC:                nil,
	}

	_ = pb.calculateCRC()
	return pb
}

// HasFragmentation returns true if the bundle processing control flags
// indicates a fragmented bundle. In this case the FragmentOffset and
// TotalDataLength fields should become relevant.
func (pb PrimaryBlock) HasFragmentation() bool {
	return pb.BundleControlFlags.Has(IsFragment)
}

// HasCRC returns if the CRCType indicates a CRC is present for this block.
func (pb PrimaryBlock) HasCRC() bool {
	return pb.GetCRCType() != CRCNo
}

// GetCRCType returns the CRCType of this block.
func (pb PrimaryBlock) GetCRCType() CRCType {
	return pb.CRCType
}

// SetCRCType sets the CRC type.
//
// While a primary block without a CRC might get parsed, created primary blocks
// MUST have a CRC value attached. Due to draft-bpbis-15 (and newer), all
// primary blocks require a CRC unless a block integrity block addressing the
// primary block is present. Thus, until BPsec is available in dtn7-go, all
// created primary blocks will have a mandatory CRC value.
func (pb *PrimaryBlock) SetCRCType(crcType CRCType) {
	// NOTE: Until BPsec has landed, set a CRC for primary blocks.
	if crcType == CRCNo {
		crcType = CRC32
	}

	pb.CRCType = crcType
	_ = pb.calculateCRC()
}

// calculateCRC serializes the PrimaryBlock once to calculate its CRC value.
// Since this block is immutable, this should not cause any errors. This method
// must be called both when creating the block and when changing its CRC.
func (pb *PrimaryBlock) calculateCRC() error {
	pb.CRC = nil
	return pb.MarshalCbor(new(bytes.Buffer))
}

// MarshalCbor writes the CBOR representation of a PrimaryBlock.
func (pb *PrimaryBlock) MarshalCbor(w io.Writer) error {
	blockLen := func() uint64 {
		switch frag, crc := pb.HasFragmentation(), pb.HasCRC(); {
		case !frag && !crc:
			return 8
		case !frag && crc:
			return 9
		case frag && !crc:
			return 10
		case frag && crc:
			return 11
		default:
			panic("impossible state")
		}
	}()

	crcBuff := new(bytes.Buffer)
	w = io.MultiWriter(w, crcBuff)

	if err := cboring.WriteArrayLength(blockLen, w); err != nil {
		return err
	}

	fields := []uint64{dtnVersion, uint64(pb.BundleControlFlags), uint64(pb.CRCType)}
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
		} else if !bytes.Equal(pb.CRC, crcVal) {
			pb.CRC = crcVal
		}
	}

	return nil
}

// UnmarshalCbor reads the CBOR representation of a PrimaryBlock.
func (pb *PrimaryBlock) UnmarshalCbor(r io.Reader) error {
	// Pipe incoming bytes into a separate CRC buffer
	crcBuff := new(bytes.Buffer)
	r = io.TeeReader(r, crcBuff)

	var blockLen uint64
	if bl, err := cboring.ReadArrayLength(r); err != nil {
		return err
	} else if !(8 <= bl && bl <= 11) {
		return fmt.Errorf("expected array with 8 to 11 elements, got %d", bl)
	} else {
		blockLen = bl
	}

	if version, err := cboring.ReadUInt(r); err != nil {
		return err
	} else if version != dtnVersion {
		return fmt.Errorf("expected version %d, got %d", dtnVersion, version)
	} else {
		pb.Version = dtnVersion
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
			return fmt.Errorf("invalid CRC value: %x instead of expected %x", crcVal, crcCalc)
		} else {
			pb.CRC = crcVal
		}
	}

	return nil
}

// MarshalJSON writes a JSON object representing this PrimaryBlock.
func (pb PrimaryBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ControlFlags      BundleControlFlags `json:"bundleControlFlags"`
		Destination       string             `json:"destination"`
		Source            string             `json:"source"`
		ReportTo          string             `json:"reportTo"`
		CreationTimestamp CreationTimestamp  `json:"creationTimestamp"`
		Lifetime          uint64             `json:"lifetime"`
	}{
		ControlFlags:      pb.BundleControlFlags,
		Destination:       pb.Destination.String(),
		Source:            pb.SourceNode.String(),
		ReportTo:          pb.ReportTo.String(),
		CreationTimestamp: pb.CreationTimestamp,
		Lifetime:          pb.Lifetime,
	})
}

// CheckValid returns an array of errors for incorrect data.
func (pb PrimaryBlock) CheckValid() (errs error) {
	if pb.Version != dtnVersion {
		errs = multierror.Append(errs,
			fmt.Errorf("PrimaryBlock: Wrong Version, %d instead of %d", pb.Version, dtnVersion))
	}

	if bcfErr := pb.BundleControlFlags.CheckValid(); bcfErr != nil {
		errs = multierror.Append(errs, bcfErr)
	}

	if destErr := pb.Destination.CheckValid(); destErr != nil {
		errs = multierror.Append(errs, destErr)
	}

	if srcErr := pb.SourceNode.CheckValid(); srcErr != nil {
		errs = multierror.Append(errs, srcErr)
	}

	if rprtToErr := pb.ReportTo.CheckValid(); rprtToErr != nil {
		errs = multierror.Append(errs, rprtToErr)
	}

	// 4.2.3 says that "if the bundle's source node is omitted [src = dtn:none]
	// [...] the bundle must not be fragmented" flag value must be 1 and all
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
			fmt.Errorf("PrimaryBlock: Source Node is dtn:none, but Bundle could "+
				"be fragmented or status report flags are not zero"))
	}

	return
}

func (pb PrimaryBlock) String() string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "version: %d, ", pb.Version)
	_, _ = fmt.Fprintf(&b, "bundle processing control flags: %b, ", pb.BundleControlFlags)
	_, _ = fmt.Fprintf(&b, "crc type: %v, ", pb.CRCType)
	_, _ = fmt.Fprintf(&b, "destination: %v, ", pb.Destination)
	_, _ = fmt.Fprintf(&b, "source node: %v, ", pb.SourceNode)
	_, _ = fmt.Fprintf(&b, "report to: %v, ", pb.ReportTo)
	_, _ = fmt.Fprintf(&b, "creation timestamp: %v, ", pb.CreationTimestamp)
	_, _ = fmt.Fprintf(&b, "lifetime: %d", pb.Lifetime)

	if pb.HasFragmentation() {
		_, _ = fmt.Fprintf(&b, " , ")
		_, _ = fmt.Fprintf(&b, "fragment offset: %d, ", pb.FragmentOffset)
		_, _ = fmt.Fprintf(&b, "total data length: %d", pb.TotalDataLength)
	}

	if pb.HasCRC() {
		_, _ = fmt.Fprintf(&b, ", crc: %x", pb.CRC)
	}

	return b.String()
}
