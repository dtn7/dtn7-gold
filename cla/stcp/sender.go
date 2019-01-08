package stcp

import (
	"bytes"
	"net"

	"github.com/geistesk/dtn7/bundle"
)

// SendPoC establishes a quick-and-dirty connection to a STCP receiver and
// transmitts the given bundle. This is a proof of concept.
func SendPoC(server string, bndl bundle.Bundle) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	buff := bytes.NewBuffer(newDataUnit(bndl).toCbor())
	buff.WriteTo(conn)
}
