package main

import (
	"os"
	"os/signal"

	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
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

	defer profile.Start(profile.ProfilePath(".")).Stop()

	core, discovery, err := parseCore(os.Args[1])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to parse config")
	}

	waitSigint()
	log.Info("Shutting down..")

	core.Close()

	if discovery != nil {
		discovery.Close()
	}
}
