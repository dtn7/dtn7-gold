package core

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

// SimpleRESTRequest is the data structure used for outbounding bundles,
// requested through the SimpleRESTAppAgent.
type SimpleRESTRequest struct {
	Destination string
	Payload     string
}

// SimpleRESTResponse is the data structure used for incoming bundles,
// handled through the SimpleRESTAppAgent.
type SimpleRESTResponse struct {
	Destination  string
	SourceNode   string
	ControlFlags string
	Timestamp    [2]string
	Payload      []byte
}

// NewSimpleRESTReponseFromBundle creates a new SimpleRESTResponse for a bundle.
func NewSimpleRESTReponseFromBundle(b bundle.Bundle) SimpleRESTResponse {
	payload, _ := b.PayloadBlock()

	return SimpleRESTResponse{
		Destination:  b.PrimaryBlock.Destination.String(),
		SourceNode:   b.PrimaryBlock.SourceNode.String(),
		ControlFlags: b.PrimaryBlock.BundleControlFlags.String(),
		Timestamp: [2]string{
			bundle.DtnTime(b.PrimaryBlock.CreationTimestamp[0]).String(),
			fmt.Sprintf("%d", b.PrimaryBlock.CreationTimestamp[1])},
		Payload: payload.Data.([]byte),
	}
}

// SimpleRESTAppAgent is an implementation of an ApplicationAgent, useable
// through simple HTTP requests for bundle creation and reception.
//
// The /fetch/ endpoint can be queried through a simple HTTP GET request and
// will return all received bundles. Those are removed from the store
// afterwards.
//
// The /send/ endpoint can be queried through a HTTP POST request with JSON
// data.
//
//	curl -d'{"Destination":"dtn:foobar", "Payload":"hello"}' http://localhost:8080/send/
//
// Would create an outbounding bundle with a "hello" payload, addressed to an
// endpoint named "dtn:foobar".
type SimpleRESTAppAgent struct {
	endpointID bundle.EndpointID
	c          *Core

	serv        *http.Server
	bundles     []bundle.Bundle
	bundleMutex sync.Mutex
}

// NewSimpleRESTAppAgent creates a new SimpleRESTAppAgent for the given
// endpoint, Core and bound to the address.
func NewSimpleRESTAppAgent(endpointID bundle.EndpointID, c *Core, addr string) (aa *SimpleRESTAppAgent) {
	aa = &SimpleRESTAppAgent{
		endpointID: endpointID,
		c:          c,
		bundles:    make([]bundle.Bundle, 0, 0),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/fetch/", aa.handleFetch)
	mux.HandleFunc("/send/", aa.handleSend)

	aa.serv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go aa.serv.ListenAndServe()
	return
}

func (aa *SimpleRESTAppAgent) handleFetch(respWriter http.ResponseWriter, _ *http.Request) {
	aa.bundleMutex.Lock()

	resps := make([]SimpleRESTResponse, 0, 0)
	for _, bndl := range aa.bundles {
		resps = append(resps, NewSimpleRESTReponseFromBundle(bndl))
	}
	aa.bundles = aa.bundles[:0]

	aa.bundleMutex.Unlock()

	codec.NewEncoder(respWriter, new(codec.JsonHandle)).Encode(resps)
}

func (aa *SimpleRESTAppAgent) handleSend(respWriter http.ResponseWriter, req *http.Request) {
	var handleErr = func(msg string) {
		fmt.Fprintf(respWriter, `{"error":"%s"}`, msg)
		log.Printf("SimpleRESTAppAgent's send errored: %s", msg)
	}

	if req.Method != "POST" {
		handleErr("Send expects a POST request")
		return
	}

	var postReq SimpleRESTRequest
	if err := codec.NewDecoder(req.Body, new(codec.JsonHandle)).Decode(&postReq); err != nil {
		handleErr("Failed to parse request")
		return
	}

	var dest, destErr = bundle.NewEndpointID(postReq.Destination)
	if destErr != nil {
		handleErr("Unintelligible destination")
		return
	}

	var bndl, bndlErr = bundle.NewBundle(
		bundle.NewPrimaryBlock(
			bundle.MustNotFragmented|bundle.StatusRequestDelivery,
			dest,
			aa.endpointID,
			bundle.NewCreationTimestamp(bundle.DtnTimeNow(), 0),
			60*60*1000000),
		[]bundle.CanonicalBlock{
			bundle.NewPayloadBlock(0, []byte(postReq.Payload)),
			bundle.NewHopCountBlock(23, 0, bundle.NewHopCount(5)),
		})
	if bndlErr != nil {
		handleErr(fmt.Sprintf("Creating bundle failed: %v", bndlErr))
		return
	}

	aa.c.SendBundle(bndl)

	fmt.Fprintf(respWriter, `{"error":""}`)
	log.Printf("SimpleRESTAppAgent's transmitted %v", bndl)
}

// EndpointID returns this SimpleRESTAppAgent's (unique) endpoint ID.
func (aa *SimpleRESTAppAgent) EndpointID() bundle.EndpointID {
	return aa.endpointID
}

// Deliver delivers a received bundle to this SimpleRESTAppAgent. This bundle
// may contain an application specific payload or an administrative record.
func (aa *SimpleRESTAppAgent) Deliver(bndl *bundle.Bundle) error {
	log.Printf("SimpleRESTAppAgent %v received a bundle: %v", aa.endpointID, bndl)

	aa.bundleMutex.Lock()
	aa.bundles = append(aa.bundles, *bndl)
	aa.bundleMutex.Unlock()

	return nil
}
