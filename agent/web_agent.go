package agent

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/gorilla/websocket"
)

type WebAgent struct {
	receiver chan Message
	sender   chan Message

	httpServer *http.Server
	httpMux    *http.ServeMux
	upgrader   websocket.Upgrader
}

func NewWebAgent(address string) (w *WebAgent, err error) {
	httpMux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:    address,
		Handler: httpMux,
	}

	w = &WebAgent{
		receiver: make(chan Message),
		sender:   make(chan Message),

		httpServer: httpServer,
		httpMux:    httpMux,
		upgrader:   websocket.Upgrader{},
	}

	httpMux.HandleFunc("/ws", w.websocketHandler)

	startupErr := make(chan error)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			startupErr <- err
		}

		close(startupErr)
	}()

	select {
	case err = <-startupErr:
		w = nil
	case <-time.After(100 * time.Millisecond):
		go w.handler()
	}

	return
}

func (w *WebAgent) log() *log.Entry {
	return log.WithField("WebAgent", w.httpServer.Addr)
}

// handler is the "generic" handler for a WebAgent.
func (w *WebAgent) handler() {
	defer func() {
		close(w.receiver)
		close(w.sender)
		_ = w.httpServer.Close()
	}()

	for m := range w.receiver {
		switch m := m.(type) {
		case BundleMessage:
			// TODO: forward to specific child processes

		case ShutdownMessage:
			// TODO: shutdown child processes
			return

		default:
			w.log().WithField("message", m).Info("Received unsupported Message")
		}
	}
}

// websocketHandler will be called for each HTTP request to /ws, our WebSocket endpoint.
func (w *WebAgent) websocketHandler(rw http.ResponseWriter, r *http.Request) {
	conn, connErr := w.upgrader.Upgrade(rw, r, nil)
	if connErr != nil {
		w.log().WithError(connErr).Warn("Upgrading HTTP request to WebSocket errored")
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("GuMo")); err != nil {
		w.log().WithError(err).Warn("no gumo ;_;")
	}

	_ = conn.Close()
}

func (w *WebAgent) Endpoints() []bundle.EndpointID {
	// TODO
	return nil
}

func (w *WebAgent) MessageReceiver() chan Message {
	return w.receiver
}

func (w *WebAgent) MessageSender() chan Message {
	return w.sender
}
