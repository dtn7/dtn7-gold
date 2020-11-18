// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/storage"
)

type Pipeline struct {
	NodeId bpv7.EndpointID

	Store        *storage.Store
	Algorithm    Algorithm
	AgentManager AgentManager

	// Checks to be passed in the initial processing.
	Checks []CheckFunc

	// SendReports only for the allowed bpv7.StatusInformationPos.
	SendReports map[bpv7.StatusInformationPos]bool

	queue chan pipelineMsg
}

type pipelineMsgType int

const (
	_ pipelineMsgType = iota

	// sendBundle message with a bpv7.Bundle payload.
	sendBundle
	// receiveBundle message with a bpv7.Bundle payload.
	receiveBundle
)

type pipelineMsg struct {
	t pipelineMsgType
	v interface{}
}

type pipelineFunc func(BundleDescriptor) pipelineFunc

func (pipeline *Pipeline) Start() {
	pipeline.Checks = []CheckFunc{CheckRouting, CheckLifetime, CheckHopCount}
	pipeline.SendReports = map[bpv7.StatusInformationPos]bool{
		bpv7.ReceivedBundle:  true,
		bpv7.ForwardedBundle: false,
		bpv7.DeliveredBundle: true,
		bpv7.DeletedBundle:   true,
	}
	pipeline.queue = make(chan pipelineMsg)

	go pipeline.run()
}

func (pipeline *Pipeline) run() {
	// TODO: this is currently just a mockup
runLoop:
	for msg := range pipeline.queue {
		var startFunc pipelineFunc
		switch msg.t {
		case sendBundle:
			startFunc = pipeline.sendOutgoing
		case receiveBundle:
			startFunc = pipeline.receiveIncoming
		default:
			continue runLoop
		}

		descriptor := NewBundleDescriptorFromBundle(msg.v.(bpv7.Bundle), pipeline.Store)
		for f := startFunc; !descriptor.HasTag(Faulty) && f != nil; f = f(descriptor) {
		}
	}
}

// log an event.
func (pipeline *Pipeline) log() *log.Entry {
	return log.WithField("pipeline", pipeline.NodeId)
}

// sendReport creates a StatusReport. No action will be performed if the internal settings are permitting it.
func (pipeline *Pipeline) sendReport(descriptor BundleDescriptor, status bpv7.StatusInformationPos, reason bpv7.StatusReportReason) {
	// Don't report if..
	// - not enabled for this status information.
	// - it's an outgoing bundle.
	// - the bundle is an administrative record itself.
	if ok, exists := pipeline.SendReports[status]; !ok || !exists {
		return
	} else if descriptor.HasTag(Outgoing) {
		return
	} else if descriptor.MustBundle().PrimaryBlock.BundleControlFlags.Has(bpv7.AdministrativeRecordPayload) {
		return
	}

	pipeline.log().WithFields(log.Fields{
		"origin": descriptor.ID(), "status": status, "reason": reason,
	}).Info("creating status report for bundle")

	reportBundle, err := bpv7.Builder().
		CRC(bpv7.CRC32).
		Source(pipeline.NodeId).
		Destination(descriptor.MustBundle().PrimaryBlock.ReportTo).
		CreationTimestampNow().
		Lifetime(descriptor.MustBundle().PrimaryBlock.Lifetime).
		StatusReport(descriptor.MustBundle(), status, reason).
		Build()
	if err != nil {
		pipeline.log().WithField("origin", descriptor.ID()).WithError(err).Warn("status report creation errored")
	} else {
		pipeline.queue <- pipelineMsg{t: sendBundle, v: reportBundle}
	}
}
