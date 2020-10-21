// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

// Modem is the interface for possible broadcasting modems. Every Modem must be able to broadcast and
// receive packets resp. Fragments. The Mtu method indicates the maximum transmission unit (Mtu) for
// outgoing Fragments.
type Modem interface {
	// Mtu returns the maximum transmission unit for this Modem.
	Mtu() int

	// Send broadcasts a Fragment over this Modem. This method might block.
	Send(Fragment) error

	// Receive waits for the next Fragment to be received. This method blocks.
	Receive() (Fragment, error)

	// Close this Modem. Furthermore, the Receive method should be interrupted.
	Close() error
}
