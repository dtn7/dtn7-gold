package stcp

import (
	"io/ioutil"
	"net"
	"time"

	"github.com/geistesk/dtn7/bundle"
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
			// TODO: handle error
		}
	}()

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

		serv.reportChan <- b
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
