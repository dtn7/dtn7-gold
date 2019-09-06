package tcpcl

import (
	"bufio"
	"net"
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
	active  bool
	state   ClientState
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

	go client.handler()
	return
}

func (client *TCPCLClient) handler() {
	rw := bufio.NewReadWriter(bufio.NewReader(client.conn), bufio.NewWriter(client.conn))

	for {
		switch client.state {
		case Contact:
			if client.active {
				ch := NewContactHeader(0)
				if err := ch.Marshal(rw); err != nil {
					log.WithError(err).Error("Marshaling errored")
				} else if err := rw.Flush(); err != nil {
					log.WithError(err).Error("Flushing errored")
				} else {
					client.state += 1
				}
			} else {
				var ch ContactHeader
				if err := ch.Unmarshal(rw); err != nil {
					log.WithError(err).Error("Unmarshaling errored")
				} else {
					log.WithField("msg", ch).Info("Contact Header received")
					client.state += 1
				}
			}
		}
	}
}
