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
	tests := []struct {
		cb1 CanonicalBlock
		len int
	}{
		// No CRC
		{CanonicalBlock{1, 0, 0, CRCNo, "hello world", 0}, 5},
		// CRC
		{CanonicalBlock{1, 0, 0, CRC16, "hello world", 0}, 6},
	}

	for _, test := range tests {
		var b []byte = make([]byte, 0, 64)
		var h codec.Handle = new(codec.CborHandle)
		var enc *codec.Encoder = codec.NewEncoderBytes(&b, h)

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
			t.Errorf("CBOR decoding failed: %v", err)
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
