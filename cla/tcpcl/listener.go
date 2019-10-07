package tcpcl

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

type TCPCLListener struct {
	listenAddress string
	endpointID    bundle.EndpointID
	manager       *cla.Manager
	clas          []cla.Convergence

	stopSyn chan struct{}
	stopAck chan struct{}
}

func NewTCPCLListener(listenAddress string, endpointID bundle.EndpointID, manager *cla.Manager) *TCPCLListener {
	return &TCPCLListener{
		listenAddress: listenAddress,
		endpointID:    endpointID,
		manager:       manager,

		stopSyn: make(chan struct{}),
		stopAck: make(chan struct{}),
	}
}

func (listener *TCPCLListener) Start() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", listener.listenAddress)
	if err != nil {
		return err
	}

	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	go func(ln *net.TCPListener) {
		for {
			select {
			case <-listener.stopSyn:
				for _, c := range listener.clas {
					listener.manager.Unregister(c)
				}

				ln.Close()
				close(listener.stopAck)

				return

			default:
				if err := ln.SetDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
					log.WithError(err).WithField("cla", listener).Warn(
						"TCPCLListener failed to set deadline on TCP socket")

					listener.Close()
				} else if conn, err := ln.Accept(); err == nil {
					client := NewTCPCLClient(conn, listener.endpointID)
					listener.clas = append(listener.clas, client)
					listener.manager.Register(client)
				}
			}
		}
	}(ln)

	return nil
}

func (listener *TCPCLListener) Close() {
	close(listener.stopSyn)
	<-listener.stopAck
}

func (listener TCPCLListener) String() string {
	return fmt.Sprintf("tcpcl://%s", listener.listenAddress)
}
