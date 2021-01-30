// SPDX-FileCopyrightText: 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// pipeline_pre contains common Pipeline code for preprocessing incoming and outgoing bundles.

// receiveIncoming is the first action for received / incoming bundles.
func (pipeline *Pipeline) receiveIncoming(descriptor BundleDescriptor) pipelineFunc {
	descriptor.AddTag(Incoming)
	return pipeline.receiveCheckBlocks
}

// receiveCheckBlocks checks incoming block's canonical blocks.
func (pipeline *Pipeline) receiveCheckBlocks(descriptor BundleDescriptor) pipelineFunc {
	var sendReport bool

	for i := len(descriptor.MustBundle().CanonicalBlocks) - 1; i >= 0; i-- {
		var canonical = descriptor.MustBundle().CanonicalBlocks[i]

		if bpv7.GetExtensionBlockManager().IsKnown(canonical.TypeCode()) {
			continue
		}

		if canonical.BlockControlFlags.Has(bpv7.DeleteBundle) {
			pipeline.log().WithFields(log.Fields{
				"bundle": descriptor.ID(), "block": canonical.BlockNumber,
			}).Info("deleting bundle because of an unknown block")

			descriptor.AddTag(Faulty)
			break
		} else if canonical.BlockControlFlags.Has(bpv7.RemoveBlock) {
			pipeline.log().WithFields(log.Fields{
				"bundle": descriptor.ID(), "block": canonical.BlockNumber,
			}).Info("removing unknown block")

			descriptor.MustBundle().RemoveExtensionBlockByBlockNumber(canonical.BlockNumber)
		}
		if canonical.BlockControlFlags.Has(bpv7.StatusReportBlock) {
			sendReport = true
		}
	}

	if sendReport {
		pipeline.sendReport(descriptor, bpv7.ReceivedBundle, bpv7.BlockUnsupported)
	} else if descriptor.MustBundle().PrimaryBlock.BundleControlFlags.Has(bpv7.StatusRequestReception) {
		pipeline.sendReport(descriptor, bpv7.ReceivedBundle, bpv7.NoInformation)
	}

	return pipeline.processInitial
}

// sendOutgoing is the first action for sent / outgoing bundles.
func (pipeline *Pipeline) sendOutgoing(descriptor BundleDescriptor) pipelineFunc {
	descriptor.AddTag(Outgoing)
	return pipeline.processInitial
}

// processInitial checks all BundleDescriptors before deciding to drop, forward or local deliver.
func (pipeline *Pipeline) processInitial(descriptor BundleDescriptor) pipelineFunc {
	pipeline.Algorithm.NotifyNewBundle(descriptor)

	for _, checkFunc := range pipeline.Checks {
		if err := checkFunc(pipeline, descriptor); err != nil {
			pipeline.log().WithField("bundle", descriptor.ID()).WithError(err).Warn("preprocessing check failed")

			descriptor.AddTag(Faulty)
			break
		}
	}

	if pipeline.NodeId.SameNode(descriptor.Receiver) {
		// TODO better check if this bundle is addressed to this node
		return pipeline.localDelivery
	} else {
		// TODO implement forwarding logic
		return nil
	}
}
