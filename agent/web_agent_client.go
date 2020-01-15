package agent

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/gorilla/websocket"
)

type webAgentClient struct {
	sync.Mutex

	conn     *websocket.Conn
	endpoint bundle.EndpointID
	receiver chan Message
	sender   chan Message
}

func newWebAgentClient(conn *websocket.Conn) *webAgentClient {
	return &webAgentClient{
		conn:     conn,
		endpoint: bundle.EndpointID{},
		receiver: make(chan Message),
		sender:   make(chan Message),
	}
}

func (client *webAgentClient) start() {
	client.handleConn()
}

func (client *webAgentClient) handleReceiver() {
	var logger = log.WithField("web agent client", client.conn.RemoteAddr().String())

	defer func() {
		close(client.receiver)
		close(client.sender)
		_ = client.conn.Close()
	}()

	for msg := range client.receiver {
		switch msg := msg.(type) {
		case ShutdownMessage:
			return

		case BundleMessage:
			if err := client.handleOutgoingBundle(msg.Bundle); err != nil {
				logger.WithError(err).Warn("Sending outgoing Bundle errored")
				return
			}
		}
	}
}

func (client *webAgentClient) handleConn() {
	var logger = log.WithField("web agent client", client.conn.RemoteAddr().String())

	defer func() {
		close(client.receiver)
		close(client.sender)
		_ = client.conn.Close()
	}()

	for {
		if messageType, reader, err := client.conn.NextReader(); err != nil {
			logger.WithError(err).Warn("Opening next Websocket Reader errored")
			return
		} else if messageType != websocket.BinaryMessage {
			logger.WithField("message type", messageType).Warn("Websocket Reader's type is not binary")
			return
		} else if message, err := unmarshalCbor(reader); err != nil {
			logger.WithError(err).Warn("Unmarshal CBOR errored")
			return
		} else {
			var err error

			switch message := message.(type) {
			case *wamRegister:
				err = client.handleIncomingRegister(message)

			case *wamBundle:
				// TODO

			default:
				// TODO
			}

			if err = client.acknowledgeIncoming(err); err != nil {
				logger.WithError(err).Warn("Handling incoming message / acknowledgment errored")
				return
			}
		}
	}
}

func (client *webAgentClient) handleIncomingRegister(m *wamRegister) error {
	client.Lock()
	defer client.Unlock()

	var logger = log.WithFields(log.Fields{
		"web agent client": client.conn.RemoteAddr().String(),
		"message":          m,
	})

	if client.endpoint == (bundle.EndpointID{}) {
		if eid, err := bundle.NewEndpointID(m.endpoint); err != nil {
			logger.WithError(err).Warn("Parsing endpoint ID errored")
			return err
		} else {
			logger.WithField("endpoint", eid).Debug("Setting endpoint id")
			client.endpoint = eid
			return nil
		}
	} else {
		msg := "register errored, an endpoint ID is already present"
		logger.Warn(msg)
		return fmt.Errorf(msg)
	}
}

func (client *webAgentClient) handleOutgoingBundle(b bundle.Bundle) error {
	client.Lock()
	defer client.Unlock()

	wc, wcErr := client.conn.NextWriter(websocket.BinaryMessage)
	if wcErr != nil {
		return wcErr
	}

	if cborErr := marshalCbor(newBundleMessage(b), wc); cborErr != nil {
		return cborErr
	}

	if wcErr := wc.Close(); wcErr != nil {
		return wcErr
	}

	return nil
}

func (client *webAgentClient) acknowledgeIncoming(err error) error {
	client.Lock()
	defer client.Unlock()

	wc, wcErr := client.conn.NextWriter(websocket.BinaryMessage)
	if wcErr != nil {
		return wcErr
	}

	if cborErr := marshalCbor(newStatusMessage(err), wc); cborErr != nil {
		return cborErr
	}

	if wcErr := wc.Close(); wcErr != nil {
		return wcErr
	}

	return err
}

func (client *webAgentClient) Endpoints() []bundle.EndpointID {
	client.Lock()
	defer client.Unlock()

	if client.endpoint == (bundle.EndpointID{}) {
		return nil
	} else {
		return []bundle.EndpointID{client.endpoint}
	}
}

func (client *webAgentClient) MessageReceiver() chan Message {
	return client.receiver
}

func (client *webAgentClient) MessageSender() chan Message {
	return client.sender
}
