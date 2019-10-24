package bbc

import (
	"github.com/dtn7/rf95modem-go/rf95"
)

// Rf95Modem is a Modem for transmitting and receiving Fragments by using LoRa over a rf95modem.
type Rf95Modem struct {
	modem *rf95.Modem
}

// NewRf95Modem creates a new Rf95Modem using a serial connection to the given device, e.g., /dev/ttyUSB0.
func NewRf95Modem(device string) (rfModem *Rf95Modem, err error) {
	if m, mErr := rf95.OpenModem(device); mErr != nil {
		err = mErr
	} else {
		rfModem = &Rf95Modem{modem: m}
	}

	return
}

func (rfModem *Rf95Modem) Mtu() (mtu int) {
	mtu, _ = rfModem.modem.Mtu()
	return
}

func (rfModem *Rf95Modem) Send(f Fragment) (err error) {
	_, err = rfModem.modem.Write(f.Bytes())
	return
}

func (rfModem *Rf95Modem) Receive() (fragment Fragment, err error) {
	buf := make([]byte, rfModem.Mtu())
	if n, readErr := rfModem.modem.Read(buf); readErr != nil {
		err = readErr
	} else if f, fErr := ParseFragment(buf[:n]); fErr != nil {
		err = fErr
	} else {
		fragment = f
	}

	return
}
