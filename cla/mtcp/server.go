package mtcp

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// MTCPServer is an implementation of a Minimal TCP Convergence-Layer server
// which accepts bundles from multiple connections and forwards them to the
// given channel.
type MTCPServer struct {
	listenAddress string
	reportChan    chan cla.RecBundle
	endpointID    bundle.EndpointID
	permanent     bool

	stopSyn chan struct{}
	stopAck chan struct{}
}

// NewMTCPServer creates a new MTCPServer for the given listen address. The
// permanent flag indicates if this MTCPServer should never be removed from
// the core.
func NewMTCPServer(listenAddress string, endpointID bundle.EndpointID, permanent bool) *MTCPServer {
	return &MTCPServer{
		listenAddress: listenAddress,
		reportChan:    make(chan cla.RecBundle),
		endpointID:    endpointID,
		permanent:     permanent,
		stopSyn:       make(chan struct{}),
		stopAck:       make(chan struct{}),
	}
}

// Start starts this MTCPServer and might return an error and a boolean
// indicating if another Start should be tried later.
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
				ln.Close()
				close(serv.reportChan)
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

	return nil, true
}

func (serv *MTCPServer) handleSender(conn net.Conn) {
	defer func() {
		conn.Close()

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

	for {
		var err error
		if du, err := ioutil.ReadAll(bufio.NewReader(conn)); err == nil {
			if len(du) == 0 {
				log.WithFields(log.Fields{
					"cla":  serv,
					"conn": conn,
				}).Debug("MTCP handleServer connection was closed")
				return
			}

			log.WithFields(log.Fields{
				"cla":  serv,
				"conn": conn,
			}).Debug("MTCP handleServer connection received a byte string")

			offset := (1 << (du[0] - 0x58)) + 1
			bndlData := du[offset:]

			if bndl, err := bundle.NewBundleFromCbor(&bndlData); err == nil {
				log.WithFields(log.Fields{
					"cla":  serv,
					"conn": conn,
				}).Debug("MTCP handleServer connection received a bundle")

				serv.reportChan <- cla.NewRecBundle(&bndl, serv.endpointID)
			}
		}

		if err != nil {
			log.WithFields(log.Fields{
				"cla":   serv,
				"conn":  conn,
				"error": err,
			}).Warn("Reception of MTCP data unit failed, closing conn's handler")
			return
		}
	}
}

// Channel returns a channel of received bundles.
func (serv *MTCPServer) Channel() chan cla.RecBundle {
	return serv.reportChan
}

// Close shuts this MTCPServer down.
func (serv *MTCPServer) Close() {
	close(serv.stopSyn)
	<-serv.stopAck
}

// GetEndpointID returns the endpoint ID assigned to this CLA.
func (serv MTCPServer) GetEndpointID() bundle.EndpointID {
	return serv.endpointID
}

// Address should return a unique address string to both identify this
// ConvergenceReceiver and ensure it will not opened twice.
func (serv MTCPServer) Address() string {
	return fmt.Sprintf("mtcp://%s", serv.listenAddress)
}

// IsPermanent returns true, if this CLA should not be removed after failures.
func (serv MTCPServer) IsPermanent() bool {
	return serv.permanent
}

func (serv MTCPServer) String() string {
	return serv.Address()
}
