// SPDX-FileCopyrightText: 2019 Markus Sommer
// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package core

import (
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/storage"
	log "github.com/sirupsen/logrus"
)

// RoutingAlgorithm is an interface to specify routing algorithms for
// delay-tolerant networks. An implementation might store a reference to a Core
// struct to refer the ConvergenceSenders.
type RoutingAlgorithm interface {
	// NotifyIncoming notifies this RoutingAlgorithm about incoming bundles.
	// Whether the algorithm acts on this information or ignores it, is both a
	// design and implementation decision.
	NotifyIncoming(bp BundlePack)

	// DispatchingAllowed will be called from within the *dispatching* step of
	// the processing pipeline. A RoutingAlgorithm is allowed to drop the
	// proceeding of a bundle before being inspected further or being delivered
	// locally or to another node.
	DispatchingAllowed(bp BundlePack) bool

	// SenderForBundle returns an array of ConvergenceSender for a requested
	// bundle. Furthermore the delete flags indicates if this BundlePack should
	// be deleted afterwards.
	// The CLA selection is based on the algorithm's design.
	SenderForBundle(bp BundlePack) (sender []cla.ConvergenceSender, delete bool)

	// ReportFailure notifies the RoutingAlgorithm about a failed transmission to
	// a previously selected CLA. Compare: SenderForBundle.
	ReportFailure(bp BundlePack, sender cla.ConvergenceSender)

	// ReportPeerAppeared notifies the RoutingAlgorithm about a new neighbor.
	ReportPeerAppeared(peer cla.Convergence)

	// ReportPeerDisappeared notifies the RoutingAlgorithm about the
	// disappearance of a neighbor.
	ReportPeerDisappeared(peer cla.Convergence)
}

// RoutingConf contains necessary configuration data to initialize a routing algorithm.
type RoutingConf struct {
	// Algorithm is one of the implemented routing algorithms.
	//
	// One of: "epidemic", "spray", "binary_spray", "dtlsr", "prophet"
	Algorithm string

	// SprayConf contains data to initialize "spray" or "binary_spray"
	SprayConf SprayConfig

	// DTLSRConf contains data to initialize "dtlsr"
	DTLSRConf DTLSRConfig

	// ProphetConf contains data to initialize "prophet"
	ProphetConf ProphetConfig
}

// sendMetadataBundle can be used by routing algorithm to send relevant metadata to peers
// Metadata needs to be serialised as an ExtensionBlock
func sendMetadataBundle(c *Core, source bundle.EndpointID, destination bundle.EndpointID, metadataBlock bundle.ExtensionBlock) error {
	bundleBuilder := bundle.Builder()
	bundleBuilder.Source(source)
	bundleBuilder.Destination(destination)
	bundleBuilder.CreationTimestampNow()
	bundleBuilder.Lifetime("1m")
	bundleBuilder.BundleCtrlFlags(bundle.MustNotFragmented)
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
func filterCLAs(bundleItem storage.BundleItem, clas []cla.ConvergenceSender, algorithm string) (filtered []cla.ConvergenceSender, sentEids []bundle.EndpointID) {
	filtered = make([]cla.ConvergenceSender, 0)

	sentEids, ok := bundleItem.Properties["routing/"+algorithm+"/sent"].([]bundle.EndpointID)
	if !ok {
		sentEids = make([]bundle.EndpointID, 0)
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
