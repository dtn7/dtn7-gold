package core

import (
	"log"

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

// ProtocolAgent is the Bundle Protocol Agent (BPA) which handles transmission
// and reception of bundles.
type Core struct {
	ConvergenceSenders   []cla.ConvergenceSender
	ConvergenceReceivers []cla.ConvergenceReceiver
	AppEndpoints         []bundle.EndpointID
}

func (c *Core) RegisterConvergenceSender(sender cla.ConvergenceSender) {
	c.ConvergenceSenders = append(c.ConvergenceSenders, sender)
}

func (c *Core) RegisterConvergenceReceiver(rec cla.ConvergenceReceiver) {
	c.ConvergenceReceivers = append(c.ConvergenceReceivers, rec)

	go func() {
		var chnl = rec.Channel()
		for {
			select {
			case bndl := <-chnl:
				c.Receive(NewBundlePack(bndl))
			}
		}
	}()
}

func (c Core) clasForDestination(endpoint bundle.EndpointID) []cla.ConvergenceSender {
	var clas []cla.ConvergenceSender

	for _, cla := range c.ConvergenceSenders {
		if cla.GetPeerEndpointID() == endpoint {
			clas = append(clas, cla)
		}
	}

	return clas
}

func (c Core) clasForBudlePack(bp BundlePack) []cla.ConvergenceSender {
	// TODO: This software is kind of stupid at this moment and will return all
	// currently known CLAs.

	return c.ConvergenceSenders
}

// HasEndpoint returns true if the given endpoint ID is assigned either to an
// application or a CLA governed by this Application Agent.
func (c Core) HasEndpoint(endpoint bundle.EndpointID) bool {
	for _, ep := range c.AppEndpoints {
		if ep == endpoint {
			return true
		}
	}

	for _, rec := range c.ConvergenceReceivers {
		if rec.GetEndpointID() == endpoint {
			return true
		}
	}

	return false
}

// SendStatusReport creates a new status report in response to the given
// BundlePack and transmits it.
func (c Core) SendStatusReport(bp BundlePack,
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

	// TODO change this
	var aaEndpoint = c.AppEndpoints[0]

	var primary = bundle.NewPrimaryBlock(
		bundle.AdministrativeRecordPayload,
		inBndl.PrimaryBlock.ReportTo,
		aaEndpoint,
		bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0),
		60*60*1000000)

	var outBndl, err = bundle.NewBundle(
		primary,
		[]bundle.CanonicalBlock{
			ar.ToCanonicalBlock(),
		})

	if err != nil {
		log.Printf("Creating status report bundle regarding %v failed: %v",
			bp.Bundle, err)

		return
	}

	c.Transmit(NewBundlePack(outBndl))
}
