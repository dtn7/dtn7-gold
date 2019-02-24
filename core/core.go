package core

import (
	"log"
	"sync"
	"time"

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
	Agents []ApplicationAgent

	convergenceSenders   []cla.ConvergenceSender
	convergenceReceivers []cla.ConvergenceReceiver
	convergenceMutex     sync.Mutex

	idKeeper IdKeeper
	store    Store
	routing  RoutingAlgorithm

	reloadConvRecs chan struct{}
	stopSyn        chan struct{}
	stopAck        chan struct{}
}

// NewCore creates and returns a new core.
func NewCore(storePath string) (*Core, error) {
	var c = new(Core)

	store, err := NewSimpleStore(storePath)
	if err != nil {
		return nil, err
	}
	c.store = store

	c.idKeeper = NewIdKeeper()
	c.reloadConvRecs = make(chan struct{})

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

			close(c.stopAck)
			return

		// Handle a received bundle, also checks if the channel is open
		case bndl, ok := <-chnl:
			if ok {
				c.receive(NewRecBundlePack(bndl))
			}

		// Check back on contraindicated bundles
		case <-tick.C:
			for _, bp := range QueryPending(c.store) {
				log.Printf("Retrying bundle %v from store", bp)
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

// RegisterConvergenceSender adds a new ConvergenceSender to this Core's list.
// Bundles will be sent through this ConvergenceSender.
func (c *Core) RegisterConvergenceSender(sender cla.ConvergenceSender) {
	c.convergenceMutex.Lock()
	for _, cs := range c.convergenceSenders {
		if cs.Address() == sender.Address() {
			log.Printf("ConvergenceSender's address is already known: %v", sender)
			c.convergenceMutex.Unlock()
			return
		}
	}
	c.convergenceMutex.Unlock()

	if c.HasEndpoint(sender.GetPeerEndpointID()) {
		log.Printf("Node contains ConvergenceSender's endpoint ID: %v", sender)
		return
	}

	if err, retry := sender.Start(); err != nil {
		log.Printf("Failed to start ConvergenceSender %v: %v", sender, err)

		if retry {
			go func(sender cla.ConvergenceSender) {
				time.Sleep(5 * time.Second)
				c.RegisterConvergenceSender(sender)
			}(sender)
		}
	} else {
		log.Printf("Started ConvergenceSender %v", sender)

		c.convergenceMutex.Lock()
		c.convergenceSenders = append(c.convergenceSenders, sender)
		c.convergenceMutex.Unlock()
	}
}

// RegisterConvergenceReceiver adds a new ConvergenceReceiver to this Core's
// list. Bundles will be received through this ConvergenceReceiver
func (c *Core) RegisterConvergenceReceiver(rec cla.ConvergenceReceiver) {
	if err, retry := rec.Start(); err != nil {
		log.Printf("Failed to start ConvergenceReceiver %v: %v", rec, err)

		if retry {
			go func(rec cla.ConvergenceReceiver) {
				time.Sleep(5 * time.Second)
				c.RegisterConvergenceReceiver(rec)
			}(rec)
		}
	} else {
		log.Printf("Started ConvergenceReceiver %v", rec)

		c.convergenceMutex.Lock()
		c.convergenceReceivers = append(c.convergenceReceivers, rec)
		c.convergenceMutex.Unlock()

		c.reloadConvRecs <- struct{}{}
	}
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

// HasEndpoint returns true if the given endpoint ID is assigned either to an
// application or a CLA governed by this Application Agent.
func (c *Core) HasEndpoint(endpoint bundle.EndpointID) bool {
	for _, agent := range c.Agents {
		if agent.EndpointID() == endpoint {
			return true
		}
	}

	c.convergenceMutex.Lock()
	for _, rec := range c.convergenceReceivers {
		if rec.GetEndpointID() == endpoint {
			c.convergenceMutex.Unlock()
			return true
		}
	}
	c.convergenceMutex.Unlock()

	return false
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

	log.Printf("Creation of a %v \"%v\" status report regarding %v",
		status, reason, bp.Bundle)

	var inBndl = *bp.Bundle
	var sr = NewStatusReport(inBndl, status, reason, bundle.DtnTimeNow())
	var ar = NewAdministrativeRecord(BundleStatusReportTypeCode, sr)

	var aaEndpoint = bp.Receiver
	if !c.HasEndpoint(aaEndpoint) {
		log.Printf(
			"Failed to create status report for %v, receiver %v is not a current endpoint",
			bp.Bundle, aaEndpoint)

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
		log.Printf("Creating status report bundle regarding %v failed: %v",
			bp.Bundle, err)

		return
	}

	c.transmit(NewBundlePack(outBndl))
}
