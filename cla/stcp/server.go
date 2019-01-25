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
// reporting channel. To activate the STCPServer, the Construct method needs
// to be called.
func NewSTCPServer(listenAddress string, reportChan chan bundle.Bundle) *STCPServer {
	return &STCPServer{
		listenAddress: listenAddress,
		reportChan:    reportChan,
		stopSyn:       make(chan struct{}),
		stopAck:       make(chan struct{}),
	}
}

func (serv STCPServer) handleSender(conn net.Conn) {
	defer func() {
		conn.Close()

		if r := recover(); r != nil {
			log.Printf("STCPServer.handleSender: %v", r)
		}
	}()

	for {
		var dataUnit = new(DataUnit)
		var dec = codec.NewDecoder(conn, new(codec.CborHandle))

		if err := dec.Decode(dataUnit); err == nil {
			if bndl, err := dataUnit.toBundle(); err == nil {
				serv.reportChan <- bndl
			} else {
				log.Panicf("Reception of STCP data unit failed: %v", err)
			}
		} else if err != io.EOF {
			log.Panicf("Reception of STCP data unit failed: %v", err)
		}
	}
}

// Construct starts this STCPServer.
func (serv *STCPServer) Construct() {
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
}

// Destruct shuts this STCPServer down.
func (serv *STCPServer) Destruct() {
	close(serv.stopSyn)
	<-serv.stopAck
}
