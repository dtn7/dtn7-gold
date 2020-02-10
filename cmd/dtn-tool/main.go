package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/agent"
	"github.com/dtn7/dtn7-go/bundle"
)

func createBundle(from, to string, data []byte) bundle.Bundle {
	b, err := bundle.Builder().
		CRC(bundle.CRC32).
		Source(from).
		Destination(to).
		CreationTimestampNow().
		Lifetime("24h").
		HopCountBlock(64).
		PayloadBlock(data).
		Build()
	if err != nil {
		log.WithError(err).Fatal("Building Bundle errored")
	}

	return b
}

func main() {
	api := "ws://localhost:8080/ws"
	eid := "dtn://foo/bar"

	wac, err := agent.NewWebSocketAgentConnector(api, eid)
	if err != nil {
		log.WithError(err).Fatal("Creating WebSocket agent errored")
	}

	b := createBundle(eid, "dtn://uff/", []byte("hello world"))
	if err := wac.WriteBundle(b); err != nil {
		log.WithError(err).Fatal("Sending Bundle errored")
	}

	wac.Close()
}
