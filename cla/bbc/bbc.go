package bbc

import (
	"fmt"
	"net/url"
)

// NewBundleBroadcastingConnector tries to create a new Bundle Broadcasting Connector based on an bbc URI.
//
// bbc://rf95modem/dev/ttyUSB0 would open a rf95modem for /dev/ttyUSB0.
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
		if rf95M, rf95Err := NewRf95Modem(uri.Path); rf95Err != nil {
			err = rf95Err
			return
		} else {
			m = rf95M
		}

	default:
		err = fmt.Errorf("unknown host type %s", uri.Host)
		return
	}

	c = NewConnector(m, permanent)
	return
}
