package bbc

import (
	"bytes"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
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
	var logger = log.WithField("bbc", c.Address())

	for {
		if frag, err := c.modem.Receive(); err != nil {
			logger.WithError(err).Warn("Receiving Fragments from Modem errored")
		} else if trans, ok := c.transmissions[frag.TransmissionID()]; ok {
			if fin, err := trans.ReadFragment(frag); err != nil {
				logger.WithError(err).WithField("transaction", trans).Warn("Reading Fragment from Modem errored")
			} else if fin {
				if bndl, err := trans.Bundle(); err != nil {
					logger.WithError(err).WithField("transaction", trans).Warn(
						"Extracting Bundle from Transmission errored")
				} else {
					logger.WithFields(log.Fields{
						"transaction": trans,
						"bundle":      bndl.ID(),
					}).Info("Bundle Broadcasting Connector received Bundle")

					c.reportChan <- cla.NewConvergenceReceivedBundle(c, bundle.DtnNone(), &bndl)
					delete(c.transmissions, frag.TransmissionID())
				}
			} else {
				logger.WithField("transaction", trans).Debug("Received next Fragment for a Transaction")
			}
		} else { // Transmission ID is not stored in the map
			if trans, err := NewIncomingTransmission(frag); err != nil {
				logger.WithError(err).Warn("Creating new Transmission from first Fragment errored")
			} else if trans.IsFinished() {
				if bndl, err := trans.Bundle(); err != nil {
					logger.WithError(err).WithField("transaction", trans).Warn(
						"Extracting Bundle from Transmission errored")
				} else {
					logger.WithFields(log.Fields{
						"transaction": trans,
						"bundle":      bndl.ID(),
					}).Info("Bundle Broadcasting Connector received Bundle")

					c.reportChan <- cla.NewConvergenceReceivedBundle(c, bundle.DtnNone(), &bndl)
				}
			} else {
				logger.WithField("transaction", trans).Debug("Starting new Transaction")

				c.transmissions[frag.TransmissionID()] = trans
			}
		}
	}
}

func (c *Connector) Close() {
	if err := c.modem.Close(); err != nil {
		log.WithField("bbc", c.Address()).WithError(err).Warn("Closing Modem errored")
	}
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
		f, fin, err := t.WriteFragment()
		logger := log.WithFields(log.Fields{
			"bbc":      c.Address(),
			"fragment": f,
		})

		if err != nil {
			logger.WithError(err).Warn("Creating Fragment errored")
			return err
		} else if err := c.modem.Send(f); err != nil {
			logger.WithError(err).Warn("Transmitting Fragment errored")
			return err
		} else if fin {
			logger.Debug("Transmitted last Fragment")
			break
		} else {
			logger.Debug("Transmitted Fragment")
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

func (c *Connector) String() string {
	return c.Address()
}
