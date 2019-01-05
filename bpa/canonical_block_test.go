package bpa

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ugorji/go/codec"
)

func TestNewCanonicalBlock(t *testing.T) {
	b := NewPayloadBlock(
		BlckCFBlockMustBeReplicatedInEveryFragment, []byte("hello world"))

	if b.HasCRC() {
		t.Errorf("Canonical Block (Payload Block) has CRC: %v", b)
	}

	b.CRCType = CRC32
	if !b.HasCRC() {
		t.Errorf("Canonical Block (Payload Block) has no CRC: %v", b)
	}
}

func TestCanonicalBlockCbor(t *testing.T) {
	ep, _ := NewEndpointID("dtn", "foo/bar")

	tests := []struct {
		cb1 CanonicalBlock
		len int
	}{
		// Generic CanonicalBlock: No CRC
		{CanonicalBlock{1, 0, 0, CRCNo, []byte("hello world"), nil}, 5},
		// Generic CanonicalBlock: CRC
		{CanonicalBlock{1, 0, 0, CRC16, []byte("hello world"), nil}, 6},
		// Payload block
		{NewPayloadBlock(0, []byte("test")), 5},
		// Previous Node block (dtn:none)
		{NewPreviousNodeBlock(23, 0, *DtnNone), 5},
		// Previous Node block (dtn:foo/bar)
		{NewPreviousNodeBlock(23, 0, *ep), 5},
		// Bundle Age block
		{NewBundleAgeBlock(23, 0, 100000), 5},
		// Hop Count block
		{NewHopCountBlock(23, 0, HopCount{Limit: 100, Count: 0}), 5},
	}

	for _, test := range tests {
		var b []byte = make([]byte, 0, 64)
		var h codec.Handle = new(codec.CborHandle)
		var enc *codec.Encoder = codec.NewEncoderBytes(&b, h)

		// If we are going to test block's with a CRC value, we also have to
		// calculate it.
		if test.cb1.HasCRC() {
			SetCRC(&test.cb1)
		}

		err := enc.Encode(test.cb1)
		if err != nil {
			t.Errorf("CBOR encoding failed: %v", err)
		}

		var decGeneric interface{}
		err = codec.NewDecoderBytes(b, h).Decode(&decGeneric)
		if err != nil {
			t.Errorf("Generic CBOR decoding failed: %v", err)
		}

		if ty := reflect.TypeOf(decGeneric); ty.Kind() != reflect.Slice {
			t.Errorf("Decoded CBOR has wrong type: %v instead of slice", ty.Kind())
		}

		var arr []interface{} = decGeneric.([]interface{})
		if len(arr) != test.len {
			t.Errorf("CBOR-Array has wrong length: %d instead of %d",
				len(arr), test.len)
		}

		var cb2 CanonicalBlock
		err = codec.NewDecoderBytes(b, h).Decode(&cb2)
		if err != nil {
			t.Errorf("CBOR decoding failed for %v: %v", test, err)
		}

		v1 := reflect.ValueOf(test.cb1)
		v2 := reflect.ValueOf(cb2)

		if v1.NumField() != v2.NumField() {
			t.Errorf("CanonicalBlock's number of fields changed after CBOR: %d to %d",
				v1.NumField(), v2.NumField())
		}

		for i := 0; i < v1.NumField(); i++ {
			val1 := v1.Field(i)
			val2 := v2.Field(i)

			if val1.Type() != val2.Type() {
				t.Errorf("Type of value no %d differs: %T and %T",
					i, val1.Type(), val2.Type())
			}

			s1 := fmt.Sprintf("%v", val1)
			s2 := fmt.Sprintf("%v", val2)

			if s1 != s2 {
				t.Errorf("String representation of value no %d differs: %v and %v",
					i, s1, s2)
			}
		}
	}
}

func TestExtensionBlockTypes(t *testing.T) {
	tests := []struct {
		name      string
		block     CanonicalBlock
		blockType uint
		typeLike  reflect.Kind
	}{
		{"Payload", NewPayloadBlock(0, []byte("foobar")), 1, reflect.Slice},
		{"Previous Node", NewPreviousNodeBlock(23, 0, *DtnNone), 7, reflect.Slice},
		{"Bundle Age", NewBundleAgeBlock(23, 0, 42000), 8, reflect.Uint64},
		{"Hop Count", NewHopCountBlock(23, 0, HopCount{Limit: 42, Count: 23}), 9, reflect.Slice},
	}

	for _, test := range tests {
		if test.block.BlockType != test.blockType {
			t.Errorf("%s Block has wrong Block Type:  %d instead of %d",
				test.name, test.block.BlockType, test.blockType)
		}

		var b []byte = make([]byte, 0, 64)
		var h codec.Handle = new(codec.CborHandle)
		var enc *codec.Encoder = codec.NewEncoderBytes(&b, h)

		err := enc.Encode(test.block)
		if err != nil {
			t.Errorf("CBOR encoding fo %s Block failed: %v", test.name, err)
		}

		var decGeneric interface{}
		err = codec.NewDecoderBytes(b, h).Decode(&decGeneric)
		if err != nil {
			t.Errorf("CBOR decoding of %s Block failed: %v", test.name, err)
		}

		if ty := reflect.TypeOf(decGeneric); ty.Kind() != reflect.Slice {
			t.Errorf("CBOR for %s Block has wrong type: %v instead of slice",
				test.name, ty.Kind())
		}

		var decArr []interface{} = decGeneric.([]interface{})
		var blockType uint = uint(decArr[0].(uint64))
		var blockData = decArr[4]

		if blockType != test.blockType {
			t.Errorf("%s Block has wrong Block Type after CBOR:  %d instead of %d",
				test.name, blockType, test.blockType)
		}

		if ty := reflect.TypeOf(blockData); ty.Kind() != test.typeLike {
			t.Errorf("%s Block's CBOR data has wrong type: %v instead of %v",
				test.name, ty.Kind(), test.typeLike)
		}
	}
}
