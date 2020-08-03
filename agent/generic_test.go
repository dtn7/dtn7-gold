// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"net"
	"testing"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
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

// isAddrReachable checks if a TCP address - like localhost:2342 - is reachable.
func isAddrReachable(addr string) (open bool) {
	if conn, err := net.DialTimeout("tcp", addr, time.Second); err != nil {
		open = false
	} else {
		open = true
		_ = conn.Close()
	}
	return
}

// createBundle from src to dst for testing purpose.
func createBundle(src, dst string, t *testing.T) bundle.Bundle {
	b, err := bundle.Builder().
		Source(src).
		Destination(dst).
		CreationTimestampNow().
		Lifetime("24h").
		PayloadBlock([]byte("hello world")).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	return b
}
