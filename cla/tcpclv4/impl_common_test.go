// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpclv4

import (
	"net"
	"testing"
)

// randomPort returns a random open TCP port.
func randomPort(t *testing.T) (port int) {
	if addr, err := net.ResolveTCPAddr("tcp", "localhost:0"); err != nil {
		t.Fatal(err)
	} else if l, err := net.ListenTCP("tcp", addr); err != nil {
		t.Fatal(err)
	} else {
		port = l.Addr().(*net.TCPAddr).Port
		_ = l.Close()
	}
	return
}
