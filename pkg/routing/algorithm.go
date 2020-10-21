// SPDX-FileCopyrightText: 2019 Markus Sommer
// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routing

import (
	"fmt"
	"regexp"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla"
	"github.com/dtn7/dtn7-go/pkg/storage"

	log "github.com/sirupsen/logrus"
)

// Algorithm is an interface to specify routing algorithms for delay-tolerant networks.
type Algorithm interface {
	// NotifyNewBundle notifies this Algorithm about new bundles. They
	// might be generated at this node or received from a peer. Whether an
	// algorithm acts on this information or ignores it, is implementation matter.
	NotifyNewBundle(descriptor BundleDescriptor)

	// DispatchingAllowed will be called from within the *dispatching* step of
	// the processing pipeline. A Algorithm is allowed to drop the
	// proceeding of a bundle before being inspected further or being delivered
	// locally or to another node.
	DispatchingAllowed(descriptor BundleDescriptor) bool

	// SenderForBundle returns an array of ConvergenceSender for a requested
	// bundle. Furthermore the delete flags indicates if this BundleDescriptor should
	// be deleted afterwards.
	// The CLA selection is based on the algorithm's design.
	SenderForBundle(descriptor BundleDescriptor) (sender []cla.ConvergenceSender, delete bool)

	// ReportFailure notifies the Algorithm about a failed transmission to
	// a previously selected CLA. Compare: SenderForBundle.
	ReportFailure(descriptor BundleDescriptor, sender cla.ConvergenceSender)

	// ReportPeerAppeared notifies the Algorithm about a new neighbor.
	ReportPeerAppeared(peer cla.Convergence)

	// ReportPeerDisappeared notifies the Algorithm about the
	// disappearance of a neighbor.
	ReportPeerDisappeared(peer cla.Convergence)
}

// RoutingConf contains necessary configuration data to initialize a routing algorithm.
type RoutingConf struct {
	// Algorithm is one of the implemented routing algorithms.
	//
	// One of: "epidemic", "spray", "binary_spray", "dtlsr", "prophet", "sensor-mule"
	Algorithm string

	// SprayConf contains data to initialize "spray" or "binary_spray"
	SprayConf SprayConfig

	// DTLSRConf contains data to initialize "dtlsr"
	DTLSRConf DTLSRConfig

	// ProphetConf contains data to initialize "prophet"
	ProphetConf ProphetConfig

	// SensorNetworkMuleConfig contains data to initialize "sensor-mule"
	SensorMuleConf SensorNetworkMuleConfig `toml:"sensor-mule-conf"`
}

// RoutingAlgorithm from its configuration.
func (routingConf RoutingConf) RoutingAlgorithm(c *Core) (algo Algorithm, err error) {
	switch routingConf.Algorithm {
	case "epidemic":
		algo = NewEpidemicRouting(c)

	case "spray":
		algo = NewSprayAndWait(c, routingConf.SprayConf)

	case "binary_spray":
		algo = NewBinarySpray(c, routingConf.SprayConf)

	case "dtlsr":
		algo = NewDTLSR(c, routingConf.DTLSRConf)

	case "prophet":
		algo = NewProphet(c, routingConf.ProphetConf)

	case "sensor-mule":
		if muleAlgo, muleAlgoErr := routingConf.SensorMuleConf.Algorithm.RoutingAlgorithm(c); muleAlgoErr != nil {
			err = muleAlgoErr
		} else if sensorNode, sensorNodeErr := regexp.Compile(routingConf.SensorMuleConf.SensorNodeRegex); sensorNodeErr != nil {
			err = sensorNodeErr
		} else {
			algo = NewSensorNetworkMuleRouting(muleAlgo, sensorNode)
		}

	default:
		err = fmt.Errorf("unknown routing algorithm %s", routingConf.Algorithm)
	}

	return
}

// sendMetadataBundle can be used by routing algorithm to send relevant metadata to peers
// Metadata needs to be serialised as an ExtensionBlock
func sendMetadataBundle(c *Core, source bpv7.EndpointID, destination bpv7.EndpointID, metadataBlock bpv7.ExtensionBlock) error {
	bundleBuilder := bpv7.Builder()
	bundleBuilder.Source(source)
	bundleBuilder.Destination(destination)
	bundleBuilder.CreationTimestampNow()
	bundleBuilder.Lifetime("1m")
	bundleBuilder.BundleCtrlFlags(bpv7.MustNotFragmented)
	// no Payload
	bundleBuilder.PayloadBlock(byte(1))

	bundleBuilder.Canonical(metadataBlock)
	metadataBundle, err := bundleBuilder.Build()
	if err != nil {
		return err
	} else {
		log.Debug("Metadata Bundle built")
	}

	log.Debug("Sending metadata bundle")
	c.SendBundle(&metadataBundle)
	log.WithFields(log.Fields{
		"bundle": metadataBundle,
	}).Debug("Successfully sent metadata bundle")

	return nil
}

// filterCLAs filters the nodes which already received a Bundle for a specific routing algorithm, e.g., "epidemic".
// It returns a list of unused ConvergenceSenders and an updated list of all sent EndpointIDs. The second should be
// stored as "routing/${algorithm}/sent" within the specific algorithm.
func filterCLAs(bundleItem storage.BundleItem, clas []cla.ConvergenceSender, algorithm string) (filtered []cla.ConvergenceSender, sentEids []bpv7.EndpointID) {
	filtered = make([]cla.ConvergenceSender, 0)

	sentEids, ok := bundleItem.Properties["routing/"+algorithm+"/sent"].([]bpv7.EndpointID)
	if !ok {
		sentEids = make([]bpv7.EndpointID, 0)
	}

	for _, cs := range clas {
		skip := false

		for _, eid := range sentEids {
			if cs.GetPeerEndpointID() == eid {
				skip = true
				break
			}
		}

		if !skip {
			filtered = append(filtered, cs)
			sentEids = append(sentEids, cs.GetPeerEndpointID())
		}
	}

	return
}
