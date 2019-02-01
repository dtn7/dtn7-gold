package stcp

import (
	"fmt"
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
	endpointID    bundle.EndpointID

	stopSyn chan struct{}
	stopAck chan struct{}
}

// NewSTCPServer creates a new STCPServer for the given listen address.
func NewSTCPServer(listenAddress string, endpointID bundle.EndpointID) *STCPServer {
	var serv = &STCPServer{
		listenAddress: listenAddress,
		reportChan:    make(chan bundle.Bundle),
		endpointID:    endpointID,
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
				log.Printf("Reception of STCP data unit failed: %v", err)
			}
		} else if err != io.EOF {
			log.Printf("Reception of STCP data unit failed: %v", err)
		}
	}
}

// Channel returns a channel of received bundles.
func (serv STCPServer) Channel() <-chan bundle.Bundle {
	return serv.reportChan
}

// Close shuts this STCPServer down.
func (serv *STCPServer) Close() {
	close(serv.stopSyn)
	<-serv.stopAck
}

// GetEndpointID returns the endpoint ID assigned to this CLA.
func (serv STCPServer) GetEndpointID() bundle.EndpointID {
	return serv.endpointID
}

func (serv STCPServer) String() string {
	return fmt.Sprintf("stcp://%s", serv.listenAddress)
}
