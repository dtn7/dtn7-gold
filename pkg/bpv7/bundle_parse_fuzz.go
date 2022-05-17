// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build gofuzz
// +build gofuzz

package bpv7

import "bytes"

func Fuzz(data []byte) int {
	// Make sure go-fuzz has the right start
	if len(data) > 0 && data[0] != 0x9f {
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
