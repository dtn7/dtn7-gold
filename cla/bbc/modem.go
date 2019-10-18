package bbc

// Modem is the interface for possible broadcasting modems. Every Modem must be able to broadcast and
// receive packets resp. Fragments. The MTU method indicates the maximum transmission unit (MTU) for
// outgoing Fragments.
type Modem interface {
	// MTU returns the maximum transmission unit for this Modem.
	MTU() int

	// Send broadcasts a Fragment over this Modem. This method might block.
	Send(Fragment) error

	// Receive waits for the next Fragment to be received. This method blocks.
	Receive() (Fragment, error)
}
