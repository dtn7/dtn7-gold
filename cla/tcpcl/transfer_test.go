// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import (
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
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
			bndlOut, err := bundle.Builder().
				CRC(bundle.CRC32).
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
