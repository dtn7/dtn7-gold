// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

import (
	"bytes"
	"testing"
)

func TestSuccessfulIncomingTransmission(t *testing.T) {
	fs := []Fragment{
		NewFragment(0, 5, true, false, false, []byte{1, 2, 3, 4}),
		NewFragment(0, 6, false, false, false, []byte{5, 6, 7, 8}),
		NewFragment(0, 7, false, false, false, []byte{9, 10, 11, 12}),
		NewFragment(0, 8, false, false, false, []byte{13, 14, 15, 16}),
		NewFragment(0, 9, false, true, false, []byte{17, 18}),
	}

	tr, trErr := NewIncomingTransmission(fs[0])
	if trErr != nil {
		t.Fatal(trErr)
	}

	for i, f := range fs[1:] {
		if fin, err := tr.ReadFragment(f); err != nil {
			t.Fatal(err)
		} else if fin && i != len(fs[1:])-1 {
			t.Fatalf("Finished at index %d/%d", i, len(fs[1:]))
		}
	}

	if !tr.IsFinished() {
		t.Fatal("Transfer was not marked as finished")
	}

	expected := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}
	if !bytes.Equal(tr.Payload, expected) {
		t.Fatalf("Expected payload of %x, got %x", expected, tr.Payload)
	}
}

func TestSuccessfulOutgoingTransmission(t *testing.T) {
	payload := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}

	tr, trErr := newPlainOutgoingTransmission(3, payload, 3)
	if trErr != nil {
		t.Fatal(trErr)
	}

	var fs []Fragment
	for i := 0; ; i++ {
		if f, fin, err := tr.WriteFragment(); err != nil {
			t.Fatal(err)
		} else {
			fs = append(fs, f)
			if fin {
				break
			}
		}

		if i > len(payload)/(3-fragmentIdentifierSize) {
			t.Fatalf("Processed %d fragments", i)
		}
	}

	outputData := make([]byte, 0, len(fs))
	for _, f := range fs {
		outputData = append(outputData, f.Payload...)
	}
	if !bytes.Equal(payload, outputData) {
		t.Fatalf("Sent payload of %x, got %x", payload, outputData)
	}
}

func TestDummyTransmission(t *testing.T) {
	payload := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}
	out, outErr := newPlainOutgoingTransmission(0, payload, 4)
	if outErr != nil {
		t.Fatal(outErr)
	}

	f, _, fErr := out.WriteFragment()
	if fErr != nil {
		t.Fatal(fErr)
	}
	in, inErr := NewIncomingTransmission(f)
	if inErr != nil {
		t.Fatal(inErr)
	}

	for !out.IsFinished() {
		if f, _, fErr := out.WriteFragment(); fErr != nil {
			t.Fatal(fErr)
		} else if _, fErr := in.ReadFragment(f); fErr != nil {
			t.Fatal(fErr)
		}
	}

	if !in.IsFinished() {
		t.Fatal("IncomingTransmission is not finished")
	}

	if !bytes.Equal(payload, in.Payload) {
		t.Fatalf("Sent payload of %x, got %x", payload, in.Payload)
	}
}

func TestTransmissionMissingFragment(t *testing.T) {
	payload := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}
	out, outErr := newPlainOutgoingTransmission(0, payload, 4)
	if outErr != nil {
		t.Fatal(outErr)
	}

	f, _, fErr := out.WriteFragment()
	if fErr != nil {
		t.Fatal(fErr)
	}
	in, inErr := NewIncomingTransmission(f)
	if inErr != nil {
		t.Fatal(inErr)
	}

	// Drop second Fragment
	if _, _, fErr := out.WriteFragment(); fErr != nil {
		t.Fatal(fErr)
	}

	if f, _, fErr := out.WriteFragment(); fErr != nil {
		t.Fatal(fErr)
	} else if _, fErr := in.ReadFragment(f); fErr == nil {
		t.Fatalf("Reading skipped Fragment did not errored")
	}
}
