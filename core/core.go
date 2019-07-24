package core

import (
	"encoding/gob"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/bundle/arecord"
	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/storage"
)

// Core is the inner core of our DTN which handles transmission, reception and
// reception of bundles.
type Core struct {
	Agents            []ApplicationAgent
	InspectAllBundles bool
	NodeId            bundle.EndpointID

	cron       *Cron
	claManager *cla.Manager
	idKeeper   IdKeeper
	routing    RoutingAlgorithm

	store *storage.Store

	stopSyn chan struct{}
	stopAck chan struct{}
}

// NewCore creates and returns a new Core. A SimpleStore will be created or used
// at the given path. The inspectAllBundles flag indicates if all
// administrative records - next to the bundles addressed to this node - should
// be inspected. This allows bundle deletion for forwarding bundles.
func NewCore(storePath string, nodeId bundle.EndpointID, inspectAllBundles bool, routing string) (*Core, error) {
	var c = new(Core)

	gob.Register([]bundle.EndpointID{})
	gob.Register(bundle.EndpointID{})
	gob.Register(map[Constraint]bool{})
	gob.Register(time.Time{})

	c.cron = NewCron()

	c.InspectAllBundles = inspectAllBundles
	c.NodeId = nodeId

	if store, err := storage.NewStore(storePath); err != nil {
		return nil, err
	} else {
		c.store = store
	}

	c.claManager = cla.NewManager()

	c.idKeeper = NewIdKeeper()

	switch routing {
	case "epidemic":
		c.routing = NewEpidemicRouting(c)
	case "spray":
		c.routing = NewSprayAndWait(c)
	case "binary_spray":
		c.routing = NewBinarySpray(c)
	case "dtlsr":
		c.routing = NewDTLSR(c)
	default:
		log.WithFields(log.Fields{
			"routing_string": routing,
		}).Fatal("Unknown routing algorithm")
	}

	c.stopSyn = make(chan struct{})
	c.stopAck = make(chan struct{})

	c.cron.Register("pending_bundles", c.checkPendingBundles, 10*time.Second)
	c.cron.Register("clean_store", c.store.DeleteExpired, 10*time.Minute)

	go c.handler()

	return c, nil
}

// SetRoutingAlgorithm overwrites the used RoutingAlgorithm, which defaults to
// EpidemicRouting.
func (c *Core) SetRoutingAlgorithm(routing RoutingAlgorithm) {
	c.routing = routing
}

// checkPendingBundles queries pending bundle (packs) from the store and
// tries to dispatch them.
func (c *Core) checkPendingBundles() {
	if bis, err := c.store.QueryPending(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Failed to fetch pending bundle packs")
	} else {
		for _, bi := range bis {
			log.WithFields(log.Fields{
				"bundle": bi.Id,
			}).Info("Retrying bundle from store")

			c.dispatching(NewBundlePack(bi.BId, c.store))
		}
	}
}

// handler does the Core's background tasks
func (c *Core) handler() {
	for {
		select {
		// Invoked by Close(), shuts down
		case <-c.stopSyn:
			c.cron.Stop()

			c.claManager.Close()

			if storeErr := c.store.Close(); storeErr != nil {
				log.WithFields(log.Fields{
					"error": storeErr,
				}).Warn("Closing store while shutting down the Core errors")
			}

			close(c.stopAck)
			return

		// Handle a received ConvergenceStatus
		case cs := <-c.claManager.Channel():
			switch cs.MessageType {
			case cla.ReceivedBundle:
				crb := cs.Message.(cla.ConvergenceReceivedBundle)

				bp := NewBundlePackFromBundle(*crb.Bundle, c.store)
				bp.Receiver = crb.Endpoint
				bp.Sync()

				c.receive(bp)

			case cla.PeerAppeared:
				c.routing.ReportPeerAppeared(cs.Sender)
				c.checkPendingBundles()

			case cla.PeerDisappeared:
				c.routing.ReportPeerDisappeared(cs.Sender)

			default:
				log.WithFields(log.Fields{
					"cla":    cs.Sender,
					"type":   cs.MessageType,
					"status": cs,
				}).Warn("Received ConvergenceStatus with unknown type")
			}
		}
	}
}

// Close shuts the Core down and notifies all bounded ConvergenceReceivers to
// also close the connection.
func (c *Core) Close() {
	close(c.stopSyn)
	<-c.stopAck
}

// RegisterApplicationAgent adds a new ApplicationAgent to this Core's list.
func (c *Core) RegisterApplicationAgent(agent ApplicationAgent) {
	c.Agents = append(c.Agents, agent)
}

// senderForDestination returns an array of ConvergenceSenders whose endpoint ID
// equals the requested one. This is used for direct delivery, comparing the
// PrimaryBlock's destination to the assigned endpoint ID of each CLA.
func (c *Core) senderForDestination(endpoint bundle.EndpointID) (css []cla.ConvergenceSender) {
	for _, cs := range c.claManager.Sender() {
		if cs.GetPeerEndpointID() == endpoint {
			css = append(css, cs)
		}
	}
	return
}

// hasEndpoint checks if this Core has some endpoint, but does not secure this
// request with a Mutex. Therefore, the safe HasEndpoint method exists.
func (c *Core) hasEndpoint(endpoint bundle.EndpointID) bool {
	for _, agent := range c.Agents {
		if agent.EndpointID() == endpoint {
			return true
		}
	}

	for _, cr := range c.claManager.Receiver() {
		if cr.GetEndpointID() == endpoint {
			return true
		}
	}

	return false
}

// HasEndpoint returns true if the given endpoint ID is assigned either to an
// application or a CLA governed by this Application Agent.
func (c *Core) HasEndpoint(endpoint bundle.EndpointID) (state bool) {
	state = c.hasEndpoint(endpoint)

	return
}

// SendStatusReport creates a new status report in response to the given
// BundlePack and transmits it.
func (c *Core) SendStatusReport(bp BundlePack,
	status arecord.StatusInformationPos, reason arecord.StatusReportReason) {
	// Don't repond to other administrative records
	bndl, _ := bp.Bundle()
	if bndl.PrimaryBlock.BundleControlFlags.Has(bundle.AdministrativeRecordPayload) {
		return
	}

	// Don't respond to ourself
	if c.HasEndpoint(bndl.PrimaryBlock.ReportTo) {
		return
	}

	log.WithFields(log.Fields{
		"bundle": bp.ID(),
		"status": status,
		"reason": reason,
	}).Info("Sending a status report for a bundle")

	var sr = arecord.NewStatusReport(*bndl, status, reason, bundle.DtnTimeNow())
	var ar, arErr = arecord.AdministrativeRecordToCbor(&sr)
	if arErr != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"error":  arErr,
		}).Warn("Serializing administrative record failed")

		return
	}

	var aaEndpoint = bp.Receiver
	if !c.HasEndpoint(aaEndpoint) {
		log.WithFields(log.Fields{
			"bundle":   bp.ID(),
			"endpoint": aaEndpoint,
		}).Warn("Failed to create status report, receiver is not a current endpoint")

		return
	}

	var outBndl, err = bundle.Builder().
		BundleCtrlFlags(bundle.AdministrativeRecordPayload).
		Source(aaEndpoint).
		Destination(bndl.PrimaryBlock.ReportTo).
		CreationTimestampNow().
		Lifetime("60m").
		Canonical(ar).
		Build()

	if err != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"error":  err,
		}).Warn("Creating status report bundle failed")

		return
	}

	c.SendBundle(&outBndl)
}

// RegisterCla is the exposed Register method from the CLA Manager.
func (c *Core) RegisterConvergence(conv cla.Convergence) {
	c.claManager.Register(conv)
}
