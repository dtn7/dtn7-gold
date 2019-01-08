package stcp

import (
	"fmt"
	"io/ioutil"
	"net"
)

func handleSender(conn net.Conn) {
	defer conn.Close()

	for {
		data, err := ioutil.ReadAll(conn)
		if err != nil {
			panic(err)
		}

		if len(data) == 0 {
			continue
		}

		s, err := newDataUnitFromCbor(data)
		if err != nil {
			panic(err)
		}

		b, err := s.toBundle()
		if err != nil {
			panic(err)
		}

		payload, err := b.PayloadBlock()
		if err != nil {
			panic(err)
		}

		ts := b.PrimaryBlock.CreationTimestamp.DtnTime().Time()

		fmt.Printf("New Bundle: %v, %s\n", ts, payload.Data)
	}
}

// LaunchReceiver starts a STCP-server on the given port and prints received
// bundles to the stdout. This is a proof of concept.
func LaunchReceiver(port uint16) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go handleSender(conn)
	}
}
