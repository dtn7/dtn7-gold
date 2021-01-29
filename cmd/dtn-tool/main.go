// SPDX-FileCopyrightText: 2020, 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
)

// printUsage of dtn-tool and exit with an error code afterwards.
func printUsage() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage of %s create|exchange|ping|show:\n\n", os.Args[0])

	_, _ = fmt.Fprintf(os.Stderr, "%s create sender receiver -|filename [-|filename]\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  Creates a new Bundle, addressed from sender to receiver with the stdin (-)\n")
	_, _ = fmt.Fprintf(os.Stderr, "  or the given file (filename) as payload. If no further specified, the\n")
	_, _ = fmt.Fprintf(os.Stderr, "  Bundle is stored locally named after the hex representation of its ID.\n")
	_, _ = fmt.Fprintf(os.Stderr, "  Otherwise, the Bundle can be written to the stdout (-) or saved\n")
	_, _ = fmt.Fprintf(os.Stderr, "  according to a freely selectable filename.\n\n")

	_, _ = fmt.Fprintf(os.Stderr, "%s exchange websocket endpoint-id directory\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  %s registeres itself as an agent on the given websocket and writes\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  incoming Bundles in the directory. If the user dropps a new Bundle in the\n")
	_, _ = fmt.Fprintf(os.Stderr, "  directory, it will be sent to the server.\n\n")

	_, _ = fmt.Fprintf(os.Stderr, "%s ping websocket sender receiver\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  Send continuously bundles from sender to receiver over a websocket.\n\n")

	_, _ = fmt.Fprintf(os.Stderr, "%s show -|filename\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  Prints a JSON version of a Bundle, read from stdin (-) or filename.\n\n")

	os.Exit(1)
}

// printFatal of an error with a short context description and exits afterwards.
func printFatal(err error, msg string) {
	_, _ = fmt.Fprintf(os.Stderr, "%s errored: %s\n  %v\n", os.Args[0], msg, err)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
	}

	switch os.Args[1] {
	case "create":
		createBundle(os.Args[2:])

	case "exchange":
		startExchange(os.Args[2:])

	case "ping":
		ping(os.Args[2:])

	case "show":
		showBundle(os.Args[2:])

	default:
		printUsage()
	}
}
