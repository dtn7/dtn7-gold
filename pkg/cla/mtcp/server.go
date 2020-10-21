// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package mtcp

import (
	"bufio"
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/cboring"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
)

// MTCPServer is an implementation of a Minimal TCP Convergence-Layer server
// which accepts bundles from multiple connections and forwards them to the
// given channel. This struct implements a ConvergenceReceiver.
type MTCPServer struct {
	listenAddress string
	reportChan    chan cla.ConvergenceStatus
	endpointID    bpv7.EndpointID
	permanent     bool

	stopSyn chan struct{}
	stopAck chan struct{}
}

// NewMTCPServer creates a new MTCPServer for the given listen address. The
// permanent flag indicates if this MTCPServer should never be removed from
// the core.
func NewMTCPServer(listenAddress string, endpointID bpv7.EndpointID, permanent bool) *MTCPServer {
	return &MTCPServer{
		listenAddress: listenAddress,
		reportChan:    make(chan cla.ConvergenceStatus),
		endpointID:    endpointID,
		permanent:     permanent,
		stopSyn:       make(chan struct{}),
		stopAck:       make(chan struct{}),
	}
}

func (serv *MTCPServer) Start() (error, bool) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", serv.listenAddress)
	if err != nil {
		return err, false
	}

	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err, true
	}

	go func(ln *net.TCPListener) {
		for {
			select {
			case <-serv.stopSyn:
				_ = ln.Close()
				close(serv.reportChan)
				close(serv.stopAck)

				return

			default:
				if err := ln.SetDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
					log.WithFields(log.Fields{
						"cla":   serv,
						"error": err,
					}).Warn("MTCPServer failed to set deadline on TCP socket")

					_ = serv.Close()
				} else if conn, err := ln.Accept(); err == nil {
					go serv.handleSender(conn)
				}
			}
		}
	}(ln)

	return nil, true
}

func (serv *MTCPServer) handleSender(conn net.Conn) {
	defer func() {
		_ = conn.Close()

		if r := recover(); r != nil {
			log.WithFields(log.Fields{
				"cla":   serv,
				"conn":  conn,
				"error": r,
			}).Warn("MTCPServer's sender failed")
		}
	}()

	log.WithFields(log.Fields{
		"cla":  serv,
		"conn": conn,
	}).Debug("MTCP handleServer connection was established")

	connReader := bufio.NewReader(conn)
	for {
		if n, err := cboring.ReadByteStringLen(connReader); err != nil {
			log.WithFields(log.Fields{
				"cla":   serv,
				"conn":  conn,
				"error": err,
			}).Warn("MTCP handleServer connection failed to read byte string len")

			// There is no use in sending an PeerDisappeared Message at this point,
			// because a MTCPServer might hold multiple clients. Furthermore, there
			// is no linkage between unknown connections and Endpoint IDs.

			return
		} else if n == 0 {
			continue
		}

		bndl := new(bpv7.Bundle)
		if err := cboring.Unmarshal(bndl, connReader); err != nil {
			log.WithFields(log.Fields{
				"cla":   serv,
				"conn":  conn,
				"error": err,
			}).Warn("MTCP handleServer connection failed to read bundle")

			return
		} else {
			log.WithFields(log.Fields{
				"cla":  serv,
				"conn": conn,
			}).Debug("MTCP handleServer connection received a bundle")

			serv.reportChan <- cla.NewConvergenceReceivedBundle(serv, serv.endpointID, bndl)
		}
	}
}

func (serv *MTCPServer) Channel() chan cla.ConvergenceStatus {
	return serv.reportChan
}

func (serv *MTCPServer) Close() error {
	close(serv.stopSyn)
	<-serv.stopAck

	return nil
}

func (serv MTCPServer) GetEndpointID() bpv7.EndpointID {
	return serv.endpointID
}

func (serv MTCPServer) Address() string {
	return fmt.Sprintf("mtcp://%s", serv.listenAddress)
}

func (serv MTCPServer) IsPermanent() bool {
	return serv.permanent
}

func (serv MTCPServer) String() string {
	return serv.Address()
}
