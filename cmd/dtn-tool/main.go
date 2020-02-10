package main

import (
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
)

// createBundle for the "create" CLI option.
func createBundle(args []string) {
	if len(args) != 4 {
		printUsage()
	}

	var (
		sender    = args[0]
		receiver  = args[1]
		dataInput = args[2]
		outName   = args[3]

		err  error
		data []byte
		b    bundle.Bundle
		f    *os.File
	)

	if dataInput == "-" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(dataInput)
	}
	if err != nil {
		log.WithError(err).Fatal("Reading input errored")
	}

	b, err = bundle.Builder().
		CRC(bundle.CRC32).
		Source(sender).
		Destination(receiver).
		CreationTimestampNow().
		Lifetime("24h").
		HopCountBlock(64).
		PayloadBlock(data).
		Build()
	if err != nil {
		log.WithError(err).Fatal("Building Bundle errored")
	}

	if f, err = os.Create(outName); err != nil {
		log.WithError(err).Fatal("Creating file errored")
	}
	if err = b.MarshalCbor(f); err != nil {
		log.WithError(err).Fatal("Writing Bundle errored")
	}
	if err = f.Close(); err != nil {
		log.WithError(err).Fatal("Closing file errored")
	}
}

// printUsage of dtn-tool and exit with an error code afterwards.
func printUsage() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage of %s create|show|serve-dir:\n\n", os.Args[0])

	_, _ = fmt.Fprintf(os.Stderr, "%s create sender receiver -|filename bundle-name\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  Creates a new Bundle, addressed from sender to receiver, with the stdin (-) or\n")
	_, _ = fmt.Fprintf(os.Stderr, "  the given file (filename) as payload. This Bundle will be saved as bundle-name.\n\n")

	_, _ = fmt.Fprintf(os.Stderr, "%s show filename\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  Prints a human-readable version of the given Bundle.\n\n")

	_, _ = fmt.Fprintf(os.Stderr, "%s serve-dir websocket endpoint-id directory\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  %s registeres itself as an agent on the given websocket and writes\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  incoming Bundles in the directory. If the user dropps a new Bundle in the\n")
	_, _ = fmt.Fprintf(os.Stderr, "  directory, it will be sent to the server.\n\n")

	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
	}

	switch os.Args[1] {
	case "create":
		createBundle(os.Args[2:])

	case "show":

	case "serve-dir":

	default:
		printUsage()
	}
	//
	//api := "ws://localhost:8080/ws"
	//eid := "dtn://foo/bar"
	//
	//wac, err := agent.NewWebSocketAgentConnector(api, eid)
	//if err != nil {
	//	log.WithError(err).Fatal("Creating WebSocket agent errored")
	//}
	//
	//b := createBundle(eid, "dtn://uff/", []byte("hello world"))
	//if err := wac.WriteBundle(b); err != nil {
	//	log.WithError(err).Fatal("Sending Bundle errored")
	//}
	//
	//wac.Close()
}
