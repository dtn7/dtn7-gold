package tcpcl

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

type TCPCLServer struct {
	listenAddress string
	reportChan    chan cla.ConvergenceStatus
	endpointID    bundle.EndpointID

	stopSyn chan struct{}
	stopAck chan struct{}
}

func NewTCPCLServer(listenAddress string, endpointID bundle.EndpointID, permanent bool) *TCPCLServer {
	return &TCPCLServer{
		listenAddress: listenAddress,
		reportChan:    make(chan cla.ConvergenceStatus),
		endpointID:    endpointID,
		stopSyn:       make(chan struct{}),
		stopAck:       make(chan struct{}),
	}
}

func (serv *TCPCLServer) Start() (error, bool) {
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
				if err := ln.SetDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
					log.WithFields(log.Fields{
						"cla":   serv,
						"error": err,
					}).Warn("TCPCLServer failed to set deadline on TCP socket")

					serv.Close()
				} else if conn, err := ln.Accept(); err == nil {
					go func() {
						client := NewTCPCLClient(conn, serv.endpointID)

						if err, _ := client.Start(); err != nil {
							log.WithError(err).WithField("cla", serv).Warn("Starting Client errored")
							return
						}

						for {
							serv.reportChan <- <-client.Channel()
						}
					}()
				}
			}
		}
	}(ln)

	return nil, true
}

func (serv *TCPCLServer) Channel() chan cla.ConvergenceStatus {
	return serv.reportChan
}

func (serv *TCPCLServer) Close() {
	close(serv.stopSyn)
	<-serv.stopAck
}

func (serv TCPCLServer) String() string {
	return fmt.Sprintf("tcpcl://%s", serv.listenAddress)
}
