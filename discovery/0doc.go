// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package discovery contains code for peer/neighbor discovery of other DTN nodes through UDP multicast packages.
package discovery

const (
	// address4 is the default multicast IPv4 address used for discovery.
	address4 = "224.23.23.23"

	// address6 is the default multicast IPv4 add6ess used for discovery.
	address6 = "ff02::23"

	// port is the default multicast UDP port used for discovery.
	port = 35039
)
