// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// +build !windows

package mtcp

import (
	"net"
	"time"

	"github.com/felixge/tcpkeepalive"
)

func setKeepAlive(conn net.Conn) error {
	return tcpkeepalive.SetKeepAlive(conn, time.Second, 1, 500*time.Millisecond)
}
