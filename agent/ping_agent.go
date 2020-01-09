package agent

import (
	"github.com/dtn7/dtn7-go/bundle"
)

// Ping is a simple ApplicationAgent to "pong" / acknowledge incoming Bundles.
type Ping struct {
	endpoint bundle.EndpointID
	receiver chan Message
	sender   chan Message
}

// NewPing creates a new Ping ApplicationAgent.
func NewPing(endpoint bundle.EndpointID) *Ping {
	p := &Ping{
		endpoint: endpoint,
		receiver: make(chan Message),
		sender:   make(chan Message),
	}

	go p.handler()

	return p
}

func (p *Ping) handler() {
	defer func() {
		close(p.receiver)
		close(p.sender)
	}()

	for {
		select {
		case m := <-p.receiver:
			switch m.(type) {
			case BundleMessage:
				p.ackBundle(m.(BundleMessage).Bundle)

			case ShutdownMessage:
				return
			}
		}
	}
}

func (p *Ping) ackBundle(b bundle.Bundle) {
	bndl, err := bundle.Builder().
		Source(p.endpoint).
		Destination(b.PrimaryBlock.ReportTo).
		CreationTimestampNow().
		Lifetime("24h").
		HopCountBlock(64).
		PayloadBlock([]byte("pong")).
		Build()

	if err != nil {
		panic(";_;")
	}

	p.sender <- BundleMessage{bndl}
}

func (p *Ping) Endpoints() []bundle.EndpointID {
	return []bundle.EndpointID{p.endpoint}
}

func (p *Ping) MessageReceiver() chan Message {
	return p.receiver
}

func (p *Ping) MessageSender() chan Message {
	return p.sender
}
