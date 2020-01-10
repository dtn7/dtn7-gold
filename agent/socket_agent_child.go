package agent

import (
	"net"
	"time"

	"github.com/dtn7/dtn7-go/bundle"
)

// socketChild is a child process / managed connection of a SocketAgent.
type socketChild struct {
	id       string
	conn     net.Conn
	socket   *SocketAgent
	receiver chan bundle.Bundle
}

func launchSocketChild(conn net.Conn, socket *SocketAgent) {
	child := &socketChild{
		id:       conn.RemoteAddr().String(),
		conn:     conn,
		socket:   socket,
		receiver: make(chan bundle.Bundle),
	}

	socket.children.Store(child.id, child)

	child.handler()
}

func (sc *socketChild) handler() {
	logger := sc.socket.log().WithField("Child", sc.conn.RemoteAddr().String())

	defer func() {
		close(sc.receiver)
		_ = sc.conn.Close()
		sc.socket.children.Delete(sc.id)

		logger.Debug("Closing down child process")
	}()

	for {
		select {
		case b := <-sc.receiver:
			if err := b.MarshalCbor(sc.conn); err != nil {
				logger.WithError(err).Warn("Marshalling Bundle errored")
				return
			}
			logger.WithField("bundle", b).Info("Wrote Bundle to child")

		default:
			if err := sc.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
				logger.WithError(err).Warn("Setting deadline errored")
				return
			}

			var b bundle.Bundle
			if err := b.UnmarshalCbor(sc.conn); err == nil {
				logger.WithField("bundle", b).Info("Read Bundle from child")
				sc.socket.sender <- BundleMessage{b}
			}
		}
	}
}
