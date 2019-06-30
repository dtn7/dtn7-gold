package bundle

import "testing"

func TestExtensionBlockManager(t *testing.T) {
	var ebm = NewExtensionBlockManager()

	payloadBlock := NewPayloadBlock(nil)
	if err := ebm.Register(payloadBlock); err != nil {
		t.Fatal(err)
	}
	if err := ebm.Register(payloadBlock); err == nil {
		t.Fatal("Registering the PayloadBlock twice did not errored")
	}

	extBlock, extBlockErr := ebm.CreateBlock(payloadBlock.BlockTypeCode())
	if extBlockErr != nil {
		t.Fatal(extBlockErr)
	}

	if extBlock.BlockTypeCode() != payloadBlock.BlockTypeCode() {
		t.Fatalf("Block type code differs: %d != %d",
			extBlock.BlockTypeCode(), payloadBlock.BlockTypeCode())
	}

	if _, err := ebm.CreateBlock(9001); err == nil {
		t.Fatal("CreateBlock for an unknown number did not result in an errored")
	}

	ebm.Unregister(payloadBlock)
	if _, err := ebm.CreateBlock(payloadBlock.BlockTypeCode()); err == nil {
		t.Fatal("CreateBlock for an unregistered number did not result in an error")
	}
}

func TestExtensionBlockManagerSingleton(t *testing.T) {
	var ebm = GetExtensionBlockManager()

	tests := []uint64{
		ExtBlockTypePayloadBlock,
		ExtBlockTypePreviousNodeBlock,
		ExtBlockTypeBundleAgeBlock,
		ExtBlockTypeHopCountBlock}

	for _, test := range tests {
		if _, err := ebm.CreateBlock(test); err != nil {
			t.Fatalf("CreateBlock failed for %d", test)
		}
	}
}
