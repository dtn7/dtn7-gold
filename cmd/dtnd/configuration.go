package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/BurntSushi/toml"
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/cla/mtcp"
	"github.com/dtn7/dtn7-go/core"
	"github.com/dtn7/dtn7-go/discovery"
)

// tomlConfig describes the TOML-configuration.
type tomlConfig struct {
	Core       coreConf
	Logging    logConf
	Discovery  discoveryConf
	SimpleRest simpleRestConf `toml:"simple-rest"`
	Listen     []convergenceConf
	Peer       []convergenceConf
	// Routing is one of the implemented routing-algorithms
	// May be: "epidemic", "spray", "binary_spray"
	Routing    string
}

// coreConf describes the Core-configuration block.
type coreConf struct {
	Store             string
	InspectAllBundles bool   `toml:"inspect-all-bundles"`
	NodeId            string `toml:"node-id"`
}

// logConf describes the Logging-configuration block.
type logConf struct {
	Level        string
	ReportCaller bool `toml:"report-caller"`
	Format       string
}

// discoveryConf describes the Discovery-configuration block.
type discoveryConf struct {
	IPv4     bool
	IPv6     bool
	Interval uint
}

// simpleRestConf describes the SimpleRESTAppAgent.
type simpleRestConf struct {
	Node   string
	Listen string
}

// convergenceConf describes the Convergence-configuration block, used for
// "listen" and "peer".
type convergenceConf struct {
	Node     string
	Protocol string
	Endpoint string
}

// parseListen inspects a "listen" convergenceConf and returns a ConvergenceReceiver.
func parseListen(conv convergenceConf, nodeId bundle.EndpointID) (cla.ConvergenceReceiver, discovery.DiscoveryMessage, error) {
	var defaultDisc = discovery.DiscoveryMessage{}

	switch conv.Protocol {
	case "mtcp":
		_, portStr, _ := net.SplitHostPort(conv.Endpoint)
		portInt, _ := strconv.Atoi(portStr)

		msg := discovery.DiscoveryMessage{
			Type:     discovery.MTCP,
			Endpoint: nodeId,
			Port:     uint(portInt),
		}

		return mtcp.NewMTCPServer(conv.Endpoint, nodeId, true), msg, nil

	default:
		return nil, defaultDisc, fmt.Errorf("Unknown listen.protocol \"%s\"", conv.Protocol)
	}
}

func parsePeer(conv convergenceConf) (cla.ConvergenceSender, error) {
	switch conv.Protocol {
	case "mtcp":
		endpointID, err := bundle.NewEndpointID(conv.Node)
		if err != nil {
			return nil, err
		}

		return mtcp.NewMTCPClient(conv.Endpoint, endpointID, true), nil

	default:
		return nil, fmt.Errorf("Unknown peer.protocol \"%s\"", conv.Protocol)
	}
}

func parseSimpleRESTAppAgent(conf simpleRestConf, c *core.Core) (core.ApplicationAgent, error) {
	endpointID, err := bundle.NewEndpointID(conf.Node)
	if err != nil {
		return nil, err
	}

	return core.NewSimpleRESTAppAgent(endpointID, c, conf.Listen), nil
}

// parseCore creates the Core based on the given TOML configuration.
func parseCore(filename string) (c *core.Core, ds *discovery.DiscoveryService, err error) {
	var conf tomlConfig
	if _, err = toml.DecodeFile(filename, &conf); err != nil {
		return
	}

	// Logging
	if conf.Logging.Level != "" {
		if lvl, err := log.ParseLevel(conf.Logging.Level); err != nil {
			log.WithFields(log.Fields{
				"level":    conf.Logging.Level,
				"error":    err,
				"provided": "panic,fatal,error,warn,info,debug,trace",
			}).Warn("Failed to set log level. Please select one of the provided ones")
		} else {
			log.SetLevel(lvl)
		}
	}

	log.SetReportCaller(conf.Logging.ReportCaller)

	switch conf.Logging.Format {
	case "", "text":
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "15:04:05.000",
		})

	case "json":
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})

	default:
		log.Warn("Unknown logging format")
	}

	var discoveryMsgs []discovery.DiscoveryMessage

	// Core
	if conf.Core.Store == "" {
		err = fmt.Errorf("core.store is empty")
		return
	}

	nodeId, nodeErr := bundle.NewEndpointID(conf.Core.NodeId)
	if nodeErr != nil {
		err = nodeErr
		return
	}

	c, err = core.NewCore(conf.Core.Store, nodeId, conf.Core.InspectAllBundles, conf.Routing)
	if err != nil {
		return
	}

	// SimpleREST (srest)
	if conf.SimpleRest != (simpleRestConf{}) {
		if aa, err := parseSimpleRESTAppAgent(conf.SimpleRest, c); err == nil {
			c.RegisterApplicationAgent(aa)
		} else {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("Failed to register SimpleRESTAppAgent")
		}
	}

	// Listen/ConvergenceReceiver
	for _, conv := range conf.Listen {
		var convRec cla.ConvergenceReceiver
		var discoMsg discovery.DiscoveryMessage

		convRec, discoMsg, err = parseListen(conv, c.NodeId)
		if err != nil {
			return
		}

		discoveryMsgs = append(discoveryMsgs, discoMsg)

		c.RegisterConvergence(convRec)
	}

	// Peer/ConvergenceSender
	for _, conv := range conf.Peer {
		convRec, err := parsePeer(conv)
		if err != nil {
			log.WithFields(log.Fields{
				"peer":  conv.Endpoint,
				"error": err,
			}).Warn("Failed to establish a connection to a peer")
			continue
		}

		c.RegisterConvergence(convRec)
	}

	// Discovery
	if conf.Discovery.IPv4 || conf.Discovery.IPv6 {
		if conf.Discovery.Interval == 0 {
			conf.Discovery.Interval = 10
		}

		ds, err = discovery.NewDiscoveryService(
			discoveryMsgs, c, conf.Discovery.Interval,
			conf.Discovery.IPv4, conf.Discovery.IPv6)
		if err != nil {
			return
		}
	}

	return
}
