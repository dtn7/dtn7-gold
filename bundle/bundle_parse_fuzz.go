// +build gofuzz

package bundle

import "bytes"

func Fuzz(data []byte) int {
	if len(data) > 256 {
		return -1
	}

	buff := bytes.NewBuffer(data)
	b, err := ParseBundle(buff)
	if err != nil {
		return 0
	}

	if err = b.MarshalCbor(buff); err != nil {
		panic(err)
	}

	return 1
}
