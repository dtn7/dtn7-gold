// SPDX-FileCopyrightText: 2020 Alvar Penning
// SPDX-FileCopyrightText: 2020 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cla

// CLAType is one of the supported Convergence Layer Adaptors
type CLAType uint

const (
	// TCPCL is the "Delay-Tolerant Networking TCP Convergence Layer Protocol
	// Version 4" as specified in draft-ietf-dtn-tcpclv4-14 or newer.
	TCPCL CLAType = 0

	// MTCP is the "Minimal TCP Convergence-Layer Protocol" as specified in
	// draft-ietf-dtn-mtcpcl-01 or newer documents.
	MTCP CLAType = 1

	// BBC is the Bundle Broadcasting Connector
	// Only here for completeness
	BBC CLAType = 2
)
