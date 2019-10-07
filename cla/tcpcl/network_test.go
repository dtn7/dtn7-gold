package tcpcl

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
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

func handleServer(serverAddr string, msgs, clients int, clientWg, serverWg *sync.WaitGroup, errs chan error) {
	var (
		msgsRecv  uint32
		msgsDsprd uint32
		msgsApprd uint32
	)

	defer serverWg.Done()

	serv := NewTCPCLServer(serverAddr, bundle.MustNewEndpointID("dtn://server/"), true)
	if err, _ := serv.Start(); err != nil {
		errs <- err
		return
	}

	go func() {
		for {
			switch cs := <-serv.Channel(); cs.MessageType {
			case cla.ReceivedBundle:
				atomic.AddUint32(&msgsRecv, 1)

			case cla.PeerDisappeared:
				atomic.AddUint32(&msgsDsprd, 1)

			case cla.PeerAppeared:
				atomic.AddUint32(&msgsApprd, 1)
			}
		}
	}()

	clientWg.Wait()
	// TODO
	// serv.Close()

	time.Sleep(250 * time.Millisecond)

	if r := atomic.LoadUint32(&msgsRecv); r != uint32(msgs*clients) {
		errs <- fmt.Errorf("Server received %d messages instead of %d", r, msgs*clients)
	}
	if d := atomic.LoadUint32(&msgsDsprd); d != uint32(clients) {
		errs <- fmt.Errorf("Server received %d disappeared peers instead of %d", d, clients)
	}
	if a := atomic.LoadUint32(&msgsApprd); a != uint32(clients) {
		errs <- fmt.Errorf("Server received %d appeared peers instead of %d", a, clients)
	}
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

	go func() {
		for {
			<-client.Channel()
		}
	}()

	go func() {
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
	}()

	clientWg.Wait()
	client.Close()
}

func startTestTCPCLNetwork(msgs, clients int, t *testing.T) {
	log.SetLevel(log.DebugLevel)

	var serverAddr = fmt.Sprintf("localhost:%d", getRandomPort(t))
	var errs = make(chan error)

	var clientWg sync.WaitGroup
	var serverWg sync.WaitGroup

	clientWg.Add(clients)
	serverWg.Add(1)

	go handleServer(serverAddr, msgs, clients, &clientWg, &serverWg, errs)
	time.Sleep(100 * time.Millisecond)

	for i := 0; i < clients; i++ {
		go handleClient(serverAddr, i, msgs, &clientWg, errs)
	}

	go func() {
		serverWg.Wait()
		close(errs)
	}()

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
