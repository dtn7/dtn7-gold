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
