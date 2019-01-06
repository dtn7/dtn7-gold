package bpa

import "testing"

func TestCRCBackAndForth(t *testing.T) {
	pbCRCNo := setupPrimaryBlock()

	pbCRC16 := pbCRCNo
	pbCRC16.CRCType = CRC16

	pbCRC32 := pbCRCNo
	pbCRC32.CRCType = CRC32

	cbCRCNo := CanonicalBlock{1, 0, 0, CRCNo, []byte("hello world"), nil}

	cbCRC16 := CanonicalBlock{1, 0, 0, CRC16, []byte("hello world"), nil}

	cbCRC32 := CanonicalBlock{1, 0, 0, CRC32, []byte("hello world"), nil}

	tests := []struct {
		blck    block
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
		if test.blck.GetCRCType() != test.crcType {
			t.Errorf("Wrong CRC Type: %d instead of %d for %v",
				test.blck.GetCRCType(), test.crcType, test.blck)
		}

		test.blck.CalculateCRC()
		if !checkCRC(test.blck) {
			t.Errorf("Setting and checking CRC failed for %v", test.blck)
		}

		test.blck.resetCRC()
		if test.blck.HasCRC() && checkCRC(test.blck) {
			t.Errorf("CRC check succeeded after resetting for %v", test.blck)
		}
	}
}
