// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bbc

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/rf95modem-go/rf95"
)

// Rf95Modem is a Modem for transmitting and receiving Fragments by using LoRa over a rf95modem.
type Rf95Modem struct {
	device string
	modem  *rf95.Modem
}

// NewRf95Modem creates a new Rf95Modem using a serial connection to the given device, e.g., /dev/ttyUSB0.
func NewRf95Modem(device string) (rfModem *Rf95Modem, err error) {
	if m, mErr := rf95.OpenSerial(device); mErr != nil {
		err = mErr
	} else {
		rfModem = &Rf95Modem{
			device: device,
			modem:  m,
		}
	}

	return
}

// Frequency changes the internal rf95modem's frequency, specified in MHz.
func (rfModem *Rf95Modem) Frequency(frequency float64) error {
	log.WithFields(log.Fields{
		"modem":     rfModem,
		"frequency": frequency,
	}).Debug("Shifting frequency")

	return rfModem.modem.Frequency(frequency)
}

// Mode sets the internal rf95modem's modem config.
func (rfModem *Rf95Modem) Mode(mode rf95.ModemMode) error {
	log.WithFields(log.Fields{
		"modem": rfModem,
		"mode":  mode,
	}).Debug("Changing mode")

	return rfModem.modem.Mode(mode)
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

func (rfModem *Rf95Modem) Close() error {
	return rfModem.modem.Close()
}

func (rfModem *Rf95Modem) String() string {
	status, err := rfModem.modem.FetchStatus()
	if err != nil {
		return fmt.Sprintf("rf95modem%s", rfModem.device)
	}

	return fmt.Sprintf("rf95modem%s?frequency=%f&mode=%d", rfModem.device, status.Frequency, status.Mode)
}
