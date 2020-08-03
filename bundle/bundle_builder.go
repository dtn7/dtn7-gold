package bundle

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"time"
)

// BundleBuilder is a simple framework to create bundles by method chaining.
//
//   bndl, err := bundle.Builder().
//     CRC(bundle.CRC32).
//     Source("dtn://src/").
//     Destination("dtn://dest/").
//     CreationTimestampNow().
//     Lifetime("30m").
//     HopCountBlock(64).
//     PayloadBlock([]byte("hello world!")).
//     Build()
//
type BundleBuilder struct {
	err error

	primary          PrimaryBlock
	canonicals       []CanonicalBlock
	canonicalCounter uint64
	crcType          CRCType
}

// Builder creates a new BundleBuilder.
func Builder() *BundleBuilder {
	return &BundleBuilder{
		err: nil,

		primary:          PrimaryBlock{Version: dtnVersion},
		canonicals:       []CanonicalBlock{},
		canonicalCounter: 2,
		crcType:          CRCNo,
	}
}

// Error returns the BundleBuilder's error, if one is present.
func (bldr *BundleBuilder) Error() error {
	return bldr.err
}

// CRC sets the bundle's CRC value.
func (bldr *BundleBuilder) CRC(crcType CRCType) *BundleBuilder {
	if bldr.err == nil {
		bldr.crcType = crcType
	}

	return bldr
}

// Build creates a new Bundle and returns an optional error.
func (bldr *BundleBuilder) Build() (bndl Bundle, err error) {
	if bldr.err != nil {
		err = bldr.err
		return
	}

	// Set ReportTo to Source, if it was not set before
	if bldr.primary.ReportTo == (EndpointID{}) {
		bldr.primary.ReportTo = bldr.primary.SourceNode
	}

	// Source and Destination are necessary
	if bldr.primary.SourceNode == (EndpointID{}) || bldr.primary.Destination == (EndpointID{}) {
		err = fmt.Errorf("both Source and Destination must be set")
		return
	}

	// Calculate mandatory CRC for the PrimaryBlock.
	// Do NOT alter the PrimaryBlock after setting its CRC.
	if bldr.crcType == CRCNo {
		bldr.primary.SetCRCType(CRC32)
	} else {
		bldr.primary.SetCRCType(bldr.crcType)
	}

	sort.Sort(canonicalBlockNumberSort(bldr.canonicals))

	bndl, err = NewBundle(bldr.primary, bldr.canonicals)
	if err == nil {
		bndl.SetCRCType(bldr.crcType)
	}

	return
}

// mustBuild is like Build, but panics on an error. This method is only intended for internal testing.
func (bldr *BundleBuilder) mustBuild() Bundle {
	if b, err := bldr.Build(); err != nil {
		panic(err)
	} else {
		return b
	}
}

// Helper functions

// bldrParseEndpoint returns an EndpointID for a given EndpointID or a string,
// representing an endpoint identifier as an URI.
func bldrParseEndpoint(eid interface{}) (e EndpointID, err error) {
	switch eid := eid.(type) {
	case EndpointID:
		e = eid
	case string:
		e, err = NewEndpointID(eid)
	default:
		err = fmt.Errorf("%T is neither an EndpointID nor a string", eid)
	}
	return
}

// bldrParseLifetime returns a millisecond as an uint for a given millisecond or a duration string, which will be parsed.
func bldrParseLifetime(duration interface{}) (ms uint64, err error) {
	switch duration := duration.(type) {
	case uint64:
		ms = duration
	case int:
		if duration < 0 {
			err = fmt.Errorf("lifetime's duration %d <= 0", duration)
		} else {
			ms = uint64(duration)
		}
	case float64:
		if duration < 0 {
			err = fmt.Errorf("lifetime's duration %f <= 0", duration)
		} else {
			ms = uint64(duration)
		}
	case string:
		dur, durErr := time.ParseDuration(duration)
		if durErr != nil {
			err = durErr
		} else if dur <= 0 {
			err = fmt.Errorf("lifetime's duration %d <= 0", dur)
		} else {
			ms = uint64(dur.Milliseconds())
		}
	case time.Duration:
		ms = uint64(duration.Milliseconds())
	default:
		err = fmt.Errorf("%T is an unsupported type to parse a Duration from", duration)
	}
	return
}

// PrimaryBlock related methods

// Destination sets the bundle's destination, stored in its primary block.
func (bldr *BundleBuilder) Destination(eid interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	if e, err := bldrParseEndpoint(eid); err != nil {
		bldr.err = err
	} else {
		bldr.primary.Destination = e
	}

	return bldr
}

// Source sets the bundle's source, stored in its primary block.
func (bldr *BundleBuilder) Source(eid interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	if e, err := bldrParseEndpoint(eid); err != nil {
		bldr.err = err
	} else {
		bldr.primary.SourceNode = e
	}

	return bldr
}

// ReportTo sets the bundle's report-to address, stored in its primary block.
func (bldr *BundleBuilder) ReportTo(eid interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	if e, err := bldrParseEndpoint(eid); err != nil {
		bldr.err = err
	} else {
		bldr.primary.ReportTo = e
	}

	return bldr
}

// creationTimestamp sets the bundle's creation timestamp.
func (bldr *BundleBuilder) creationTimestamp(t DtnTime) *BundleBuilder {
	if bldr.err == nil {
		bldr.primary.CreationTimestamp = NewCreationTimestamp(t, 0)
	}

	return bldr
}

// CreationTimestampEpoch sets the bundle's creation timestamp to the epoch
// time, stored in its primary block.
func (bldr *BundleBuilder) CreationTimestampEpoch() *BundleBuilder {
	return bldr.creationTimestamp(DtnTimeEpoch)
}

// CreationTimestampNow sets the bundle's creation timestamp to the current
// time, stored in its primary block.
func (bldr *BundleBuilder) CreationTimestampNow() *BundleBuilder {
	return bldr.creationTimestamp(DtnTimeNow())
}

// CreationTimestampTime sets the bundle's creation timestamp to a given time,
// stored in its primary block.
func (bldr *BundleBuilder) CreationTimestampTime(t time.Time) *BundleBuilder {
	return bldr.creationTimestamp(DtnTimeFromTime(t))
}

// Lifetime sets the bundle's lifetime, stored in its primary block. Possible
// values are an uint/int, representing the lifetime in milliseconds, a format
// string (compare time.ParseDuration) for the duration or a time.Duration.
//
//   Lifetime(1000)             // Lifetime of 1000ms
//   Lifetime("1000ms")         // Lifetime of 1000ms
//   Lifetime("10m")            // Lifetime of 10min
//   Lifetime(10 * time.Minute) // Lifetime of 10min
//
func (bldr *BundleBuilder) Lifetime(duration interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	if us, usErr := bldrParseLifetime(duration); usErr != nil {
		bldr.err = usErr
	} else {
		bldr.primary.Lifetime = us
	}

	return bldr
}

// BundleCtrlFlags sets the bundle processing control flags in the primary block.
func (bldr *BundleBuilder) BundleCtrlFlags(bcf BundleControlFlags) *BundleBuilder {
	if bldr.err == nil {
		bldr.primary.BundleControlFlags = bcf
	}

	return bldr
}

// CanonicalBlock related methods

// Canonical adds a canonical block to this bundle. The parameters are:
//
//   ExtensionBlock[, BlockControlFlags] or
//   CanonicalBlock
//
//   where ExtensionBlock is a bundle.ExtensionBlock and
//   BlockControlFlags are _optional_ block processing control flags or
//   CanonicalBlock is a CanonicalBlock
//
func (bldr *BundleBuilder) Canonical(args ...interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	if len(args) == 0 {
		bldr.err = fmt.Errorf("Canonical was called with no parameters")
		return bldr
	}

	var (
		blockNumber    uint64
		data           ExtensionBlock
		blockCtrlFlags BlockControlFlags
	)

	switch args[0].(type) {
	case ExtensionBlock:
		var chk0, chk1 bool

		switch l := len(args); l {
		case 1:
			data, chk0 = args[0].(ExtensionBlock)
			chk1 = true // Only one check here, so the other one is always true.
		case 2:
			data, chk0 = args[0].(ExtensionBlock)
			blockCtrlFlags, chk1 = args[1].(BlockControlFlags)
		default:
			bldr.err = fmt.Errorf(
				"Canonical was called with neither one nor two parameters")
			return bldr
		}

		if !(chk0 && chk1) {
			bldr.err = fmt.Errorf("Canonical received wrong parameter types, %v %v", chk0, chk1)
			return bldr
		}

		if data.BlockTypeCode() == ExtBlockTypePayloadBlock {
			blockNumber = 1
		} else {
			blockNumber = bldr.canonicalCounter
			bldr.canonicalCounter++
		}

		bldr.canonicals = append(bldr.canonicals,
			NewCanonicalBlock(blockNumber, blockCtrlFlags, data))

	case CanonicalBlock:
		cb := args[0].(CanonicalBlock)
		if cb.TypeCode() == ExtBlockTypePayloadBlock {
			blockNumber = 1
		} else {
			blockNumber = bldr.canonicalCounter
			bldr.canonicalCounter++
		}
		cb.BlockNumber = blockNumber

		bldr.canonicals = append(bldr.canonicals, cb)

	default:
		bldr.err = fmt.Errorf("Canonicals received unknown type")
	}

	return bldr
}

// BundleAgeBlock adds a bundle age block to this bundle. The parameters are:
//
//   Age[, BlockControlFlags]
//
//   where Age is the age as an uint in milliseconds, a format string or a time.Duration
//   and BlockControlFlags are _optional_ block processing control flags
//
func (bldr *BundleBuilder) BundleAgeBlock(args ...interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	ms, msErr := bldrParseLifetime(args[0])
	if msErr != nil {
		bldr.err = msErr
	}

	// Call Canonical as a variadic function with:
	// - ExtensionBlock: BundleAgeBlock with parsed milliseconds
	// - BlockControlFlags: BlockControlFlags, if given
	return bldr.Canonical(
		append([]interface{}{NewBundleAgeBlock(ms)}, args[1:]...)...)
}

// HopCountBlock adds a hop count block to this bundle. The parameters are:
//
//   Limit[, BlockControlFlags]
//
//   where Limit is the limit of this Hop Count Block and
//   BlockControlFlags are _optional_ block processing control flags
//
func (bldr *BundleBuilder) HopCountBlock(args ...interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	limit, chk := args[0].(int)
	if !chk {
		bldr.err = fmt.Errorf("HopCountBlock received wrong parameter type")
	}

	// Read the comment in BundleAgeBlock to grasp the following madness
	return bldr.Canonical(append(
		[]interface{}{NewHopCountBlock(uint8(limit))}, args[1:]...)...)
}

// PayloadBlock adds a payload block to this bundle. The parameters are:
//
//   Data[, BlockControlFlags]
//
//   where Data is the payload's data and
//   BlockControlFlags are _optional_ block processing control flags
func (bldr *BundleBuilder) PayloadBlock(args ...interface{}) *BundleBuilder {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, args[0]); err != nil {
		bldr.err = err
		return bldr
	}

	// Call Canonical, but add PayloadBlock as the first variadic parameter
	return bldr.Canonical(append(
		[]interface{}{NewPayloadBlock(buf.Bytes())}, args[1:]...)...)
}

// PreviousNodeBlock adds a previous node block to this bundle. The parameters
// are:
//
//   PrevNode[, BlockControlFlags]
//
//   where PrevNode is an EndpointID or a string describing an endpoint and
//   BlockControlFlags are _optional_ block processing control flags
//
func (bldr *BundleBuilder) PreviousNodeBlock(args ...interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	eid, eidErr := bldrParseEndpoint(args[0])
	if eidErr != nil {
		bldr.err = eidErr
	}

	return bldr.Canonical(
		append([]interface{}{NewPreviousNodeBlock(eid)}, args[1:]...)...)
}

// BuildFromMap creates a Bundle from a map which "calls" the BundleBuilder's methods.
//
// This function does not use reflection or other dark magic. So it is safe to be called by unchecked data.
//
//   args := map[string]interface{}{
//     "destination":            "dtn://dst/",
//     "source":                 "dtn://src/",
//     "creation_timestamp_now": true,
//     "lifetime":               "24h",
//     "payload_block":          "hello world",
//   }
//   b, err := BuildFromMap(args)
//
func BuildFromMap(m map[string]interface{}) (bndl Bundle, err error) {
	bldr := Builder()

	for method, args := range m {
		switch method {
		// func (bldr *BundleBuilder) Destination(eid interface{}) *BundleBuilder
		case "destination":
			bldr.Destination(args)

		// func (bldr *BundleBuilder) Source(eid interface{}) *BundleBuilder
		case "source":
			bldr.Source(args)

		// func (bldr *BundleBuilder) ReportTo(eid interface{}) *BundleBuilder
		case "report_to":
			bldr.ReportTo(args)

		// func (bldr *BundleBuilder) CreationTimestampEpoch() *BundleBuilder
		case "creation_timestamp_epoch":
			bldr.CreationTimestampEpoch()

		// func (bldr *BundleBuilder) CreationTimestampNow() *BundleBuilder
		case "creation_timestamp_now":
			bldr.CreationTimestampNow()

		// func (bldr *BundleBuilder) CreationTimestampTime(t time.Time) *BundleBuilder
		case "creation_timestamp_time":
			if argsT, ok := args.(time.Time); ok {
				bldr.CreationTimestampTime(argsT)
			} else {
				err = fmt.Errorf("creation_timestamp_time needs a time.Time, not %T", args)
			}

		// func (bldr *BundleBuilder) Lifetime(duration interface{}) *BundleBuilder
		case "lifetime":
			bldr.Lifetime(args)

		// func (bldr *BundleBuilder) BundleCtrlFlags(bcf BundleControlFlags) *BundleBuilder
		case "bundle_ctrl_flags":
			// TODO: implement
			err = fmt.Errorf("bundle_ctrl_flags is not yet implemented")

		// func (bldr *BundleBuilder) Canonical(args ...interface{}) *BundleBuilder
		case "canonical":
			err = fmt.Errorf("canonical is not impelemtend")

		// func (bldr *BundleBuilder) BundleAgeBlock(args ...interface{}) *BundleBuilder
		case "bundle_age_block":
			bldr.BundleAgeBlock(args)

		// func (bldr *BundleBuilder) HopCountBlock(args ...interface{}) *BundleBuilder
		case "hop_count_block":
			bldr.HopCountBlock(args)

		// func (bldr *BundleBuilder) PayloadBlock(args ...interface{}) *BundleBuilder
		case "payload_block":
			if sArgs, ok := args.(string); ok {
				bldr.PayloadBlock([]byte(sArgs))
			} else {
				bldr.PayloadBlock(args)
			}

		// func (bldr *BundleBuilder) PreviousNodeBlock(args ...interface{}) *BundleBuilder
		case "previous_node_block":
			bldr.PreviousNodeBlock(args)

		default:
			err = fmt.Errorf("method %s is either not implemented or not existing", method)
		}

		if err == nil {
			err = bldr.Error()
		}
		if err != nil {
			return
		}
	}

	return bldr.Build()
}
