package tcpcl

import "errors"

// ClientState describes the state of a TCPCL Client. Each Client can always
// upgrade its state to a later one, but cannot go back to a previous state.
// A transition can be made into the following or the termination state.
type ClientState int

const (
	// Contact is the initial Contact Header exchange state, entered directly
	// after a TCP connection was established.
	Contact ClientState = iota

	// Init is the SESS_INIT state.
	Init ClientState = iota

	// Established describes an established connection, allowing Bundles to be exchanged.
	Established ClientState = iota

	// Termination is the final state, entered when at least one client wants to
	// terminate/close the session.
	Termination ClientState = iota
)

func (cs ClientState) String() string {
	switch cs {
	case Contact:
		return "contact"
	case Init:
		return "initialization"
	case Established:
		return "established"
	case Termination:
		return "termination"
	default:
		return "INVALID"
	}
}

// Next enters the following ClientState or errors if there is no next state.
func (cs *ClientState) Next() (err error) {
	if *cs == Termination {
		err = errors.New("There is no state after termination")
	} else {
		*cs += 1
	}

	return
}
