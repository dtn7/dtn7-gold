// SPDX-FileCopyrightText: 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !linux
// +build !linux

package mtcp

import (
	"net"
	"time"
)

// This file implements a Dialer for operating systems next to Linux. The other
// file additionally sets specific socket options for a better detection of
// connection losses.

// dial a new TCP connection with a configured timeout and keepalive.
func dial(address string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   time.Second,
		KeepAlive: 5 * time.Second,
	}
	return dialer.Dial("tcp", address)
}
