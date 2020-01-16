package agent

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestWebAgentConnector(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	// Start WebAgent server
	addr := fmt.Sprintf("localhost:%d", randomPort(t))
	ws, wsErr := NewWebAgent(addr)
	if wsErr != nil {
		t.Fatal(wsErr)
	}

	// Let the WebAgent start..
	time.Sleep(250 * time.Millisecond)

	for i := 1; i <= 3; i++ {
		if isAddrReachable(addr) {
			break
		} else if i == 3 {
			t.Fatal("SocketAgent seems to be unreachable")
		}
	}

	// Attach Connector
	u := url.URL{
		Scheme: "ws",
		Host:   addr,
		Path:   "/ws",
	}
	wac, wacErr := NewWebAgentConnector(u.String(), "dtn://foobar/23/")
	if wacErr != nil {
		t.Fatal(wacErr)
	}

	wac.Close()

	// Let the WebAgent act on the closed connection
	time.Sleep(250 * time.Millisecond)

	ws.MessageReceiver() <- ShutdownMessage{}

	// Let the WebAgent shut itself down
	time.Sleep(250 * time.Millisecond)
}
