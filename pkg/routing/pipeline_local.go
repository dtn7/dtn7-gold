// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import "github.com/dtn7/dtn7-go/pkg/bpv7"

// pipeline_local contains common Pipeline code for processing bundles addressed to local endpoints.

// localDelivery dispatches bundles to be delivered locally.
func (pipeline *Pipeline) localDelivery(descriptor BundleDescriptor) pipelineFunc {
	isFragment := descriptor.MustBundle().PrimaryBlock.BundleControlFlags.Has(bpv7.IsFragment)
	isNodeRecord := descriptor.MustBundle().PrimaryBlock.BundleControlFlags.Has(bpv7.AdministrativeRecordPayload) &&
		descriptor.MustBundle().PrimaryBlock.Destination == pipeline.NodeId

	if isFragment {
		return pipeline.localFragment
	} else if isNodeRecord {
		return pipeline.localNodeRecord
	} else if pipeline.AgentManager.HasEndpoint(descriptor.MustBundle().PrimaryBlock.Destination) {
		if err := pipeline.AgentManager.Deliver(descriptor); err == nil {
			pipeline.log().WithField("bundle", descriptor.ID()).Info("delivered bundle to a local agent")
			descriptor.AddTag(Delivered)
		} else {
			pipeline.log().WithError(err).WithField("bundle", descriptor.ID()).Warn("local delivery failed")
			descriptor.AddTag(NoLocalAgent)
		}

		return nil
	} else {
		pipeline.log().WithField("bundle", descriptor.ID()).Info("no local agent for incoming bundle")
		descriptor.AddTag(NoLocalAgent)
		return nil
	}
}

// localFragment handles fragmented bundles addressed to this node.
func (pipeline *Pipeline) localFragment(descriptor BundleDescriptor) pipelineFunc {
	pipeline.log().WithField("bundle", descriptor.ID()).Info("received fragmented bundle")
	descriptor.AddTag(ReassemblyPending)

	// TODO
	return nil
}

// localNodeRecord handles administrative records addressed to this node's node id.
func (pipeline *Pipeline) localNodeRecord(descriptor BundleDescriptor) pipelineFunc {
	if adminRecord, err := descriptor.MustBundle().AdministrativeRecord(); err != nil {
		pipeline.log().WithError(err).WithField("bundle", descriptor.ID()).Warn("no administrative record is present")
		return nil
	} else if adminRecord.RecordTypeCode() == bpv7.AdminRecordTypeStatusReport {
		return pipeline.localNodeStatusReport
	} else {
		pipeline.log().WithField("bundle", descriptor.ID()).Warn("unsupported administrative record")
		return nil
	}
}

// localNodeStatusReport handles status reports addressed to this node's node id.
func (pipeline *Pipeline) localNodeStatusReport(descriptor BundleDescriptor) pipelineFunc {
	// adminRecord, err := descriptor.MustBundle().AdministrativeRecord()
	// TODO
	return nil
}
