package tcpcl

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

type TCPCLClient struct {
	address        string
	endpointID     bundle.EndpointID
	peerEndpointID bundle.EndpointID

	conn net.Conn
	rw   *bufio.ReadWriter

	active bool
	state  ClientState

	// Contact state fields:
	contactSent bool
	contactRecv bool
	chSent      ContactHeader
	chRecv      ContactHeader

	// Init state fields:
	initSent     bool
	initRecv     bool
	sessInitSent SessionInitMessage
	sessInitRecv SessionInitMessage

	keepalive   uint16
	segmentMru  uint64
	transferMru uint64

	// Established state fields:
	keepaliveStarted bool
	keepaliveStopSyn chan struct{}
	keepaliveStopAck chan struct{}
}

func NewTCPCLClient(conn net.Conn, endpointID bundle.EndpointID) *TCPCLClient {
	return &TCPCLClient{
		conn:             conn,
		active:           false,
		endpointID:       endpointID,
		keepaliveStopSyn: make(chan struct{}),
		keepaliveStopAck: make(chan struct{}),
	}
}

func Dial(address string, endpointID bundle.EndpointID) *TCPCLClient {
	return &TCPCLClient{
		address:          address,
		active:           true,
		endpointID:       endpointID,
		keepaliveStopSyn: make(chan struct{}),
		keepaliveStopAck: make(chan struct{}),
	}
}

func (client *TCPCLClient) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "TCPCL(")
	fmt.Fprintf(&b, "peer=%v, ", client.conn.RemoteAddr())
	fmt.Fprintf(&b, "active peer=%t", client.active)
	fmt.Fprintf(&b, ")")

	return b.String()
}

func (client *TCPCLClient) Start() (err error, retry bool) {
	if client.conn == nil {
		if conn, connErr := net.DialTimeout("tcp", client.address, time.Second); connErr != nil {
			err = connErr
			return
		} else {
			client.conn = conn
		}
	}
	client.rw = bufio.NewReadWriter(bufio.NewReader(client.conn), bufio.NewWriter(client.conn))

	log.Info("Starting client")

	go client.handler()
	return
}

func (client *TCPCLClient) handler() {
	var logger = log.WithFields(log.Fields{
		"session": client,
		"state":   client.state,
	})

	for {
		switch client.state {
		case Contact:
			if err := client.handleContact(); err != nil {
				logger.WithError(err).Warn("Error occured during contact header exchange")

				client.terminate(TerminationContactFailure)
				return
			}

		case Init:
			if err := client.handleSessInit(); err != nil {
				logger.WithError(err).Warn("Error occured during session initialization")

				client.terminate(TerminationUnknown)
				return
			}

		case Established:
			if err := client.handleEstablished(); err != nil {
				logger.WithError(err).Warn("Error occured during established session")

				// TODO
				return
			}
		}
	}
}
