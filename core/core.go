// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
// SPDX-FileCopyrightText: 2019, 2020 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package core

import (
	"crypto/ed25519"
	"encoding/gob"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/agent"
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/storage"
)

// Core is the inner core of our DTN which handles transmission, reception and
// reception of bundles.
type Core struct {
	InspectAllBundles bool
	NodeId            bundle.EndpointID

	agentManager *AgentManager
	cron         *Cron
	claManager   *cla.Manager
	idKeeper     IdKeeper
	routing      RoutingAlgorithm
	signPriv     ed25519.PrivateKey

	store *storage.Store

	stopSyn chan struct{}
	stopAck chan struct{}
}

// NewCore will be created according to the parameters.
//
// 	storePath: path for the bundle and metadata storage
// 	nodeId: singleton Endpoint ID/Node ID
// 	inspectAllBundles: inspect all administrative records, not only those addressed to this node
// 	routingConf: selected routing algorithm and its configuration
// 	signPriv: optional ed25519 private key (64 bytes long) to sign all outgoing bundles; or nil to not use this feature
func NewCore(storePath string, nodeId bundle.EndpointID, inspectAllBundles bool, routingConf RoutingConf, signPriv ed25519.PrivateKey) (*Core, error) {
	var c = new(Core)

	gob.Register([]bundle.EndpointID{})
	gob.Register(bundle.EndpointID{})
	gob.Register(map[cla.CLAType][]bundle.EndpointID{})
	gob.Register(bundle.DtnEndpoint{})
	gob.Register(bundle.IpnEndpoint{})
	gob.Register(map[Constraint]bool{})
	gob.Register(time.Time{})

	if !nodeId.IsSingleton() {
		return nil, fmt.Errorf("passed Node ID MUST be a singleton; %s is not", nodeId)
	}
	c.InspectAllBundles = inspectAllBundles
	c.NodeId = nodeId

	c.cron = NewCron()

	if store, err := storage.NewStore(storePath); err != nil {
		return nil, err
	} else {
		c.store = store
	}

	c.agentManager = NewAgentManager(c)

	c.claManager = cla.NewManager()

	c.idKeeper = NewIdKeeper()

	if ra, raErr := routingConf.RoutingAlgorithm(c); raErr != nil {
		return nil, raErr
	} else {
		c.routing = ra
	}

	if signPriv != nil {
		if l := len(signPriv); l != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("ed25519 private key's length is %d, not %d", l, ed25519.PrivateKeySize)
		}
		c.signPriv = signPriv

		if err := bundle.GetExtensionBlockManager().Register(&bundle.SignatureBlock{}); err != nil {
			return nil, fmt.Errorf("SignatureBlock registration errored: %v", err)
		}
	}

	c.stopSyn = make(chan struct{})
	c.stopAck = make(chan struct{})

	if err := c.cron.Register("pending_bundles", c.checkPendingBundles, 10*time.Second); err != nil {
		log.WithError(err).Warn("Failed to register pending_bundles at cron")
	}
	if err := c.cron.Register("clean_store", c.store.DeleteExpired, 10*time.Minute); err != nil {
		log.WithError(err).Warn("Failed to register clean_store at cron")
	}

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
				_ = bp.Sync()

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
func (c *Core) RegisterApplicationAgent(app agent.ApplicationAgent) {
	c.agentManager.Register(app)
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

// HasEndpoint checks if the given endpoint ID is assigned either to an
// application or a CLA governed by this Application Agent.
func (c *Core) HasEndpoint(endpoint bundle.EndpointID) bool {
	if c.NodeId.Authority() == endpoint.Authority() {
		return true
	}

	if c.agentManager.HasEndpoint(endpoint) {
		return true
	}

	if c.claManager.HasEndpoint(endpoint) {
		return true
	}

	for _, cr := range c.claManager.Receiver() {
		if cr.GetEndpointID() == endpoint {
			return true
		}
	}

	return false
}

// SendStatusReport creates a new status report in response to the given
// BundlePack and transmits it.
func (c *Core) SendStatusReport(bp BundlePack, status bundle.StatusInformationPos, reason bundle.StatusReportReason) {
	// Don't respond to other administrative records
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

	var sr = bundle.NewStatusReport(*bndl, status, reason, bundle.DtnTimeNow())
	var ar, arErr = bundle.AdministrativeRecordToCbor(&sr)
	if arErr != nil {
		log.WithFields(log.Fields{
			"bundle": bp.ID(),
			"error":  arErr,
		}).Warn("Serializing administrative record failed")

		return
	}

	var aaEndpoint = bp.Receiver
	if aaEndpoint == bundle.DtnNone() {
		aaEndpoint = c.NodeId
	}

	if !c.HasEndpoint(aaEndpoint) && aaEndpoint != c.NodeId {
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

// RegisterConvergable is the exposed Register method from the CLA Manager.
func (c *Core) RegisterConvergable(conv cla.Convergable) {
	c.claManager.Register(conv)
}

// RegisterCLA registers a CLA with the clamanager (just as the RegisterConvergable-method)
// but also adds the CLAs endpoint id to the set of registered IDs for its type.
func (c *Core) RegisterCLA(conv cla.Convergable, claType cla.CLAType, eid bundle.EndpointID) {
	c.claManager.RegisterEndpointID(claType, eid)
	c.claManager.Register(conv)
}

// RegisteredCLAs returns the EndpointIDs of all registered CLAs of the specified type.
// Returns an empty slice if no CLAs of the tye exist.
func (c *Core) RegisteredCLAs(claType cla.CLAType) []bundle.EndpointID {
	return c.claManager.EndpointIDs(claType)
}
