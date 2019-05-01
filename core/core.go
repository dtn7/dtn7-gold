package core

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla"
)

// isKnownBlockType checks if this program's core knows the given block type.
func isKnownBlockType(blocktype bundle.CanonicalBlockType) bool {
	switch blocktype {
	case
		bundle.PayloadBlock,
		bundle.PreviousNodeBlock,
		bundle.BundleAgeBlock,
		bundle.HopCountBlock:
		return true

	default:
		return false
	}
}

// Core is the inner core of our DTN which handles transmission, reception and
// reception of bundles.
type Core struct {
	Agents            []ApplicationAgent
	InspectAllBundles bool
	NodeId            bundle.EndpointID

	// Used by the convergence methods, defined in core/convergence.go
	convergenceSenders   []cla.ConvergenceSender
	convergenceReceivers []cla.ConvergenceReceiver
	convergenceQueue     []*convergenceQueueElement
	convergenceMutex     sync.Mutex

	idKeeper IdKeeper
	store    Store
	routing  RoutingAlgorithm

	reloadConvRecs chan struct{}
	stopSyn        chan struct{}
	stopAck        chan struct{}
}

// NewCore creates and returns a new Core. A SimpleStore will be created or used
// at the given path. The inspectAllBundles flag indicates if all
// administrative records - next to the bundles addressed to this node - should
// be inspected. This allows bundle deletion for forwarding bundles.
func NewCore(storePath string, nodeId bundle.EndpointID, inspectAllBundles bool) (*Core, error) {
	var c = new(Core)

	c.InspectAllBundles = inspectAllBundles
	c.NodeId = nodeId

	store, err := NewSimpleStore(storePath)
	if err != nil {
		return nil, err
	}
	c.store = store

	c.idKeeper = NewIdKeeper()
	c.reloadConvRecs = make(chan struct{}, 9000)

	c.routing = NewEpidemicRouting(c, false)

	c.stopSyn = make(chan struct{})
	c.stopAck = make(chan struct{})

	go c.checkConvergenceReceivers()

	return c, nil
}

// SetRoutingAlgorithm overwrites the used RoutingAlgorithm, which defaults to
// EpidemicRouting.
func (c *Core) SetRoutingAlgorithm(routing RoutingAlgorithm) {
	c.routing = routing
}

// checkConvergenceReceivers checks all ConvergenceReceivers for new bundles.
func (c *Core) checkConvergenceReceivers() {
	var chnl = cla.JoinReceivers()
	var tick = time.NewTicker(30 * time.Second)

	for {
		select {
		// Invoked by Close(), shuts down
		case <-c.stopSyn:
			tick.Stop()

			c.convergenceMutex.Lock()
			for _, claRec := range c.convergenceReceivers {
				claRec.Close()
			}
			c.convergenceMutex.Unlock()

			if storeErr := c.store.Close(); storeErr != nil {
				log.WithFields(log.Fields{
					"error": storeErr,
				}).Warn("Closing store while shutting down the Core errors")
			}

			close(c.stopAck)
			return

		// Handle a received bundle, also checks if the channel is open
		case bndl := <-chnl:
			c.receive(NewRecBundlePack(bndl))

		// Check back on enqueued CLAs and contraindicated bundles
		case <-tick.C:
			// CLAs
			var tmpQueue = make([]*convergenceQueueElement, len(c.convergenceQueue))
			var delQueue = make([]*convergenceQueueElement, 0, 0)

			c.convergenceMutex.Lock()
			copy(tmpQueue, c.convergenceQueue)
			c.convergenceMutex.Unlock()

			for _, cqe := range tmpQueue {
				log.WithFields(log.Fields{
					"cla": cqe.conv,
				}).Debug("Getting CLA from queue")

				if retry := cqe.activate(c); !retry {
					delQueue = append(delQueue, cqe)

					log.WithFields(log.Fields{
						"cla": cqe.conv,
					}).Debug("Removing CLA from queue")
				} else {
					log.WithFields(log.Fields{
						"cla": cqe.conv,
					}).Debug("CLA stays in the queue")
				}
			}

			c.convergenceMutex.Lock()
			for _, cqe := range delQueue {
				for i := len(c.convergenceQueue) - 1; i >= 0; i-- {
					if cqe == c.convergenceQueue[i] {
						c.convergenceQueue = append(
							c.convergenceQueue[:i], c.convergenceQueue[i+1:]...)
					}
				}
			}
			c.convergenceMutex.Unlock()

			// Bundles
			for _, bp := range QueryPending(c.store) {
				log.WithFields(log.Fields{
					"bundle": bp.Bundle,
				}).Info("Retrying bundle from store")

				c.dispatching(bp)
			}

		// Invoked by RegisterConvergenceReceiver, recreates chnl
		case <-c.reloadConvRecs:
			c.convergenceMutex.Lock()
			chnl = cla.JoinReceivers()
			for _, claRec := range c.convergenceReceivers {
				chnl = cla.JoinReceivers(chnl, claRec.Channel())
			}
			c.convergenceMutex.Unlock()
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
func (c *Core) senderForDestination(endpoint bundle.EndpointID) []cla.ConvergenceSender {
	var css []cla.ConvergenceSender

	c.convergenceMutex.Lock()
	for _, cs := range c.convergenceSenders {
		if cs.GetPeerEndpointID() == endpoint {
			css = append(css, cs)
		}
	}
	c.convergenceMutex.Unlock()

	return css
}

// hasEndpoint checks if this Core has some endpoint, but does not secure this
// request with a Mutex. Therefore, the safe HasEndpoint method exists.
func (c *Core) hasEndpoint(endpoint bundle.EndpointID) bool {
	for _, agent := range c.Agents {
		if agent.EndpointID() == endpoint {
			return true
		}
	}

	for _, rec := range c.convergenceReceivers {
		if rec.GetEndpointID() == endpoint {
			return true
		}
	}

	return false
}

// HasEndpoint returns true if the given endpoint ID is assigned either to an
// application or a CLA governed by this Application Agent.
func (c *Core) HasEndpoint(endpoint bundle.EndpointID) (state bool) {
	c.convergenceMutex.Lock()
	state = c.hasEndpoint(endpoint)
	c.convergenceMutex.Unlock()

	return
}

// SendStatusReport creates a new status report in response to the given
// BundlePack and transmits it.
func (c *Core) SendStatusReport(bp BundlePack,
	status StatusInformationPos, reason StatusReportReason) {
	// Don't repond to other administrative records
	if bp.Bundle.PrimaryBlock.BundleControlFlags.Has(bundle.AdministrativeRecordPayload) {
		return
	}

	// Don't respond to ourself
	if c.HasEndpoint(bp.Bundle.PrimaryBlock.ReportTo) {
		return
	}

	log.WithFields(log.Fields{
		"bundle": bp.Bundle,
		"status": status,
		"reason": reason,
	}).Info("Sending a status report for a bundle")

	var inBndl = *bp.Bundle
	var sr = NewStatusReport(inBndl, status, reason, bundle.DtnTimeNow())
	var ar = NewAdministrativeRecord(BundleStatusReportTypeCode, sr)

	var aaEndpoint = bp.Receiver
	if !c.HasEndpoint(aaEndpoint) {
		log.WithFields(log.Fields{
			"bundle":   bp.Bundle,
			"endpoint": aaEndpoint,
		}).Warn("Failed to create status report, receiver is not a current endpoint")

		return
	}

	var primary = bundle.NewPrimaryBlock(
		bundle.AdministrativeRecordPayload,
		inBndl.PrimaryBlock.ReportTo,
		aaEndpoint,
		bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0),
		60*60*1000000)

	var outBndl, err = bundle.NewBundle(
		primary,
		[]bundle.CanonicalBlock{
			bundle.NewHopCountBlock(23, 0, bundle.NewHopCount(5)),
			ar.ToCanonicalBlock(),
		})

	if err != nil {
		log.WithFields(log.Fields{
			"bundle": bp.Bundle,
			"error":  err,
		}).Warn("Creating status report bundle failed")

		return
	}

	c.transmit(NewBundlePack(outBndl))
}
