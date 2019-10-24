package bbc

import (
	"bytes"
	"fmt"
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"

	log "github.com/sirupsen/logrus"
)

// Connector implements both a cla.ConvergenceReceiver and cla.ConvergenceSender and supplies the possibility
// to receive and send Bundles over a Modem. However, based on the broadcasting nature of this CLA, addressing
// specific recipients is not possible. Furthermore, attributing senders is also not possible.
type Connector struct {
	modem         Modem
	permanent     bool
	tid           byte
	transmissions map[byte]*IncomingTransmission
	reportChan    chan cla.ConvergenceStatus
}

// NewConnector creates a new Connector, wrapping around the given Modem.
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
			log.WithError(err).Warn("Receiving Fragments from Modem errored")
		} else if trans, ok := c.transmissions[frag.TransmissionID()]; ok {
			if fin, err := trans.ReadFragment(frag); err != nil {
				log.WithError(err).WithField("transaction", trans).Warn("Reading Fragment from Modem errored")
			} else if fin {
				if bndl, err := trans.Bundle(); err != nil {
					log.WithError(err).WithField("transaction", trans).Warn(
						"Extracting Bundle from Transmission errored")
				} else {
					log.WithFields(log.Fields{
						"transaction": trans,
						"bundle":      bndl.ID(),
					}).Info("Bundle Broadcasting Connector received Bundle")

					c.reportChan <- cla.NewConvergenceReceivedBundle(c, bundle.DtnNone(), &bndl)
					delete(c.transmissions, frag.TransmissionID())
				}
			} else {
				log.WithField("transaction", trans).Debug("Received next Fragment for a Transaction")
			}
		} else { // Transmission ID is not stored in the map
			if trans, err := NewIncomingTransmission(frag); err != nil {
				log.WithError(err).Warn("Creating new Transmission from first Fragment errored")
			} else if trans.IsFinished() {
				if bndl, err := trans.Bundle(); err != nil {
					log.WithError(err).WithField("transaction", trans).Warn(
						"Extracting Bundle from Transmission errored")
				} else {
					log.WithFields(log.Fields{
						"transaction": trans,
						"bundle":      bndl.ID(),
					}).Info("Bundle Broadcasting Connector received Bundle")

					c.reportChan <- cla.NewConvergenceReceivedBundle(c, bundle.DtnNone(), &bndl)
				}
			} else {
				log.WithField("transaction", trans).Debug("Starting new Transaction")

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
			log.WithError(err).WithField("fragment", f).Warn("Creating Fragment errored")
			return err
		} else if err := c.modem.Send(f); err != nil {
			log.WithError(err).WithField("fragment", f).Warn("Transmitting Fragment errored")
			return err
		} else if fin {
			log.WithField("fragment", f).Debug("Transmitted last Fragment")
			break
		} else {
			log.WithField("fragment", f).Debug("Transmitted Fragment")
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
