// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

// showBundle for the "show" CLI options.
func showBundle(args []string) {
	if len(args) != 1 {
		printUsage()
	}

	var (
		input = args[0]

		err  error
		f    io.ReadCloser
		b    bpv7.Bundle
		bMsg []byte
	)

	if input == "-" {
		f = os.Stdin
	} else if f, err = os.Open(input); err != nil {
		printFatal(err, "Opening file for reading errored")
	}

	if err = b.UnmarshalCbor(f); err != nil {
		printFatal(err, "Unmarshaling Bundle errored")
	}
	if err = f.Close(); err != nil {
		printFatal(err, "Closing file errored")
	}

	if bMsg, err = b.MarshalJSON(); err != nil {
		printFatal(err, "Marshaling JSON errored")
	}
	fmt.Println(string(bMsg))
}
