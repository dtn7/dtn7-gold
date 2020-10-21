// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
// SPDX-FileCopyrightText: 2019, 2020 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"crypto/ed25519"
	"encoding/gob"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/agent"
	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
	"github.com/dtn7/dtn7-go/pkg/storage"
)

// Core is the inner processing of our DTN which handles transmission, reception and
// reception of bundles.
type Core struct {
	InspectAllBundles bool
	NodeId            bpv7.EndpointID

	agentManager *AgentManager
	cron         *Cron
	claManager   *cla.Manager
	idKeeper     IdKeeper
	routing      Algorithm
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
func NewCore(storePath string, nodeId bpv7.EndpointID, inspectAllBundles bool, routingConf RoutingConf, signPriv ed25519.PrivateKey) (*Core, error) {
	var c = new(Core)

	gob.Register([]bpv7.EndpointID{})
	gob.Register(bpv7.EndpointID{})
	gob.Register(map[cla.CLAType][]bpv7.EndpointID{})
	gob.Register(bpv7.DtnEndpoint{})
	gob.Register(bpv7.IpnEndpoint{})
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

		if err := bpv7.GetExtensionBlockManager().Register(&bpv7.SignatureBlock{}); err != nil {
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

// SetRoutingAlgorithm overwrites the used Algorithm, which defaults to
// EpidemicRouting.
func (c *Core) SetRoutingAlgorithm(routing Algorithm) {
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

			c.dispatching(NewBundleDescriptor(bi.BId, c.store))
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

			if err := c.claManager.Close(); err != nil {
				log.WithError(err).Warn("Closing CLA Manager while shutting down errored")
			}

			if err := c.store.Close(); err != nil {
				log.WithError(err).Warn("Closing store while shutting down errored")
			}

			close(c.stopAck)
			return

		// Handle a received ConvergenceStatus
		case cs := <-c.claManager.Channel():
			switch cs.MessageType {
			case cla.ReceivedBundle:
				crb := cs.Message.(cla.ConvergenceReceivedBundle)

				bp := NewBundleDescriptorFromBundle(*crb.Bundle, c.store)
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
func (c *Core) senderForDestination(endpoint bpv7.EndpointID) (css []cla.ConvergenceSender) {
	for _, cs := range c.claManager.Sender() {
		if cs.GetPeerEndpointID().SameNode(endpoint) {
			css = append(css, cs)
		}
	}
	return
}

// HasEndpoint checks if the given endpoint ID is assigned either to an
// application or a CLA governed by this Application Agent.
func (c *Core) HasEndpoint(endpoint bpv7.EndpointID) bool {
	if c.NodeId.SameNode(endpoint) {
		return true
	}

	if c.agentManager.HasEndpoint(endpoint) {
		return true
	}

	if c.claManager.HasEndpoint(endpoint) {
		return true
	}

	for _, cr := range c.claManager.Receiver() {
		if cr.GetEndpointID().SameNode(endpoint) {
			return true
		}
	}

	return false
}

// SendStatusReport creates a new status report in response to the given
// BundleDescriptor and transmits it.
func (c *Core) SendStatusReport(descriptor BundleDescriptor, status bpv7.StatusInformationPos, reason bpv7.StatusReportReason) {
	// Don't respond to other administrative records
	bndl, _ := descriptor.Bundle()
	if bndl.PrimaryBlock.BundleControlFlags.Has(bpv7.AdministrativeRecordPayload) {
		return
	}

	// Don't respond to ourself
	if c.HasEndpoint(bndl.PrimaryBlock.ReportTo) {
		return
	}

	log.WithFields(log.Fields{
		"bundle": descriptor.ID(),
		"status": status,
		"reason": reason,
	}).Info("Sending a status report for a bundle")

	var sr = bpv7.NewStatusReport(*bndl, status, reason, bpv7.DtnTimeNow())
	var ar, arErr = bpv7.AdministrativeRecordToCbor(&sr)
	if arErr != nil {
		log.WithFields(log.Fields{
			"bundle": descriptor.ID(),
			"error":  arErr,
		}).Warn("Serializing administrative record failed")

		return
	}

	var aaEndpoint = descriptor.Receiver
	if aaEndpoint == bpv7.DtnNone() {
		aaEndpoint = c.NodeId
	}

	if !c.HasEndpoint(aaEndpoint) && aaEndpoint != c.NodeId {
		log.WithFields(log.Fields{
			"bundle":   descriptor.ID(),
			"endpoint": aaEndpoint,
		}).Warn("Failed to create status report, receiver is not a current endpoint")

		return
	}

	var outBndl, err = bpv7.Builder().
		BundleCtrlFlags(bpv7.AdministrativeRecordPayload).
		Source(aaEndpoint).
		Destination(bndl.PrimaryBlock.ReportTo).
		CreationTimestampNow().
		Lifetime("60m").
		Canonical(ar).
		Build()

	if err != nil {
		log.WithFields(log.Fields{
			"bundle": descriptor.ID(),
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
func (c *Core) RegisterCLA(conv cla.Convergable, claType cla.CLAType, eid bpv7.EndpointID) {
	c.claManager.RegisterEndpointID(claType, eid)
	c.claManager.Register(conv)
}

// RegisteredCLAs returns the EndpointIDs of all registered CLAs of the specified type.
// Returns an empty slice if no CLAs of the tye exist.
func (c *Core) RegisteredCLAs(claType cla.CLAType) []bpv7.EndpointID {
	return c.claManager.EndpointIDs(claType)
}
