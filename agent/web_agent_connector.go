package agent

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type WebAgentConnector struct {
	conn *websocket.Conn

	closeSyn chan struct{}
	closeAck chan struct{}
}

func NewWebAgentConnector(apiUrl, endpointId string) (wac *WebAgentConnector, err error) {
	var conn *websocket.Conn
	if conn, _, err = websocket.DefaultDialer.Dial(apiUrl, nil); err != nil {
		return
	}

	wac = &WebAgentConnector{
		conn:     conn,
		closeSyn: make(chan struct{}),
		closeAck: make(chan struct{}),
	}

	if err = wac.registerEndpoint(endpointId); err != nil {
		wac = nil
		return
	}

	go wac.handler()

	return
}

func (wac *WebAgentConnector) writeMessage(msg webAgentMessage) error {
	wc, wcErr := wac.conn.NextWriter(websocket.BinaryMessage)
	if wcErr != nil {
		return wcErr
	}

	if cborErr := marshalCbor(msg, wc); cborErr != nil {
		return cborErr
	}

	return wc.Close()
}

func (wac *WebAgentConnector) readMessage() (msg webAgentMessage, err error) {
	if mt, r, rErr := wac.conn.NextReader(); rErr != nil {
		err = rErr
		return
	} else if mt != websocket.BinaryMessage {
		err = fmt.Errorf("expected binary message, got %d", mt)
		return
	} else {
		msg, err = unmarshalCbor(r)
		return
	}
}

func (wac *WebAgentConnector) registerEndpoint(endpointId string) error {
	if err := wac.writeMessage(newRegisterMessage(endpointId)); err != nil {
		return err
	}

	if msg, err := wac.readMessage(); err != nil {
		return err
	} else if status, ok := msg.(*wamStatus); !ok {
		return fmt.Errorf("expected wamStatus, got %T", msg)
	} else if status.errorMsg != "" {
		return fmt.Errorf("received non-empty error message: %s", status.errorMsg)
	} else {
		return nil
	}
}

func (wac *WebAgentConnector) handler() {
	defer func() {
		close(wac.closeAck)
		_ = wac.conn.Close()
	}()

	for {
		select {
		case <-wac.closeSyn:
			return
		}
	}
}

func (wac *WebAgentConnector) Close() {
	close(wac.closeSyn)
	<-wac.closeAck
}
