package bundle

import (
	"fmt"
	"time"
)

// BundleBuilder is WIP, TODO.
type BundleBuilder struct {
	err error

	primary          PrimaryBlock
	canonicals       []CanonicalBlock
	canonicalCounter uint
	crcType          CRCType
}

func Builder() *BundleBuilder {
	return &BundleBuilder{
		err: nil,

		primary:          PrimaryBlock{Version: dtnVersion},
		canonicals:       []CanonicalBlock{},
		canonicalCounter: 1,
		crcType:          CRCNo,
	}
}

func (bldr *BundleBuilder) Error() error {
	return bldr.err
}

func (bldr *BundleBuilder) CRC(crcType CRCType) *BundleBuilder {
	if bldr.err == nil {
		bldr.crcType = crcType
	}

	return bldr
}

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
		err = fmt.Errorf("Both Source and Destination must be set")
		return
	}

	// TODO: sort canonicals

	bndl, err = NewBundle(bldr.primary, bldr.canonicals)
	if err == nil {
		bndl.SetCRCType(bldr.crcType)
		bndl.CalculateCRC()
	}

	return
}

// Helper functions

// bldrParseEndpoint returns an EndpointID for a given EndpointID or a string,
// representing an endpoint identifier as an URI.
func bldrParseEndpoint(eid interface{}) (e EndpointID, err error) {
	switch eid.(type) {
	case EndpointID:
		e = eid.(EndpointID)
	case string:
		e, err = NewEndpointID(eid.(string))
	default:
		err = fmt.Errorf("%T is neither an EndpointID nor a string", eid)
	}
	return
}

// bldrParseLifetime returns a microsecond as an uint for a given microsecond
// or a duration string, which will be parsed.
func bldrParseLifetime(duration interface{}) (us uint, err error) {
	switch duration.(type) {
	case uint:
		us = duration.(uint)
	case int:
		us = uint(duration.(int))
	case string:
		dur, durErr := time.ParseDuration(duration.(string))
		if durErr != nil {
			err = durErr
		} else if dur <= 0 {
			err = fmt.Errorf("Lifetime's duration %d <= 0", dur)
		} else {
			us = uint(dur.Nanoseconds() / 1000)
		}
	default:
		err = fmt.Errorf(
			"%T is neither an uin nor a string for a Duration", duration)
	}
	return
}

// PrimaryBlock related methods

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

func (bldr *BundleBuilder) creationTimestamp(t DtnTime) *BundleBuilder {
	if bldr.err == nil {
		bldr.primary.CreationTimestamp = NewCreationTimestamp(t, 0)
	}

	return bldr
}

func (bldr *BundleBuilder) CreationTimestampEpoch() *BundleBuilder {
	return bldr.creationTimestamp(DtnTimeEpoch)
}

func (bldr *BundleBuilder) CreationTimestampNow() *BundleBuilder {
	return bldr.creationTimestamp(DtnTimeNow())
}

func (bldr *BundleBuilder) CreationTimestampTime(t time.Time) *BundleBuilder {
	return bldr.creationTimestamp(DtnTimeFromTime(t))
}

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

func (bldr *BundleBuilder) BundleCtrlFlags(bcf BundleControlFlags) *BundleBuilder {
	if bldr.err == nil {
		bldr.primary.BundleControlFlags = bcf
	}

	return bldr
}

// CanonicalBlock related methods

// Canonical: BlockType, Data[, BlockControlFlags]
func (bldr *BundleBuilder) Canonical(args ...interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	var (
		blockNumber    uint
		blockType      CanonicalBlockType
		data           interface{}
		blockCtrlFlags BlockControlFlags

		chk0, chk1 bool = true, true
	)

	switch l := len(args); l {
	case 2:
		blockType, chk0 = args[0].(CanonicalBlockType)
		data = args[1]
	case 3:
		blockType, chk0 = args[0].(CanonicalBlockType)
		data = args[1]
		blockCtrlFlags, chk1 = args[2].(BlockControlFlags)
	default:
		bldr.err = fmt.Errorf(
			"Canonical was called with neither two nor three parameters")
		return bldr
	}

	if !(chk0 && chk1) {
		bldr.err = fmt.Errorf("Canonical received wrong parameter types, %v %v", chk0, chk1)
		return bldr
	}

	if blockType == PayloadBlock {
		blockNumber = 0
	} else {
		blockNumber = bldr.canonicalCounter
		bldr.canonicalCounter++
	}

	bldr.canonicals = append(bldr.canonicals,
		NewCanonicalBlock(blockType, blockNumber, blockCtrlFlags, data))

	return bldr
}

// BundleAgeBlock: Age[, BlockControlFlags]
// Age <- { us as uint, duration as string }
func (bldr *BundleBuilder) BundleAgeBlock(args ...interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	us, usErr := bldrParseLifetime(args[0])
	if usErr != nil {
		bldr.err = usErr
	}

	// Call Canonical as a variadic function with:
	// - BlockType: BundleAgeBlock,
	// - Data: us (us parsed from given age)
	// - BlockControlFlags: BlockControlFlags, if given
	return bldr.Canonical(
		append([]interface{}{BundleAgeBlock, us}, args[1:]...)...)
}

// HopCountBlock: Limit[, BlockControlFlags]
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
		[]interface{}{HopCountBlock, NewHopCount(uint(limit))}, args[1:]...)...)
}

// PayloadBlock: Data[, BlockControlFlags]
func (bldr *BundleBuilder) PayloadBlock(args ...interface{}) *BundleBuilder {
	// Call Canonical, but add PayloadBlock as the first variadic parameter
	return bldr.Canonical(append([]interface{}{PayloadBlock}, args...)...)
}

// PreviousNodeBlock: PrevNode[, BlockControlFlags]
// PrevNode <- { EndpointID, endpoint as string }
func (bldr *BundleBuilder) PreviousNodeBlock(args ...interface{}) *BundleBuilder {
	if bldr.err != nil {
		return bldr
	}

	eid, eidErr := bldrParseEndpoint(args[0])
	if eidErr != nil {
		bldr.err = eidErr
	}

	return bldr.Canonical(
		append([]interface{}{PreviousNodeBlock, eid}, args[1:]...)...)
}
