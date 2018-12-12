package bpa

import "testing"

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
