package tcpcl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

// sessTermErr will be returned from a state handler iff a SESS_TERM was received.
var sessTermErr = errors.New("SESS_TERM received")

type TCPCLClient struct {
	address        string
	endpointID     bundle.EndpointID
	peerEndpointID bundle.EndpointID

	conn net.Conn

	msgsOut chan Message
	msgsIn  chan Message

	closed   bool
	closedWg sync.WaitGroup

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
	keepaliveLast    time.Time
	keepaliveTicker  *time.Ticker

	transferIdOut uint64
}

func NewTCPCLClient(conn net.Conn, endpointID bundle.EndpointID) *TCPCLClient {
	return &TCPCLClient{
		conn:       conn,
		active:     false,
		msgsOut:    make(chan Message, 100),
		msgsIn:     make(chan Message),
		endpointID: endpointID,
	}
}

func Dial(address string, endpointID bundle.EndpointID) *TCPCLClient {
	return &TCPCLClient{
		address:    address,
		active:     true,
		msgsOut:    make(chan Message, 100),
		msgsIn:     make(chan Message),
		endpointID: endpointID,
	}
}

func (client TCPCLClient) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "TCPCL(")
	fmt.Fprintf(&b, "peer=%v, ", client.conn.RemoteAddr())
	fmt.Fprintf(&b, "active peer=%t", client.active)
	fmt.Fprintf(&b, ")")

	return b.String()
}

// log prepares a new log entry with predefined session data.
func (client TCPCLClient) log() *log.Entry {
	return log.WithFields(log.Fields{
		"session": client.String(),
		"state":   client.state,
	})
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

	log.Info("Starting client")

	client.closedWg.Add(2)
	go client.handleConnection()
	go client.handleState()

	return
}

func (client *TCPCLClient) handleConnection() {
	defer func() {
		client.log().Debug("Leaving connection handler function")
		client.state.Terminate()

		client.closed = true
		client.closedWg.Done()
	}()

	var rw = bufio.NewReadWriter(bufio.NewReader(client.conn), bufio.NewWriter(client.conn))

	for {
		select {
		case msg := <-client.msgsOut:
			if err := msg.Marshal(rw); err != nil {
				client.log().WithError(err).WithField("msg", msg).Error("Sending message errored")
				return
			} else if err := rw.Flush(); err != nil {
				client.log().WithError(err).WithField("msg", msg).Error("Flushing errored")
				return
			} else {
				client.log().WithField("msg", msg).Debug("Sent message")
			}

			if _, ok := msg.(*SessionTerminationMessage); ok {
				client.log().WithField("msg", msg).Debug("Closing connection after sending SESS_TERM")

				if err := client.conn.Close(); err != nil {
					client.log().WithError(err).Warn("Failed to close TCP connection")
				}
				return
			}

		default:
			if err := client.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				client.log().WithError(err).Error("Setting read deadline errored")
				return
			}

			if msg, err := ReadMessage(rw); err == nil {
				client.log().WithField("msg", msg).Debug("Received message")
				client.msgsIn <- msg
			} else if err == io.EOF {
				client.log().Info("Read EOF, closing down.")
				return
			} else if netErr, ok := err.(net.Error); ok && !netErr.Timeout() {
				client.log().WithError(netErr).Error("Network error occured")
				return
			} else if !ok {
				client.log().WithError(err).Error("Parsing next message errored")
				return
			}
		}
	}
}

func (client *TCPCLClient) handleState() {
	defer func() {
		client.log().Debug("Leaving state handler function")

		client.closed = true
		client.closedWg.Done()
	}()

	for {
		switch client.state {
		case Contact, Init, Established:
			var stateHandler func() error

			switch client.state {
			case Contact:
				stateHandler = client.handleContact
			case Init:
				stateHandler = client.handleSessInit
			case Established:
				stateHandler = client.handleEstablished
			}

			if err := stateHandler(); err != nil {
				if err == sessTermErr {
					client.log().Info("Received SESS_TERM, switching to Termination state")
				} else {
					client.log().WithError(err).Warn("State handler errored")
				}

				client.state.Terminate()
				goto terminationCase
			}
			break

		terminationCase:
			fallthrough

		case Termination:
			client.log().Info("Entering Termination state")

			var sessTerm = NewSessionTerminationMessage(0, TerminationUnknown)
			client.msgsOut <- &sessTerm

			return
		}

		if client.closed {
			return
		}
	}
}

func (client *TCPCLClient) Close() {
	client.state.Terminate()
	client.closedWg.Wait()
}
