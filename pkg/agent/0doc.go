// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package agent describes an interface for modules to receive and send bundles.
//
// The main interface is the ApplicationAgent, which only requires two channels for incoming and outgoing Messages.
// Additionally, it requests a list of endpoints. Due to this flexibility, an ApplicationAgent can be implemented in
// various forms, e.g., as an external interface for third-party programs or as an internal module. Both possibilities
// are already included in this package, for example the WebSocketAgent or the PingAgent.
package agent
