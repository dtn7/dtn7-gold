package bbc

// dummyHub connects multiple dummyModems and helps mocking the bbc package.
type dummyHub struct {
	modems []*dummyModem
}

// newDummyHub creates a new dummyHub.
func newDummyHub() *dummyHub {
	return &dummyHub{}
}

// connect a dummyModem to this dummyHub. This method is called from the newDummyModem function.
func (dh *dummyHub) connect(m *dummyModem) {
	dh.modems = append(dh.modems, m)
}

// receive a Fragment to this dummyHub and distribute it to its dummyModems.
func (dh *dummyHub) receive(f Fragment) {
	for _, m := range dh.modems {
		m.deliver(f)
	}
}

// dummyModem is a mocking Modem used for testing.
type dummyModem struct {
	mtu    int
	hub    *dummyHub
	inChan chan Fragment
}

// newDummyModem creates a new dummyModem and connects itself to a dummyHub.
func newDummyModem(mtu int, hub *dummyHub) *dummyModem {
	d := &dummyModem{
		mtu:    mtu,
		hub:    hub,
		inChan: make(chan Fragment, 10),
	}
	hub.connect(d)

	return d
}

// deliver a Fragment from a dummyHub to this dummyModem.
func (d *dummyModem) deliver(f Fragment) {
	d.inChan <- f
}

func (d *dummyModem) Mtu() int {
	return d.mtu
}

func (d *dummyModem) Send(f Fragment) error {
	d.hub.receive(f)
	return nil
}

func (d *dummyModem) Receive() (f Fragment, err error) {
	f = <-d.inChan
	return
}
