// SPDX-FileCopyrightText: 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build linux
// +build linux

package mtcp

import (
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Within this file, Linux-specific socket options are configured for the
// client's TCP connection. By doing so, a better transmission quality regarding
// abrupt connection losses is expected. This is especially useful in mobile
// scenarios where nodes might move out of range at any time.
//
// The socket options are based on the Linux tcp(7) manual page.
// <https://man7.org/linux/man-pages/man7/tcp.7.html>

// dialControl is the net.Dialer's Control function to set the socket options.
func dialControl(_, _ string, rawConn syscall.RawConn) (err error) {
	const (
		// dialTcpKeepCnt sets TCP_KEEPCNT, the maximum number of keepalive
		// probes to be sent before dropping the connection.
		dialTcpKeepCnt int = 1

		// dialTcpKeepIdle sets TCP_KEEPIDLE, the time (in seconds) the
		// connections needs to remain idle before keepalive probes being sent.
		dialTcpKeepIdle int = 5

		// dialTcpKeepIntvl sets TCP_KEEPINTVL, the time (in seconds) between
		// keepalive probes.
		dialTcpKeepIntvl int = 3

		// dialTcpUserTimeout sets TCP_USER_TIMEOUT, the maximum time (in
		// milliseconds) that transmitted data may remain unacknowledged before
		// the connection will forcibly be closed.
		dialTcpUserTimeout int = 2000
	)

	opts := map[int]int{
		unix.TCP_KEEPCNT:      dialTcpKeepCnt,
		unix.TCP_KEEPIDLE:     dialTcpKeepIdle,
		unix.TCP_KEEPINTVL:    dialTcpKeepIntvl,
		unix.TCP_USER_TIMEOUT: dialTcpUserTimeout,
	}

	err = rawConn.Control(func(fd uintptr) {
		for opt, value := range opts {
			err = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, opt, value)
			if err != nil {
				return
			}
		}
	})

	return
}

// dial a new TCP connection with socket options set.
func dial(address string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: time.Second,
		Control: dialControl,
	}
	return dialer.Dial("tcp", address)
}
