package main

import (
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/geistesk/dtn7/bundle"
	"github.com/geistesk/dtn7/cla"
	"github.com/geistesk/dtn7/cla/stcp"
	"github.com/geistesk/dtn7/core"
	"github.com/geistesk/dtn7/core/appagent"
	"github.com/geistesk/dtn7/core/appagent/srest"
)

// tomlConfig describes the TOML-configuration.
type tomlConfig struct {
	Core       coreConf
	SimpleRest simpleRestConf `toml:"simple-rest"`
	Listen     []convergenceConf
	Peer       []convergenceConf
}

// coreConf describes the Core-configuration block.
type coreConf struct {
	Store string
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
func parseListen(conv convergenceConf) (cla.ConvergenceReceiver, error) {
	switch conv.Protocol {
	case "stcp":
		endpointID, err := bundle.NewEndpointID(conv.Node)
		if err != nil {
			return nil, err
		}

		return stcp.NewSTCPServer(conv.Endpoint, endpointID), nil

	default:
		return nil, fmt.Errorf("Unknown listen.protocol \"%s\"", conv.Protocol)
	}
}

func parsePeer(conv convergenceConf) (cla.ConvergenceSender, error) {
	switch conv.Protocol {
	case "stcp":
		endpointID, err := bundle.NewEndpointID(conv.Node)
		if err != nil {
			return nil, err
		}

		return stcp.NewSTCPClient(conv.Endpoint, endpointID), nil

	default:
		return nil, fmt.Errorf("Unknown peer.protocol \"%s\"", conv.Protocol)
	}
}

func parseSimpleRESTAppAgent(conf simpleRestConf, c *core.Core) (appagent.ApplicationAgent, error) {
	endpointID, err := bundle.NewEndpointID(conf.Node)
	if err != nil {
		return nil, err
	}

	return srest.NewSimpleRESTAppAgent(endpointID, c, conf.Listen), nil
}

// parseCore creates the Core based on the given TOML configuration.
func parseCore(filename string) (c *core.Core, err error) {
	var conf tomlConfig
	if _, err = toml.DecodeFile(filename, &conf); err != nil {
		return
	}

	// Core
	if conf.Core.Store == "" {
		err = fmt.Errorf("core.store is empty")
		return
	}

	c, err = core.NewCore(conf.Core.Store)
	if err != nil {
		return
	}

	if conf.SimpleRest != (simpleRestConf{}) {
		if aa, err := parseSimpleRESTAppAgent(conf.SimpleRest, c); err == nil {
			c.RegisterApplicationAgent(aa)
		} else {
			log.Printf("Failed to register SimpleRESTAppAgent: %v", err)
		}
	}

	// Listen/ConvergenceReceiver
	for _, conv := range conf.Listen {
		var convRec cla.ConvergenceReceiver
		convRec, err = parseListen(conv)
		if err != nil {
			return
		}

		c.RegisterConvergenceReceiver(convRec)
	}

	// Peer/ConvergenceSender
	for _, conv := range conf.Peer {
		convRec, err := parsePeer(conv)
		if err != nil {
			log.Printf("Failed to establish a connection to peer %v", conv.Endpoint)
			continue
		}

		c.RegisterConvergenceSender(convRec)
	}

	return
}
