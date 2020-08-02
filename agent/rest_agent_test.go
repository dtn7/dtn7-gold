package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/gorilla/mux"
)

func TestRestAgentCycle(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	// Start REST server
	addr := fmt.Sprintf("localhost:%d", randomPort(t))

	r := mux.NewRouter()
	restRouter := r.PathPrefix("/rest").Subrouter()
	httpServer := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	go func() { _ = httpServer.ListenAndServe() }()

	restAgent := NewRestAgent(restRouter)

	for i := 1; i <= 3; i++ {
		if isAddrReachable(addr) {
			break
		} else if i == 3 {
			t.Fatal("RestAgent seems to be unreachable")
		}
	}

	// Register new client
	registerEid := bundle.MustNewEndpointID("dtn://foo/bar")

	registerUrl := fmt.Sprintf("http://%s/rest/register", addr)
	registerRequestBuf := new(bytes.Buffer)
	registerRequest := RestRegisterRequest{EndpointId: registerEid.String()}
	if err := json.NewEncoder(registerRequestBuf).Encode(registerRequest); err != nil {
		t.Fatal(err)
	}
	registerResponse := RestRegisterResponse{}

	if resp, err := http.Post(registerUrl, "application/json", registerRequestBuf); err != nil {
		t.Fatal(err)
	} else if err := json.NewDecoder(resp.Body).Decode(&registerResponse); err != nil {
		t.Fatal(err)
	} else if registerResponse.Error != "" {
		t.Fatal(registerResponse.Error)
	}

	time.Sleep(250 * time.Millisecond)

	// Check registration
	if !AppAgentHasEndpoint(restAgent, registerEid) {
		t.Fatal("endpoint was not registered")
	}

	// Send bundle to client
	b := createBundle("dtn://sender/", registerEid.String(), t)
	restAgent.MessageReceiver() <- BundleMessage{Bundle: b}

	time.Sleep(250 * time.Millisecond)

	// Fetch bundle
	fetchUrl := fmt.Sprintf("http://%s/rest/fetch", addr)
	fetchRequestBuf := new(bytes.Buffer)
	fetchRequest := RestFetchRequest{UUID: registerResponse.UUID}
	if err := json.NewEncoder(fetchRequestBuf).Encode(fetchRequest); err != nil {
		t.Fatal(err)
	}
	var fetchResponse interface{}

	if resp, err := http.Post(fetchUrl, "application/json", fetchRequestBuf); err != nil {
		t.Fatal(err)
	} else if err := json.NewDecoder(resp.Body).Decode(&fetchResponse); err != nil {
		t.Fatal(err)
	} else if m, ok := fetchResponse.(map[string]interface{}); !ok {
		t.Fatal("failed to read response as a map")
	} else if errorMsg, ok := m["error"]; !ok {
		t.Fatal("error field is missing")
	} else if errorMsg != "" {
		t.Fatal(errorMsg)
	} else if bArr, ok := m["bundles"]; !ok {
		t.Fatal("bundles field is missing")
	} else if l := len(bArr.([]interface{})); l != 1 {
		t.Fatalf("bundles arrays has %d elements, not 1", l)
	}

	// Fetch again; must error
	if err := json.NewEncoder(fetchRequestBuf).Encode(fetchRequest); err != nil {
		t.Fatal(err)
	}

	if resp, err := http.Post(fetchUrl, "application/json", fetchRequestBuf); err != nil {
		t.Fatal(err)
	} else if err := json.NewDecoder(resp.Body).Decode(&fetchResponse); err != nil {
		t.Fatal(err)
	} else if m, ok := fetchResponse.(map[string]interface{}); !ok {
		t.Fatal("failed to read response as a map")
	} else if errorMsg, ok := m["error"]; !ok {
		t.Fatal("error field is missing")
	} else if errorMsg != "" {
		t.Fatal(errorMsg)
	} else if bArr, ok := m["bundles"]; !ok {
		t.Fatal("bundles field is missing")
	} else if len(bArr.([]interface{})) != 0 {
		t.Fatalf("bundles arrays is not empty: %v", bArr.([]interface{}))
	}

	// Build bundle
	buildUrl := fmt.Sprintf("http://%s/rest/build", addr)
	buildR := strings.NewReader(fmt.Sprintf(`{
		"uuid": "%s",
		"arguments": {
			"destination":              "dtn://dst/",
			"source":                   "%s",
			"creation_timestamp_epoch": 1,
			"bundle_age_block":         42000000,
			"lifetime":                 "24h",
			"payload_block":            "hello world"
		}
	}`, registerResponse.UUID, registerEid.String()))
	buildResponse := RestBuildResponse{}

	buildBndl, buildBndlErr := bundle.Builder().
		Destination("dtn://dst/").
		Source(registerEid).
		CreationTimestampEpoch().
		Lifetime("24h").
		BundleAgeBlock(42000000).
		PayloadBlock([]byte("hello world")).
		Build()
	if buildBndlErr != nil {
		t.Fatal(buildBndlErr)
	}

	var (
		buildResponseBundle    bundle.Bundle
		buildResponseWaitGroup sync.WaitGroup
	)
	buildResponseWaitGroup.Add(1)

	go func() {
		defer buildResponseWaitGroup.Done()

		select {
		case msg := <-restAgent.MessageSender():
			if bMsg, ok := msg.(BundleMessage); ok {
				buildResponseBundle = bMsg.Bundle
			}
			return

		case <-time.After(250 * time.Millisecond):
			return
		}
	}()

	if resp, err := http.Post(buildUrl, "application/json", buildR); err != nil {
		t.Fatal(err)
	} else if err := json.NewDecoder(resp.Body).Decode(&buildResponse); err != nil {
		t.Fatal(err)
	}

	// sync bundle reception from channel and json response
	buildResponseWaitGroup.Wait()

	if buildResponse.Error != "" {
		t.Fatal(buildResponse.Error)
	} else if !reflect.DeepEqual(buildResponseBundle, buildBndl) {
		t.Fatalf("%v != %v", buildResponseBundle, buildBndl)
	}

	// Unregister client
	unregisterUrl := fmt.Sprintf("http://%s/rest/unregister", addr)
	unregisterRequestBuf := new(bytes.Buffer)
	unregisterRequest := RestUnregisterRequest{UUID: registerResponse.UUID}
	if err := json.NewEncoder(unregisterRequestBuf).Encode(unregisterRequest); err != nil {
		t.Fatal(err)
	}
	unregisterResponse := RestUnregisterResponse{}

	if resp, err := http.Post(unregisterUrl, "application/json", unregisterRequestBuf); err != nil {
		t.Fatal(err)
	} else if err := json.NewDecoder(resp.Body).Decode(&unregisterResponse); err != nil {
		t.Fatal(err)
	} else if unregisterResponse.Error != "" {
		t.Fatalf("unregistration errored: %s", unregisterResponse.Error)
	}

	if AppAgentHasEndpoint(restAgent, registerEid) {
		t.Fatal("endpoint is still registered")
	}
}
