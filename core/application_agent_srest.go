package core

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/ugorji/go/codec"
)

// SimpleRESTRequest is the data structure used for outbounding bundles,
// requested through the SimpleRESTAppAgent.
type SimpleRESTRequest struct {
	Destination string
	Payload     string
}

// SimpleRESTRequestResponse is the response, sent to a SimpleRESTRequest.
type SimpleRESTRequestResponse struct {
	Error string
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
func NewSimpleRESTReponseFromBundle(b *bundle.Bundle) SimpleRESTResponse {
	payload, _ := b.PayloadBlock()

	return SimpleRESTResponse{
		Destination:  b.PrimaryBlock.Destination.String(),
		SourceNode:   b.PrimaryBlock.SourceNode.String(),
		ControlFlags: b.PrimaryBlock.BundleControlFlags.String(),
		Timestamp: [2]string{
			bundle.DtnTime(b.PrimaryBlock.CreationTimestamp[0]).String(),
			fmt.Sprintf("%d", b.PrimaryBlock.CreationTimestamp[1])},
		Payload: payload.Value.(*bundle.PayloadBlock).Data(),
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
// data. The payload must be base64 encoded.
//
//	curl -d "{\"Destination\":\"dtn:foobar\", \"Payload\":\"`base64 <<< "hello world"`\"}" http://localhost:8080/send/
//
// Would create an outbounding bundle with a "hello" payload, addressed to an
// endpoint named "dtn:foobar".
type SimpleRESTAppAgent struct {
	endpointID bundle.EndpointID
	c          *Core

	serv        *http.Server
	bundleIds   []string
	bundleMutex sync.Mutex
}

// NewSimpleRESTAppAgent creates a new SimpleRESTAppAgent for the given
// endpoint, Core and bound to the address.
func NewSimpleRESTAppAgent(endpointID bundle.EndpointID, c *Core, addr string) (aa *SimpleRESTAppAgent) {
	aa = &SimpleRESTAppAgent{
		endpointID: endpointID,
		c:          c,
		bundleIds:  make([]string, 0, 0),
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
	for _, bndlId := range aa.bundleIds {
		if bp, err := aa.c.store.QueryId(bndlId); err != nil {
			log.WithFields(log.Fields{
				"bundle": bndlId,
				"error":  err,
			}).Warn("SimpleRESTAppAgent failed to fetch bundle from store")
		} else {
			resps = append(resps, NewSimpleRESTReponseFromBundle(bp.Bundle))

			bp.RemoveConstraint(LocalEndpoint)
			aa.c.store.Push(bp)
		}
	}
	aa.bundleIds = aa.bundleIds[:0]

	aa.bundleMutex.Unlock()

	codec.NewEncoder(respWriter, new(codec.JsonHandle)).Encode(resps)
}

func (aa *SimpleRESTAppAgent) handleSend(respWriter http.ResponseWriter, req *http.Request) {
	var resp SimpleRESTRequestResponse

	defer func(resp SimpleRESTRequestResponse) {
		codec.NewEncoder(respWriter, new(codec.JsonHandle)).Encode(resp)
	}(resp)

	var handleErr = func(msg string) {
		resp = SimpleRESTRequestResponse{msg}
		log.WithFields(log.Fields{
			"srest":   aa.EndpointID(),
			"request": req,
			"error":   msg,
		}).Warn("SimpleRESTAppAgent's send errored")
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

	var payload, base64Err = base64.StdEncoding.DecodeString(postReq.Payload)
	if base64Err != nil {
		handleErr("Failed to decode base64 payload")
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
			bundle.NewCanonicalBlock(1, 0, bundle.NewPayloadBlock(payload))})
	if bndlErr != nil {
		handleErr(fmt.Sprintf("Creating bundle failed: %v", bndlErr))
		return
	}

	aa.c.SendBundle(&bndl)

	resp = SimpleRESTRequestResponse{}
	log.WithFields(log.Fields{
		"srest":  aa.EndpointID(),
		"bundle": bndl.ID(),
	}).Info("SimpleRESTAppAgent's transmitted bundle")
}

// EndpointID returns this SimpleRESTAppAgent's (unique) endpoint ID.
func (aa *SimpleRESTAppAgent) EndpointID() bundle.EndpointID {
	return aa.endpointID
}

// Deliver delivers a received bundle to this SimpleRESTAppAgent. This bundle
// may contain an application specific payload or an administrative record.
func (aa *SimpleRESTAppAgent) Deliver(bp BundlePack) error {
	log.WithFields(log.Fields{
		"srest":  aa.EndpointID(),
		"bundle": bp.ID(),
	}).Info("SimpleRESTAppAgent received a bundle")

	aa.bundleMutex.Lock()
	aa.bundleIds = append(aa.bundleIds, bp.ID())
	aa.bundleMutex.Unlock()

	return nil
}
