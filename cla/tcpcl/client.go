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

type ClientState int

const (
	Contact        ClientState = iota
	Initialization ClientState = iota
	Established    ClientState = iota
	Termination    ClientState = iota
)

func (cs ClientState) String() string {
	switch cs {
	case Contact:
		return "contact"
	case Initialization:
		return "initialization"
	case Established:
		return "established"
	case Termination:
		return "termination"
	default:
		return "INVALID"
	}
}

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

	// Initialization state fields:
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

		case Initialization:
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

// handleContact manges the contact stage with the Contact Header exchange.
func (client *TCPCLClient) handleContact() error {
	var logger = log.WithFields(log.Fields{
		"session": client,
		"state":   "contact",
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

func (client *TCPCLClient) handleSessInit() error {
	var logger = log.WithFields(log.Fields{
		"session": client,
		"state":   "initialization",
	})

	// XXX
	const (
		keepalive   = 10
		segmentMru  = 0xFFFFFFFFFFFFFFFF
		transferMru = 0xFFFFFFFFFFFFFFFF
	)

	switch {
	case client.active && !client.initSent, !client.active && !client.initSent && client.initRecv:
		client.sessInitSent = NewSessionInitMessage(keepalive, segmentMru, transferMru, client.endpointID.String())
		if err := client.sessInitSent.Marshal(client.rw); err != nil {
			return err
		} else if err := client.rw.Flush(); err != nil {
			return err
		} else {
			client.initSent = true
			logger.WithField("msg", client.sessInitSent).Debug("Sent SESS_INIT message")
		}

	case !client.active && !client.initRecv, client.active && client.initSent && !client.initRecv:
		if err := client.sessInitRecv.Unmarshal(client.rw); err != nil {
			return err
		} else {
			client.initRecv = true
			logger.WithField("msg", client.sessInitRecv).Debug("Received SESS_INIT message")
		}

	case client.initSent && client.initRecv:
		if eid, err := bundle.NewEndpointID(client.sessInitRecv.Eid); err != nil {
			return err
		} else {
			client.peerEndpointID = eid
		}

		client.keepalive = client.sessInitSent.KeepaliveInterval
		if client.sessInitRecv.KeepaliveInterval < client.keepalive {
			client.keepalive = client.sessInitRecv.KeepaliveInterval
		}
		client.segmentMru = client.sessInitSent.SegmentMru
		if client.sessInitRecv.SegmentMru < client.segmentMru {
			client.segmentMru = client.sessInitRecv.SegmentMru
		}
		client.transferMru = client.sessInitSent.TransferMru
		if client.sessInitRecv.TransferMru < client.transferMru {
			client.transferMru = client.sessInitRecv.TransferMru
		}

		logger.WithFields(log.Fields{
			"endpoint ID":  client.peerEndpointID,
			"keepalive":    client.keepalive,
			"segment MRU":  client.segmentMru,
			"transfer MRU": client.transferMru,
		}).Debug("Exchanged SESS_INIT messages")
		client.state += 1
	}

	return nil
}

func (client *TCPCLClient) keepaliveHandler() {
	var logger = log.WithField("session", client)

	var keepaliveTicker = time.NewTicker(time.Duration(client.keepalive) * time.Second)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-keepaliveTicker.C:
			var keepaliveMsg = NewKeepaliveMessage()
			if err := keepaliveMsg.Marshal(client.rw); err != nil {
				logger.WithError(err).Warn("Sending KEEPALIVE errored")
			} else if err := client.rw.Flush(); err != nil {
				logger.WithError(err).Warn("Flushing KEEPALIVE errored")
			} else {
				log.WithField("msg", keepaliveMsg).Debug("Sent KEEPALIVE message")
			}

		case <-client.keepaliveStopSyn:
			close(client.keepaliveStopAck)
			return
		}
	}
}

func (client *TCPCLClient) handleEstablished() error {
	var logger = log.WithField("session", client)

	if !client.keepaliveStarted {
		go client.keepaliveHandler()
		client.keepaliveStarted = true
	}

	nextMsg, nextMsgErr := client.rw.ReadByte()
	if nextMsgErr != nil {
		return nextMsgErr
	} else if err := client.rw.UnreadByte(); err != nil {
		return err
	}

	switch nextMsg {
	case KEEPALIVE:
		var keepaliveMsg KeepaliveMessage
		if err := keepaliveMsg.Unmarshal(client.rw); err != nil {
			return err
		} else {
			logger.WithField("msg", keepaliveMsg).Debug("Received KEEPALIVE message")
		}

	default:
		logger.WithField("magic", nextMsg).Debug("Received unsupported magic")
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
