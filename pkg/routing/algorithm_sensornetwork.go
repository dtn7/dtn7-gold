// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"fmt"
	"regexp"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/pkg/cla"
)

// SensorNetworkMuleRouting is a simple proxy routing algorithm for data mules in specific sensor networks.
//
// This type of sensor network is constructed in such a way that several unconnected sensors exist in an area. In
// addition, there is a permanently connected server to which sensor data should be sent. For the connection there are
// data mules, which are travelling between sensors and server.
//
// Addressing within the network is based on the "dtn" URI scheme, compare bpv7.DtnEndpoint. The role of the node can
// be determined from the node name, e.g., "dtn://tree23.sensor/" is a sensor. The node naming syntax is configurable.
//
// This algorithm is exclusively for data mules. No completely new algorithm is defined here, but an existing one is
// used, such as epidemic routing. The only difference is that the selection of the receiving nodes is limited so that
// sensors do not receive other sensors' data. A data mule therefore only receives data from sensors or forwards
// bundles to sensors that are addressed to them, such as receipt confirmations as administrative records.
type SensorNetworkMuleRouting struct {
	algorithm  Algorithm
	sensorNode *regexp.Regexp
}

// SensorNetworkMuleConfig describes a SensorNetworkMuleRouting.
type SensorNetworkMuleConfig struct {
	Algorithm       *RoutingConf `toml:"routing"`
	SensorNodeRegex string       `toml:"sensor-node-regex"`
}

// NewSensorNetworkMuleRouting based on an underlying algorithm and a regex to identify sensor nodes by their Node ID.
func NewSensorNetworkMuleRouting(algorithm Algorithm, sensorNode *regexp.Regexp) *SensorNetworkMuleRouting {
	return &SensorNetworkMuleRouting{
		algorithm:  algorithm,
		sensorNode: sensorNode,
	}
}

// NotifyNewBundle will be handled by the underlying algorithm.
func (snm *SensorNetworkMuleRouting) NotifyNewBundle(bp BundleDescriptor) {
	snm.algorithm.NotifyNewBundle(bp)
}

// DispatchingAllowed if the underlying algorithm says so.
func (snm *SensorNetworkMuleRouting) DispatchingAllowed(bp BundleDescriptor) bool {
	return snm.algorithm.DispatchingAllowed(bp)
}

// SenderForBundle queries the underlying algorithm and optionally filters the result.
func (snm *SensorNetworkMuleRouting) SenderForBundle(bp BundleDescriptor) (sender []cla.ConvergenceSender, delete bool) {
	sender, delete = snm.algorithm.SenderForBundle(bp)
	log.WithField("convergence-senders", sender).Debug("Sensor Mule's algorithm selected peers")

	// Filter sender list: Remove sensor nodes iff a bundle is not addressed to it.
	for i := len(sender) - 1; i >= 0; i-- {
		logger := log.WithFields(log.Fields{
			"bundle":             bp.ID(),
			"convergence-sender": sender[i],
		})

		// If this ConvergenceSender is not a sensor, do not exclude it.
		if !snm.sensorNode.MatchString(sender[i].GetPeerEndpointID().String()) {
			logger.Debug("Convergence Sender's Node ID does not match a Sensor Mule's sensor mask")
			continue
		}

		// Otherwise, check if this Bundle is not addressed to it.
		if !sender[i].GetPeerEndpointID().SameNode(bp.Receiver) {
			logger.Info("Sensor Mule excludes Sensor-Convergence Sender for Bundle delivery")

			snm.algorithm.ReportFailure(bp, sender[i])
			sender = append(sender[:i], sender[i+1:]...)

			continue
		}

		logger.Info("Sensor Mule allows Bundle delivery to Sensor-Convergence Sender")
	}

	// Optionally reset delete flag
	// Generally speaking, this is hard to decide without context. Thus, reset the flag if the sender array is empty.
	if delete && len(sender) == 0 {
		delete = false
	}

	return
}

// ReportFailure back to the underlying algorithm.
func (snm *SensorNetworkMuleRouting) ReportFailure(bp BundleDescriptor, sender cla.ConvergenceSender) {
	snm.algorithm.ReportFailure(bp, sender)
}

// ReportPeerAppeared to the underlying algorithm.
func (snm *SensorNetworkMuleRouting) ReportPeerAppeared(peer cla.Convergence) {
	snm.algorithm.ReportPeerAppeared(peer)
}

// ReportPeerDisappeared to the underlying algorithm.
func (snm *SensorNetworkMuleRouting) ReportPeerDisappeared(peer cla.Convergence) {
	snm.algorithm.ReportPeerDisappeared(peer)
}

func (snm *SensorNetworkMuleRouting) String() string {
	return fmt.Sprintf("sensor mule overlaying %v", snm.algorithm)
}
