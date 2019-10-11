package tcpcl

import (
	"bufio"
	"io"
	"net"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

func (client *TCPCLClient) handleMeta() {
	for range client.handleMetaStop {
		client.log().Info("Handler received stop signal")

		client.state.Terminate()

		chans := []chan struct{}{client.handlerConnInStop, client.handlerConnOutStop, client.handlerStateStop}
		for _, chn := range chans {
			close(chn)
		}

		close(client.handleMetaStopAck)

		return
	}
}

func (client *TCPCLClient) handleConnIn() {
	defer func() {
		client.log().Debug("Leaving incoming connection handler")
		client.handleMetaStop <- struct{}{}
	}()

	var r = bufio.NewReader(client.conn)

	for {
		select {
		case <-client.handlerConnInStop:
			return

		default:
			if err := client.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
				client.log().WithError(err).Error("Setting read deadline errored")
				return
			}

			if msg, err := ReadMessage(r); err == nil {
				client.log().WithField("msg", msg).Debug("Received message")
				client.msgsIn <- msg
			} else if err == io.EOF {
				client.log().Info("Read EOF, closing down.")
				return
			} else if netErr, ok := err.(net.Error); ok && !netErr.Timeout() {
				client.log().WithError(netErr).Error("Network error occurred")
				return
			} else if !ok {
				client.log().WithError(err).Error("Parsing next message errored")
				return
			}
		}
	}
}

func (client *TCPCLClient) handleConnOut() {
	defer func() {
		client.log().Debug("Leaving outgoing connection handler")
		client.handleMetaStop <- struct{}{}
	}()

	var w = bufio.NewWriter(client.conn)

	for {
		select {
		case <-client.handlerConnOutStop:
			return

		case msg := <-client.msgsOut:
			if err := msg.Marshal(w); err != nil {
				client.log().WithError(err).WithField("msg", msg).Error("Sending message errored")
				return
			} else if err := w.Flush(); err != nil {
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
		}
	}
}

func (client *TCPCLClient) handleState() {
	defer func() {
		client.log().Debug("Leaving state handler")
		client.handleMetaStop <- struct{}{}
	}()

	for {
		select {
		case <-client.handlerStateStop:
			return

		default:
			switch {
			case !client.state.IsTerminated():
				var stateHandler func() error

				switch {
				case client.state.IsContact():
					stateHandler = client.handleContact
				case client.state.IsInit():
					stateHandler = client.handleSessInit
				case client.state.IsEstablished():
					stateHandler = client.handleEstablished
				default:
					client.log().WithField("state", client.state).Fatal("Illegal state")
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

			default:
				client.log().Info("Entering Termination state")

				var sessTerm = NewSessionTerminationMessage(0, TerminationUnknown)
				client.msgsOut <- &sessTerm

				emptyEndpoint := bundle.EndpointID{}
				if client.endpointID != emptyEndpoint {
					client.reportChan <- cla.NewConvergencePeerDisappeared(client, client.peerEndpointID)
				}

				return
			}
		}
	}
}