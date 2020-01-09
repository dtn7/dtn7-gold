package agent

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

// Socket is a ApplicationAgent to send and receive raw Bundles on a TCP socket.
type Socket struct {
	listener *net.TCPListener
	// children: map[socketChild.id]*socketChild
	children sync.Map
	endpoint bundle.EndpointID
	receiver chan Message
	sender   chan Message
}

// NewSocket starts a new Socket on the given TCP address.
func NewSocket(address string, endpoint bundle.EndpointID) (s *Socket, err error) {
	addr, addrErr := net.ResolveTCPAddr("tcp", address)
	if addrErr != nil {
		err = addrErr
		return
	}

	l, lErr := net.ListenTCP("tcp", addr)
	if lErr != nil {
		err = lErr
		return
	}

	s = &Socket{
		listener: l,
		endpoint: endpoint,
		receiver: make(chan Message),
		sender:   make(chan Message),
	}

	go s.handler()

	return
}

func (s *Socket) log() *log.Entry {
	return log.WithField("Socket", s.listener)
}

func (s *Socket) handler() {
	defer func() {
		close(s.receiver)
		close(s.sender)
		_ = s.listener.Close()
	}()

	for {
		select {
		case m := <-s.receiver:
			switch m.(type) {
			case BundleMessage:
				bndl := m.(BundleMessage).Bundle
				s.children.Range(func(_, child interface{}) bool {
					child.(*socketChild).receiver <- bndl
					return true
				})

			case ShutdownMessage:
				return

			default:
				s.log().WithField("message", m).Info("Received unsupported Message")
			}

		default:
			if err := s.listener.SetDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
				s.log().WithError(err).Warn("Setting deadline errored")
				return
			} else if conn, err := s.listener.Accept(); err == nil {
				go launchSocketChild(conn, s)
			}
		}
	}
}

func (s *Socket) Endpoints() []bundle.EndpointID {
	return []bundle.EndpointID{s.endpoint}
}

func (s *Socket) MessageReceiver() chan Message {
	return s.receiver
}

func (s *Socket) MessageSender() chan Message {
	return s.sender
}
