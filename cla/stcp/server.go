package stcp

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

// STCPServer is an implementation of a Simple TCP Convergence-Layer server
// which accpets bundles from multiple connections and forwards them to the
// given channel.
type STCPServer struct {
	listenAddress string
	reportChan    chan bundle.Bundle
	stopSyn       chan struct{}
	stopAck       chan struct{}
}

// NewSTCPServer creates a new STCPServer for the given listen address and
// reporting channel.
func NewSTCPServer(listenAddress string, reportChan chan bundle.Bundle) *STCPServer {
	var serv = &STCPServer{
		listenAddress: listenAddress,
		reportChan:    reportChan,
		stopSyn:       make(chan struct{}),
		stopAck:       make(chan struct{}),
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", serv.listenAddress)
	if err != nil {
		panic(err)
	}

	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		panic(err)
	}

	go func(ln *net.TCPListener) {
		for {
			select {
			case <-serv.stopSyn:
				ln.Close()
				close(serv.stopAck)

				return

			default:
				ln.SetDeadline(time.Now().Add(50 * time.Millisecond))
				if conn, err := ln.Accept(); err == nil {
					go serv.handleSender(conn)
				}
			}
		}
	}(ln)

	return serv
}

func (serv STCPServer) handleSender(conn net.Conn) {
	defer func() {
		conn.Close()

		if r := recover(); r != nil {
			log.Printf("STCPServer.handleSender: %v", r)
		}
	}()

	for {
		var du = new(DataUnit)
		var dec = codec.NewDecoder(conn, new(codec.CborHandle))

		if err := dec.Decode(du); err == nil {
			if bndl, err := du.toBundle(); err == nil {
				serv.reportChan <- bndl
			} else {
				log.Panicf("Reception of STCP data unit failed: %v", err)
			}
		} else if err != io.EOF {
			log.Panicf("Reception of STCP data unit failed: %v", err)
		}
	}
}

// Close shuts this STCPServer down.
func (serv *STCPServer) Close() {
	close(serv.stopSyn)
	<-serv.stopAck
}
