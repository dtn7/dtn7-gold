package core

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/geistesk/dtn7/bundle"
	"github.com/ugorji/go/codec"
)

type Request struct {
	Destination string `codec:"dest"`
	Payload     string `codec:"payload"`
}

type SimpleRESTAppAgent struct {
	endpointID bundle.EndpointID
	c          *Core

	serv        *http.Server
	bundles     []bundle.Bundle
	bundleMutex sync.Mutex
}

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

	codec.NewEncoder(respWriter, new(codec.JsonHandle)).Encode(aa.bundles)
	aa.bundles = aa.bundles[:0]

	aa.bundleMutex.Unlock()
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

	var postReq Request
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
