package tcpcl

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

// sessTermErr will be returned from a state handler iff a SESS_TERM was received.
var sessTermErr = errors.New("SESS_TERM received")

type TCPCLClient struct {
	address        string
	started        bool
	permanent      bool
	endpointID     bundle.EndpointID
	peerEndpointID bundle.EndpointID

	conn net.Conn

	msgsOut chan Message
	msgsIn  chan Message

	handleMetaStop     chan struct{}
	handleMetaStopAck  chan struct{}
	handlerConnInStop  chan struct{}
	handlerConnOutStop chan struct{}
	handlerStateStop   chan struct{}

	active bool
	state  *ClientState

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

	transferOutMutex sync.Mutex
	transferOutId    uint64
	transferOutSend  chan Message
	transferOutAck   chan Message

	transferIn *IncomingTransfer

	reportChan chan cla.ConvergenceStatus
}

func NewTCPCLClient(conn net.Conn, endpointID bundle.EndpointID) *TCPCLClient {
	return &TCPCLClient{
		address:         conn.RemoteAddr().String(),
		conn:            conn,
		active:          false,
		state:           new(ClientState),
		msgsOut:         make(chan Message, 100),
		msgsIn:          make(chan Message, 100),
		transferOutSend: make(chan Message),
		transferOutAck:  make(chan Message),
		endpointID:      endpointID,
	}
}

func Dial(address string, endpointID bundle.EndpointID, permanent bool) *TCPCLClient {
	return &TCPCLClient{
		address:         address,
		permanent:       permanent,
		active:          true,
		state:           new(ClientState),
		msgsOut:         make(chan Message, 100),
		msgsIn:          make(chan Message, 100),
		transferOutSend: make(chan Message),
		transferOutAck:  make(chan Message),
		endpointID:      endpointID,
	}
}

func (client *TCPCLClient) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "TCPCL(")
	if client.conn != nil {
		fmt.Fprintf(&b, "peer=%v, ", client.conn.RemoteAddr())
	} else {
		fmt.Fprintf(&b, "peer=NONE, ")
	}
	fmt.Fprintf(&b, "active peer=%t", client.active)
	fmt.Fprintf(&b, ")")

	return b.String()
}

// log prepares a new log entry with predefined session data.
func (client *TCPCLClient) log() *log.Entry {
	return log.WithFields(log.Fields{
		"session": client,
		"state":   client.state,
	})
}

func (client *TCPCLClient) Start() (err error, retry bool) {
	if client.started {
		if client.active {
			client.conn = nil
		} else {
			err = fmt.Errorf("Passive client cannot be restarted")
			retry = false
			return
		}
	}

	client.started = true

	if client.conn == nil {
		if conn, connErr := net.DialTimeout("tcp", client.address, time.Second); connErr != nil {
			err = connErr
			retry = true
			return
		} else {
			client.conn = conn
			client.address = conn.RemoteAddr().String()
		}
	}

	client.log().Info("Starting client")

	client.handleMetaStop = make(chan struct{}, 10)
	client.handleMetaStopAck = make(chan struct{}, 2)
	client.handlerConnInStop = make(chan struct{}, 2)
	client.handlerConnOutStop = make(chan struct{}, 2)
	client.handlerStateStop = make(chan struct{}, 2)

	client.reportChan = make(chan cla.ConvergenceStatus, 100)

	go client.handleMeta()
	go client.handleConnIn()
	go client.handleConnOut()
	go client.handleState()

	return
}

func (client *TCPCLClient) Close() {
	client.handleMetaStop <- struct{}{}
	<-client.handleMetaStopAck
}

func (client *TCPCLClient) Channel() chan cla.ConvergenceStatus {
	return client.reportChan
}

func (client *TCPCLClient) Address() string {
	return client.address
}

func (client *TCPCLClient) IsPermanent() bool {
	return client.permanent
}

func (client *TCPCLClient) GetEndpointID() bundle.EndpointID {
	return client.endpointID
}

func (client *TCPCLClient) GetPeerEndpointID() bundle.EndpointID {
	return client.peerEndpointID
}
