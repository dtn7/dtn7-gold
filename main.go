package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	/*
		bndl, err := bundle.NewBundle(
			bundle.NewPrimaryBlock(
				bundle.MustNotFragmented|bundle.StatusRequestReception|bundle.StatusRequestDelivery,
				ep3,
				ep1,
				bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0), 1000000),
			[]bundle.CanonicalBlock{
				bundle.NewHopCountBlock(1, 0, bundle.NewHopCount(23)),
				bundle.NewPayloadBlock(0, []byte("hello world!")),
			})
		if err != nil {
			panic(err)
		}

		cl1.SendBundle(bndl)
	*/

	dtn, err := parseCore("configuration.toml")
	if err != nil {
		fmt.Printf("Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	time.Sleep(time.Second)

	dtn.Close()
}
