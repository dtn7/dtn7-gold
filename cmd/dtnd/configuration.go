package main

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/BurntSushi/toml"
	"github.com/dtn7/dtn7-go/agent"
	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
	"github.com/dtn7/dtn7-go/cla/bbc"
	"github.com/dtn7/dtn7-go/cla/mtcp"
	"github.com/dtn7/dtn7-go/cla/tcpcl"
	"github.com/dtn7/dtn7-go/core"
	"github.com/dtn7/dtn7-go/discovery"
)

// tomlConfig describes the TOML-configuration.
type tomlConfig struct {
	Core      coreConf
	Logging   logConf
	Discovery discoveryConf
	Agents    agentsConfig
	Listen    []convergenceConf
	Peer      []convergenceConf
	Routing   core.RoutingConf
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

// agentsConfig describes the ApplicationAgents/Agent-configuration block.
type agentsConfig struct {
	Webserver agentsWebserverConfig
}

// agentsWebserverConfig describes the nested "Webserver" configuration for agents.
type agentsWebserverConfig struct {
	Address   string
	Websocket bool
	Rest      bool
}

// convergenceConf describes the Convergence-configuration block, used for
// "listen" and "peer".
type convergenceConf struct {
	Node     string
	Protocol string
	Endpoint string
}

func parseListenPort(endpoint string) (port int, err error) {
	var portStr string
	_, portStr, err = net.SplitHostPort(endpoint)
	if err != nil {
		return
	}
	port, err = strconv.Atoi(portStr)
	return
}

// parseListen inspects a "listen" convergenceConf and returns a Convergable.
func parseListen(conv convergenceConf, nodeId bundle.EndpointID) (cla.Convergable, discovery.DiscoveryMessage, error) {
	log.WithFields(log.Fields{
		"EndpointID": conv.Node,
		"Endpoint":   conv.Endpoint,
		"Protocol":   conv.Protocol,
	}).Debug("Initialising convergence adaptor")

	// if the user has configured an EndpointID for this convergence adaptor
	if conv.Node != "" {
		parsedId, err := bundle.NewEndpointID(conv.Node)
		if err != nil {
			log.WithFields(log.Fields{
				"listener ID": conv.Node,
				"core ID":     nodeId,
			}).Error("Invalid endpoint configuration, falling back to core default id")
		} else {
			log.WithFields(log.Fields{
				"listener ID": conv.Node,
			}).Debug("Using alternative configured endpoint if for listener")
			nodeId = parsedId
		}
	}

	switch conv.Protocol {
	case "bbc":
		conn, err := bbc.NewBundleBroadcastingConnector(conv.Endpoint, true)
		return conn, discovery.DiscoveryMessage{}, err

	case "mtcp":
		portInt, err := parseListenPort(conv.Endpoint)
		if err != nil {
			return nil, discovery.DiscoveryMessage{}, err
		}

		msg := discovery.DiscoveryMessage{
			Type:     discovery.MTCP,
			Endpoint: nodeId,
			Port:     uint(portInt),
		}

		return mtcp.NewMTCPServer(conv.Endpoint, nodeId, true), msg, nil

	case "tcpcl":
		portInt, err := parseListenPort(conv.Endpoint)
		if err != nil {
			return nil, discovery.DiscoveryMessage{}, err
		}

		listener := tcpcl.NewListener(conv.Endpoint, nodeId)

		msg := discovery.DiscoveryMessage{
			Type:     discovery.TCPCL,
			Endpoint: nodeId,
			Port:     uint(portInt),
		}

		return listener, msg, nil

	default:
		return nil, discovery.DiscoveryMessage{}, fmt.Errorf("unknown listen.protocol \"%s\"", conv.Protocol)
	}
}

func parsePeer(conv convergenceConf, nodeId bundle.EndpointID) (cla.ConvergenceSender, error) {
	endpointID, err := bundle.NewEndpointID(conv.Node)
	if err != nil {
		return nil, err
	}

	switch conv.Protocol {
	case "mtcp":
		return mtcp.NewMTCPClient(conv.Endpoint, endpointID, true), nil

	case "tcpcl":
		return tcpcl.DialClient(conv.Endpoint, nodeId, true), nil

	default:
		return nil, fmt.Errorf("unknown peer.protocol \"%s\"", conv.Protocol)
	}
}

// parseAgents for the ApplicationAgents.
func parseAgents(conf agentsConfig) (agents []agent.ApplicationAgent, err error) {
	if (conf.Webserver != agentsWebserverConfig{}) {
		if !conf.Webserver.Websocket && !conf.Webserver.Rest {
			err = fmt.Errorf("webserver agent needs at least one of Websocket or REST")
			return
		}

		r := mux.NewRouter()

		if conf.Webserver.Websocket {
			ws := agent.NewWebSocketAgent()
			r.HandleFunc("/ws", ws.ServeHTTP)

			agents = append(agents, ws)
		}

		if conf.Webserver.Rest {
			restRouter := r.PathPrefix("/rest").Subrouter()
			ra := agent.NewRestAgent(restRouter)

			agents = append(agents, ra)
		}

		httpServer := &http.Server{
			Addr:    conf.Webserver.Address,
			Handler: r,
		}

		errChan := make(chan error)
		go func() { errChan <- httpServer.ListenAndServe() }()

		select {
		case err = <-errChan:
			return

		case <-time.After(100 * time.Millisecond):
			break
		}
	}

	return
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

	log.WithFields(log.Fields{
		"routing": conf.Routing.Algorithm,
	}).Debug("Selected routing algorithm")

	nodeId, nodeErr := bundle.NewEndpointID(conf.Core.NodeId)
	if nodeErr != nil {
		err = nodeErr
		return
	}

	c, err = core.NewCore(conf.Core.Store, nodeId, conf.Core.InspectAllBundles, conf.Routing)
	if err != nil {
		return
	}

	// Agents
	if conf.Agents != (agentsConfig{}) {
		if appAgents, appErr := parseAgents(conf.Agents); appErr != nil {
			err = appErr
			return
		} else {
			for _, appAgent := range appAgents {
				c.RegisterApplicationAgent(appAgent)
			}
		}
	}

	// Listen/ConvergenceReceiver
	for _, conv := range conf.Listen {
		if convRec, discoMsg, lErr := parseListen(conv, c.NodeId); lErr != nil {
			err = lErr
			return
		} else {
			c.RegisterConvergable(convRec)
			if discoMsg != (discovery.DiscoveryMessage{}) {
				discoveryMsgs = append(discoveryMsgs, discoMsg)
			}
		}
	}

	// Peer/ConvergenceSender
	for _, conv := range conf.Peer {
		convRec, err := parsePeer(conv, c.NodeId)
		if err != nil {
			log.WithFields(log.Fields{
				"peer":  conv.Endpoint,
				"error": err,
			}).Warn("Failed to establish a connection to a peer")
			continue
		}

		c.RegisterConvergable(convRec)
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
