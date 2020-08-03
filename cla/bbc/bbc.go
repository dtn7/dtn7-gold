// SPDX-FileCopyrightText: 2019, 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package bbc describes a simple Bundle Broadcasting Connector to receive and transmit Bundles over a shared
// broadcasting medium, e.g., LoRa.
package bbc

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/dtn7/rf95modem-go/rf95"
)

// NewBundleBroadcastingConnector tries to create a new Bundle Broadcasting Connector based on an bbc URI.
//
//   - bbc://rf95modem/dev/ttyUSB0 would open a rf95modem for /dev/ttyUSB0.
//   - bbc://rf95modem/dev/ttyUSB0?frequency=865.4 would open a rf95modem at 865.4 MHz.
//   - bbc://rf95modem/dev/ttyUSB0?mode=fast-short-range would open a rf95modem with the FAST+SHORT RANGE mode.
//   - bbc://rf95modem/dev/ttyUSB0?frequency=865.23&mode=fast-short-range would be all together.
//
func NewBundleBroadcastingConnector(addr string, permanent bool) (c *Connector, err error) {
	uri, uriErr := url.Parse(addr)
	if uriErr != nil {
		err = uriErr
		return
	}

	if uri.Scheme != "bbc" {
		err = fmt.Errorf("expected bbc scheme, not %s", uri.Scheme)
		return
	}

	var m Modem
	switch uri.Host {
	case "rf95modem":
		// general rf95modem
		rf95M, rf95Err := NewRf95Modem(uri.Path)
		if rf95Err != nil {
			err = rf95Err
			return
		}

		// frequency parameter
		if freqs, ok := uri.Query()["frequency"]; ok && len(freqs) == 1 {
			if freq, fErr := strconv.ParseFloat(freqs[0], 64); fErr != nil {
				err = fErr
				return
			} else if fErr := rf95M.Frequency(freq); fErr != nil {
				err = fErr
				return
			}
		}

		// mode parameter
		if modes, ok := uri.Query()["mode"]; ok && len(modes) == 1 {
			var m rf95.ModemMode
			switch mode := modes[0]; mode {
			case "medium-range":
				m = rf95.MediumRange
			case "fast-short-range":
				m = rf95.FastShortRange
			case "slow-long-range":
				m = rf95.SlowLongRange
			case "slow-long-range2":
				m = rf95.SlowLongRange2
			default:
				err = fmt.Errorf("unknown mode %s", mode)
				return
			}

			if mErr := rf95M.Mode(m); mErr != nil {
				err = mErr
				return
			}
		}

		m = rf95M

	default:
		err = fmt.Errorf("unknown host type %s", uri.Host)
		return
	}

	c = NewConnector(m, permanent)
	return
}
