// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"testing"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	"github.com/dtn7/dtn7-go/pkg/cla/tcpclv4/internal/msgs"
)

func testGetRandomData(size int) []byte {
	payload := make([]byte, size)

	rand.Seed(0)
	rand.Read(payload)

	return payload
}

func TestTransfer(t *testing.T) {
	var sizes = []int{1, 1024, 1048576}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			bndlOut, err := bpv7.Builder().
				CRC(bpv7.CRC32).
				Source("dtn://src/").
				Destination("dtn://dst/").
				CreationTimestampNow().
				Lifetime("30m").
				HopCountBlock(64).
				PayloadBlock(testGetRandomData(size)).
				Build()
			if err != nil {
				t.Fatal(err)
			}

			out := NewBundleOutgoingTransfer(42, bndlOut)
			in := NewIncomingTransfer(42)

			for {
				if dtm, err := out.NextSegment(1400); err == nil {
					if _, err := in.NextSegment(dtm); err != nil {
						t.Fatal(err)
					}
				} else if err == io.EOF {
					if !in.IsFinished() {
						t.Fatalf("Out has finished, In has not.")
					}

					break
				} else {
					t.Fatal(err)
				}
			}

			if bndlIn, err := in.ToBundle(); err != nil {
				t.Fatal(err)
			} else if !reflect.DeepEqual(bndlOut, bndlIn) {
				t.Fatalf("Bundles differ")
			}
		})
	}
}

func TestTransferManager(t *testing.T) {
	msgIn := make(chan msgs.Message)
	msgOut := make(chan msgs.Message)

	tm1 := NewTransferManager(msgIn, msgOut, 65535)
	tm2 := NewTransferManager(msgOut, msgIn, 65535)

	_, tm1Errs := tm1.Exchange()
	tm2Bundles, tm2Errs := tm2.Exchange()

	var sizes = []int{1, 1024, 1048576}

	for _, size := range sizes {
		bndlOut, err := bpv7.Builder().
			CRC(bpv7.CRC32).
			Source("dtn://src/").
			Destination("dtn://dst/").
			CreationTimestampNow().
			Lifetime("30m").
			HopCountBlock(64).
			PayloadBlock(testGetRandomData(size)).
			Build()
		if err != nil {
			t.Fatal(err)
		}

		sendErr := make(chan error)
		go func() {
			if err := tm1.Send(bndlOut); err != nil {
				sendErr <- err
			}
		}()

		select {
		case err := <-sendErr:
			t.Fatal(err)

		case err := <-tm1Errs:
			t.Fatal(err)

		case err := <-tm2Errs:
			t.Fatal(err)

		case bndlIn := <-tm2Bundles:
			if !reflect.DeepEqual(bndlIn, bndlOut) {
				t.Fatalf("bundles differ: %v, %v", bndlIn, bndlOut)
			}
		}
	}

	if err := tm1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tm2.Close(); err != nil {
		t.Fatal(err)
	}
}
