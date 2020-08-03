// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package tcpcl

import "sync"

// ClientState describes the state of a TCPCL Client. Each Client can always
// upgrade its state to a later one, but cannot go back to a previous state.
// A transition can be made into the following or the termination state.
type ClientState struct {
	phase int
	mutex sync.Mutex
}

const (
	// phaseContact is the initial Contact Header exchange state, entered directly
	// after a TCP connection was established.
	phaseContact int = iota

	// init is the SESS_INIT state.
	phaseInit int = iota

	// phaseEstablished describes an established connection, allowing Bundles to be exchanged.
	phaseEstablished int = iota

	// phaseTermination is the final state, entered when at least one client wants to
	// terminate/close the session.
	phaseTermination int = iota
)

func (cs *ClientState) String() string {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	switch cs.phase {
	case phaseContact:
		return "contact"
	case phaseInit:
		return "initialization"
	case phaseEstablished:
		return "established"
	case phaseTermination:
		return "termination"
	default:
		return "INVALID"
	}
}

// Next enters the following ClientState.
func (cs *ClientState) Next() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	if cs.phase != phaseTermination {
		cs.phase += 1
	}
}

// Terminate sets the ClientState into the termination state.
func (cs *ClientState) Terminate() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.phase = phaseTermination
}

func (cs *ClientState) isPhase(phase int) bool {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	return cs.phase == phase
}

// IsContact checks if the ClientState is in the contact state.
func (cs *ClientState) IsContact() bool {
	return cs.isPhase(phaseContact)
}

// IsInit checks if the ClientState is in the initialization state.
func (cs *ClientState) IsInit() bool {
	return cs.isPhase(phaseInit)
}

// IsEstablished checks if the ClientState is in the established state.
func (cs *ClientState) IsEstablished() bool {
	return cs.isPhase(phaseEstablished)
}

// IsTerminated checks if the ClientState is in the terminated state.
func (cs *ClientState) IsTerminated() bool {
	return cs.isPhase(phaseTermination)
}
