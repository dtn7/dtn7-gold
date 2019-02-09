package main

import (
	"log"
	"os"
	"os/signal"
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

	dtn, err := parseCore(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to parse config: %v\n", err)
	}

	waitSigint()
	log.Print("Shutting down")

	dtn.Close()
}
