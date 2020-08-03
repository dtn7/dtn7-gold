// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package mtcp

import (
	"fmt"
	"net"
	"time"
)

func setKeepAlive(conn net.Conn) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("Expected TCPConn, not %T", conn)
	}

	if err := tcpConn.SetKeepAlive(true); err != nil {
		return err
	}
	if err := tcpConn.SetKeepAlivePeriod(time.Second); err != nil {
		return err
	}

	return nil
}
