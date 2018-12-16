package bpa

import "testing"

func TestCRCBackAndForth(t *testing.T) {
	pbCRCNo := setupPrimaryBlock()

	pbCRC16 := pbCRCNo
	pbCRC16.CRCType = CRC16

	pbCRC32 := pbCRCNo
	pbCRC32.CRCType = CRC32

	cbCRCNo := CanonicalBlock{1, 0, 0, CRCNo, []byte("hello world"), 0}

	cbCRC16 := CanonicalBlock{1, 0, 0, CRC16, []byte("hello world"), 0}

	cbCRC32 := CanonicalBlock{1, 0, 0, CRC32, []byte("hello world"), 0}

	tests := []struct {
		block   Block
		crcType CRCType
	}{
		{&pbCRCNo, CRCNo},
		{&pbCRC16, CRC16},
		{&pbCRC32, CRC32},
		{&cbCRCNo, CRCNo},
		{&cbCRC16, CRC16},
		{&cbCRC32, CRC32},
	}

	for _, test := range tests {
		if test.block.GetCRCType() != test.crcType {
			t.Errorf("Wrong CRC Type: %d instead of %d for %v",
				test.block.GetCRCType(), test.crcType, test.block)
		}

		SetCRC(test.block)
		if !CheckCRC(test.block) {
			t.Errorf("Setting and checking CRC failed for %v", test.block)
		}

		test.block.ResetCRC()
		if test.block.HasCRC() && CheckCRC(test.block) {
			t.Errorf("CRC check succeeded after resetting for %v", test.block)
		}
	}
}
