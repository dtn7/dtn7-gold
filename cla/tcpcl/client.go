package tcpcl

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type ClientState int

const (
	Contact        ClientState = iota
	Initialization ClientState = iota
	Established    ClientState = iota
	Termination    ClientState = iota
)

type TCPCLClient struct {
	address string
	conn    net.Conn
	rw      *bufio.ReadWriter

	active bool
	state  ClientState

	// Contact state fields:
	contactSent bool
	contactRecv bool
	chSent      ContactHeader
	chRecv      ContactHeader

	// Termination state fields:
}

func NewTCPCLClient(conn net.Conn) *TCPCLClient {
	return &TCPCLClient{
		conn:   conn,
		active: false,
	}
}

func Dial(address string) *TCPCLClient {
	return &TCPCLClient{
		address: address,
		active:  true,
	}
}

func (client *TCPCLClient) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "TCPCL(")
	fmt.Fprintf(&b, "peer=%v,", client.conn.RemoteAddr())
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
	var logger = log.WithField("session", client)

	for {
		switch client.state {
		case Contact:
			if err := client.handleContact(); err != nil {
				logger.WithField("state", "contact").WithError(err).Warn(
					"Error occured during contact state")

				client.terminate(TerminationContactFailure)
				return
			}
		}
	}
}

// handleContact manges the contact stage with the Contact Header exchange.
func (client *TCPCLClient) handleContact() error {
	var logger = log.WithFields(log.Fields{
		"session":     client,
		"active peer": client.active,
		"state":       "contact",
	})

	switch {
	case client.active && !client.contactSent, !client.active && !client.contactSent && client.contactRecv:
		client.chSent = NewContactHeader(0)
		if err := client.chSent.Marshal(client.rw); err != nil {
			return err
		} else if err := client.rw.Flush(); err != nil {
			return err
		} else {
			client.contactSent = true
			logger.WithField("msg", client.chSent).Debug("Sent Contact Header")
		}

	case !client.active && !client.contactRecv, client.active && client.contactSent && !client.contactRecv:
		if err := client.chRecv.Unmarshal(client.rw); err != nil {
			return err
		} else {
			client.contactRecv = true
			logger.WithField("msg", client.chRecv).Debug("Received Contact Header")
		}

	case client.contactSent && client.contactRecv:
		// TODO: check contact header flags
		logger.Debug("Exchanged Contact Headers")
		client.state += 1
	}

	return nil
}

// terminate sends a SESS_TERM message to its peer and closes the session afterwards.
func (client *TCPCLClient) terminate(code SessionTerminationCode) {
	var logger = log.WithField("session", client)

	var sessTerm = NewSessionTerminationMessage(0, code)
	if err := sessTerm.Marshal(client.rw); err != nil {
		logger.WithError(err).Warn("Failed to send session termination message")
	} else if err := client.rw.Flush(); err != nil {
		logger.WithError(err).Warn("Failed to flush buffer")
	} else if err := client.conn.Close(); err != nil {
		logger.WithError(err).Warn("Failed to close TCP connection")
	} else {
		logger.Info("Terminated session")
	}
}
