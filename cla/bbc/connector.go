package bbc

import (
	"bytes"
	"fmt"
	"io"

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

	closedSyn chan struct{}
	closedAck chan struct{}
}

// NewConnector creates a new Connector, wrapping around the given Modem.
func NewConnector(modem Modem, permanent bool) *Connector {
	return &Connector{
		modem:         modem,
		permanent:     permanent,
		tid:           randomTransmissionId(),
		transmissions: make(map[byte]*IncomingTransmission),
		reportChan:    make(chan cla.ConvergenceStatus, 64),
	}
}

func (c *Connector) Start() (error, bool) {
	c.closedSyn = make(chan struct{})
	c.closedAck = make(chan struct{})

	go c.handler()

	return nil, false
}

func (c *Connector) handler() {
	defer close(c.closedAck)

	var logger = log.WithField("bbc", c.Address())

	for {
		select {
		case <-c.closedSyn:
			logger.Info("Received close signal, stopping handler")
			return

		default:
			if frag, err := c.modem.Receive(); err == io.EOF {
				logger.Info("Read EOF, stopping handler")
				return
			} else if err != nil {
				logger.WithError(err).Warn("Receiving Fragments from Modem errored")
			} else if err = c.handleIncomingFragment(frag); err != nil {
				logger.WithError(err).Warn("Handling incoming Fragment errored")
			}
		}
	}
}

// handleIncomingFragment inspects a new Fragment and tries to add it to a new or a known Transmission.
func (c *Connector) handleIncomingFragment(frag Fragment) (err error) {
	var (
		logger = log.WithField("bbc", c.Address())

		transmission *IncomingTransmission
		known        bool
	)

	if transmission, known = c.transmissions[frag.TransmissionID()]; !known {
		transmission, err = c.handleIncomingNewTransmission(frag)
	} else {
		err = c.handleIncomingKnownTransmission(frag, transmission)
	}
	if err != nil {
		logger.WithError(err).WithField("transmission", transmission).Warn(
			"Fetching or creating Transmission errored")

		return
	}

	if transmission.IsFinished() {
		var bndl bundle.Bundle

		if bndl, err = transmission.Bundle(); err == nil {
			logger.WithFields(log.Fields{
				"transaction": transmission,
				"bundle":      bndl.ID(),
			}).Info("Bundle Broadcasting Connector received Bundle")

			c.reportChan <- cla.NewConvergenceReceivedBundle(c, bundle.DtnNone(), &bndl)
		} else {
			// Returning error variable err keeps its value and cleanup code follows. That's why we don't return here.
			logger.WithError(err).WithField("transmission", transmission).Warn(
				"Extracting Bundle from Transmission errored")
		}

		delete(c.transmissions, transmission.TransmissionID)
	}
	return
}

// handleIncomingNewTransmission creates a new Transmission for a Fragment with an unknown Transmission ID.
func (c *Connector) handleIncomingNewTransmission(frag Fragment) (trans *IncomingTransmission, err error) {
	if trans, err = NewIncomingTransmission(frag); err == nil {
		c.transmissions[trans.TransmissionID] = trans
	}
	return
}

// handleIncomingKnownTransmission updates a known Transmission with the next Fragment.
func (c *Connector) handleIncomingKnownTransmission(frag Fragment, trans *IncomingTransmission) (err error) {
	if _, err = trans.ReadFragment(frag); err != nil {
		delete(c.transmissions, trans.TransmissionID)
	}
	return
}

func (c *Connector) Close() {
	close(c.closedSyn)

	if err := c.modem.Close(); err != nil {
		log.WithField("bbc", c.Address()).WithError(err).Warn("Closing Modem errored")
	}

	<-c.closedAck
}

func (c *Connector) Channel() chan cla.ConvergenceStatus {
	return c.reportChan
}

func (c *Connector) Address() string {
	return fmt.Sprintf("bbc://%v", c.modem)
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

	c.tid = nextTransmissionId(c.tid)

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
