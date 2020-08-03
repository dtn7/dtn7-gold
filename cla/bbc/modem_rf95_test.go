// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

// This test depends on your system's hardware, e.g., it needs two connected rf95modems.
/*
func TestRf95Modem(t *testing.T) {
	m0, m0err := NewRf95Modem("/dev/ttyUSB0")
	m1, m1err := NewRf95Modem("/dev/ttyUSB1")

	if m0err != nil || m1err != nil {
		t.Fatal(m0err, m1err)
	}

	msg := []byte("hello world")
	if err := m0.Send(NewFragment(0, 0, false, false, msg)); err != nil {
		t.Fatal(err)
	}

	if f, err := m1.Receive(); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(f.Payload, msg) {
		t.Fatalf("Wrong payload: expected %x, got %x", msg, f.Payload)
	}

	if err := m0.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m1.Close(); err != nil {
		t.Fatal(err)
	}
}
*/
