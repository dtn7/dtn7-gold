// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package tcpclv4 provides a library for the Delay-Tolerant Networking TCP Convergence Layer Protocol Version 4,
// draft-ietf-dtn-tcpclv4-21.
//
// A new TCPCLv4 server can be started by a TCPListener, which provides multiple connection to its Clients. To reach
// a remote server, a new Client connection can be dialed, see DialTCP.
package tcpclv4
