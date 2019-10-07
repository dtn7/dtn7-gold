package tcpcl

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/dtn7/dtn7-go/cla"
)

func getRandomPort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func handleServer(serverAddr string, wg *sync.WaitGroup, errs chan error) {
	serv := NewTCPCLServer(serverAddr, bundle.MustNewEndpointID("dtn://server/"), true)
	if err, _ := serv.Start(); err != nil {
		errs <- err
		return
	}

	go func(serv *TCPCLServer) {
		for {
			<-serv.Channel()
		}
	}(serv)

	wg.Wait()
}

func handleClient(serverAddr string, clientNo, msgs int, wg *sync.WaitGroup, errs chan error) {
	defer wg.Done()

	clientEid := fmt.Sprintf("dtn://client-%d/", clientNo)
	client := Dial(serverAddr, bundle.MustNewEndpointID(clientEid), false)
	if err, _ := client.Start(); err != nil {
		errs <- err
		return
	}

	var clientWg sync.WaitGroup
	clientWg.Add(1)

	go func(client cla.Convergence) {
		for {
			<-client.Channel()
		}
	}(client)

	go func(client *TCPCLClient, clientEid string, msgs int, clientWg *sync.WaitGroup, errs chan error) {
		defer clientWg.Done()

		for !client.state.IsEstablished() {
		}

		for i := 0; i < msgs; i++ {
			bndl, err := bundle.Builder().
				CRC(bundle.CRC32).
				Source(clientEid).
				Destination("dtn://server/").
				CreationTimestampNow().
				Lifetime("30m").
				HopCountBlock(64).
				PayloadBlock([]byte("hello world!")).
				Build()

			if err != nil {
				errs <- err
				return
			} else if err := client.Send(&bndl); err != nil {
				errs <- err
				return
			}
		}
	}(client, clientEid, msgs, wg, errs)

	clientWg.Wait()
	client.Close()
}

func startTestTCPCLNetwork(msgs, clients int, t *testing.T) {
	log.SetLevel(log.DebugLevel)

	var serverAddr = fmt.Sprintf("localhost:%d", getRandomPort(t))
	var errs = make(chan error)
	var wg sync.WaitGroup

	wg.Add(clients)

	go handleServer(serverAddr, &wg, errs)
	time.Sleep(100 * time.Millisecond)

	for i := 0; i < clients; i++ {
		go handleClient(serverAddr, i, msgs, &wg, errs)
	}

	go func(wg *sync.WaitGroup, errs chan error) {
		wg.Wait()
		close(errs)
	}(&wg, errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTCPCLNetwork(t *testing.T) {
	tests := []struct {
		clients int
		msgs    int
	}{{clients: 1, msgs: 1},
		{clients: 1, msgs: 25},
		{clients: 5, msgs: 25},
		{clients: 10, msgs: 25}}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d_clients_%d_msgs", test.clients, test.msgs), func(t *testing.T) {
			startTestTCPCLNetwork(test.msgs, test.clients, t)
		})
	}
}
