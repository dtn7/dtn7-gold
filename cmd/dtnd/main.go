package main

import (
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/profile"
)

// waitSigint blocks the current thread until a SIGINT appears.
func waitSigint() {
	signalSyn := make(chan os.Signal)
	signalAck := make(chan struct{})

	signal.Notify(signalSyn, os.Interrupt)

	go func() {
		<-signalSyn
		close(signalAck)
	}()

	<-signalAck
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s configuration.toml", os.Args[0])
	}

	core, discovery, profiling, err := parseCore(os.Args[1])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to parse config")
	}

	if profiling {
		defer profile.Start(profile.ProfilePath(".")).Stop()
	}

	waitSigint()
	log.Info("Shutting down..")

	core.Close()

	if discovery != nil {
		discovery.Close()
	}
}
