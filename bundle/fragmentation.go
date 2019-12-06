package bundle

import (
	"bytes"
	"fmt"
	"math"

	"github.com/dtn7/cboring"
)

// Fragment a Bundle into multiple Bundles, with each serialized Bundle limited to mtu bytes.
func (b Bundle) Fragment(mtu int) (bs []Bundle, err error) {
	if b.PrimaryBlock.BundleControlFlags.Has(MustNotFragmented) {
		err = fmt.Errorf("bundle control flags forbids bundle fragmentation")
		return
	}

	var (
		cborOverhead     int = 2
		extFirstOverhead int
		extOtherOverhead int

		payloadBlock    *CanonicalBlock
		payloadBlockLen int
	)

	if payloadBlock, err = b.PayloadBlock(); err != nil {
		return
	}
	payloadBlockLen = len(payloadBlock.Value.(*PayloadBlock).Data())

	if extFirstOverhead, extOtherOverhead, err = fragmentExtensionBlocksLen(b, mtu); err != nil {
		return
	}

	for i := 0; i < payloadBlockLen; {
		var (
			fragPrimaryBlock PrimaryBlock
			primaryOverhead  int
		)

		if fragPrimaryBlock, primaryOverhead, err = fragmentPrimaryBlock(b.PrimaryBlock, i, payloadBlockLen); err != nil {
			return
		}

		overhead := cborOverhead + primaryOverhead
		if i == 0 {
			overhead += extFirstOverhead
		} else {
			overhead += extOtherOverhead
		}

		if overhead >= mtu {
			err = fmt.Errorf("bundle overhead of fragment %d exceeds MTU", i)
			return
		}

		fragBundle := MustNewBundle(fragPrimaryBlock, nil)

		for _, cb := range b.CanonicalBlocks {
			if cb.TypeCode() == ExtBlockTypePayloadBlock {
				continue
			}
			if i > 0 && !cb.BlockControlFlags.Has(ReplicateBlock) {
				continue
			}

			fragBundle.AddExtensionBlock(cb)
		}

		fragPayloadBlockLen := mtu - overhead

		offset := int(math.Min(float64(i+fragPayloadBlockLen), float64(len(payloadBlock.Value.(*PayloadBlock).Data()))))
		fragBundle.AddExtensionBlock(CanonicalBlock{
			BlockControlFlags: payloadBlock.BlockControlFlags,
			CRCType:           CRC32,
			Value:             NewPayloadBlock(payloadBlock.Value.(*PayloadBlock).Data()[i:offset]),
		})

		if err = fragBundle.CheckValid(); err != nil {
			return
		}
		bs = append(bs, fragBundle)

		i += fragPayloadBlockLen
	}

	if len(bs) == 1 {
		bs = []Bundle{b}
	}

	return
}

// fragmentPrimaryBlock creates a fragment's Primary Block and calculates its length.
func fragmentPrimaryBlock(pb PrimaryBlock, fragmentOffset, totalDataLength int) (fragPb PrimaryBlock, l int, err error) {
	fragPb = PrimaryBlock{
		Version:            pb.Version,
		BundleControlFlags: pb.BundleControlFlags | IsFragment,
		CRCType:            CRC32,
		Destination:        pb.Destination,
		SourceNode:         pb.SourceNode,
		ReportTo:           pb.ReportTo,
		CreationTimestamp:  pb.CreationTimestamp,
		Lifetime:           pb.Lifetime,
		FragmentOffset:     uint64(fragmentOffset),
		TotalDataLength:    uint64(totalDataLength),
	}

	buff := new(bytes.Buffer)

	err = fragPb.MarshalCbor(buff)
	l = buff.Len()
	return
}

// fragmentExtensionBlocksLen calculates the estimated maximum length for the Extension Blocks for the
// first and the other fragments.
func fragmentExtensionBlocksLen(b Bundle, mtu int) (first int, others int, err error) {
	buff := new(bytes.Buffer)

	for _, cb := range b.CanonicalBlocks {
		if cb.TypeCode() == ExtBlockTypePayloadBlock {
			cb = CanonicalBlock{
				BlockNumber:       cb.BlockNumber,
				BlockControlFlags: cb.BlockControlFlags,
				Value:             NewPayloadBlock(nil),
			}
		}

		cb.CRCType = CRC32

		if err = cb.MarshalCbor(buff); err != nil {
			return
		}

		cbLen := buff.Len()
		first += cbLen
		if cb.BlockControlFlags.Has(ReplicateBlock) {
			others += cbLen
		}

		if cb.TypeCode() == ExtBlockTypePayloadBlock {
			// Update the byte string length field
			buff.Reset()
			if err = cboring.WriteByteStringLen(uint64(mtu), buff); err != nil {
				return
			}
			first += buff.Len() - 1
			others += cbLen + buff.Len() - 1
		}

		buff.Reset()
	}

	return
}
