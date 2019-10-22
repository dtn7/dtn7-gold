package bbc

import (
	"github.com/dtn7/rf95modem-go/rf95"
)

type Rf95Modem struct {
	modem *rf95.Modem
}

func NewRf95Modem(device string) (rfModem *Rf95Modem, err error) {
	if m, mErr := rf95.OpenModem(device); mErr != nil {
		err = mErr
	} else {
		rfModem = &Rf95Modem{modem: m}
	}

	return
}

func (rfModem *Rf95Modem) MTU() int {
	mtu, _ := rfModem.modem.Mtu()
	return mtu
}

func (rfModem *Rf95Modem) Send(f Fragment) error {
	_, err := rfModem.modem.Write(f.Bytes())
	return err
}

func (rfModem *Rf95Modem) Receive() (fragment Fragment, err error) {
	buf := make([]byte, rfModem.MTU())
	if n, readErr := rfModem.modem.Read(buf); readErr != nil {
		err = readErr
	} else if f, fErr := ParseFragment(buf[:n]); fErr != nil {
		err = fErr
	} else {
		fragment = f
	}

	return
}
