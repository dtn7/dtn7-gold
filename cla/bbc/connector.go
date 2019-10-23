package bbc

import (
	"bytes"
	"fmt"
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

type Connector struct {
	modem         Modem
	permanent     bool
	tid           byte
	transmissions map[byte]*IncomingTransmission
	reportChan    chan cla.ConvergenceStatus
}

func NewConnector(modem Modem, permanent bool) *Connector {
	return &Connector{
		modem:         modem,
		permanent:     permanent,
		tid:           1,
		transmissions: make(map[byte]*IncomingTransmission),
		reportChan:    make(chan cla.ConvergenceStatus, 64),
	}
}

func (c *Connector) Start() (error, bool) {
	go c.handler()

	return nil, false
}

func (c *Connector) handler() {
	for {
		if frag, err := c.modem.Receive(); err != nil {
			// TODO
		} else if trans, ok := c.transmissions[frag.TransmissionID()]; ok {
			if fin, err := trans.ReadFragment(frag); err != nil {
				// TODO
			} else if fin {
				if bndl, err := trans.Bundle(); err != nil {
					// TODO
				} else {
					c.reportChan <- cla.NewConvergenceReceivedBundle(c, bundle.DtnNone(), &bndl)
					delete(c.transmissions, frag.TransmissionID())
				}
			}
		} else { // Transmission ID is not stored in the map
			if trans, err := NewIncomingTransmission(frag); err != nil {
				// TODO
			} else if trans.IsFinished() {
				if bndl, err := trans.Bundle(); err != nil {
					// TODO
				} else {
					c.reportChan <- cla.NewConvergenceReceivedBundle(c, bundle.DtnNone(), &bndl)
				}
			} else {
				c.transmissions[frag.TransmissionID()] = trans
			}
		}
	}
}

func (c *Connector) Close() {
	// TODO
}

func (c *Connector) Channel() chan cla.ConvergenceStatus {
	return c.reportChan
}

func (c *Connector) Address() string {
	return fmt.Sprintf("bbc://%v/", c.modem)
}

func (c *Connector) IsPermanent() bool {
	return c.permanent
}

func (c *Connector) Send(bndl *bundle.Bundle) error {
	var buf bytes.Buffer
	if err := bndl.WriteBundle(&buf); err != nil {
		return err
	}

	var t, tErr = NewOutgoingTransmission(c.tid, buf.Bytes(), c.modem.Mtu())
	if tErr != nil {
		return tErr
	}

	for {
		if f, fin, err := t.WriteFragment(); err != nil {
			return err
		} else if err := c.modem.Send(f); err != nil {
			return err
		} else if fin {
			break
		}
	}

	return nil
}

func (c *Connector) GetPeerEndpointID() bundle.EndpointID {
	return bundle.DtnNone()
}

func (c *Connector) GetEndpointID() bundle.EndpointID {
	return bundle.DtnNone()
}
