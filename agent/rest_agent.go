package agent

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/dtn7/dtn7-go/bundle"
	"github.com/gorilla/mux"
)

// RestAgent is a RESTful Application Agent for an easier bundle dispatching.
type RestAgent struct {
	router *mux.Router

	receiver chan Message
	sender   chan Message

	// map UUIDs to EIDs and received bundles
	clients sync.Map // uuid[string] -> bundle.EndpointID
	mailbox sync.Map // uuid[string] -> []bundle.Bundle
}

// NewRestAgent creates a new RESTful Application Agent.
func NewRestAgent(router *mux.Router) (ra *RestAgent) {
	ra = &RestAgent{
		router: router,

		receiver: make(chan Message),
		sender:   make(chan Message),
	}

	ra.router.HandleFunc("/register", ra.handleRegister).Methods(http.MethodPost)
	ra.router.HandleFunc("/unregister", ra.handleUnregister).Methods(http.MethodPost)
	ra.router.HandleFunc("/fetch", ra.handleFetch).Methods(http.MethodPost)

	go ra.handler()

	return ra
}

// handler checks the receiver channel and deals with inbounding messages.
func (ra *RestAgent) handler() {
	defer close(ra.sender)

	for msg := range ra.receiver {
		switch msg := msg.(type) {
		case BundleMessage:
			ra.receiveBundleMessage(msg)

		case ShutdownMessage:
			// TODO
			return

		default:
			// TODO
		}
	}
}

// receiveBundleMessage checks incoming BundleMessages and puts them inbox.
func (ra *RestAgent) receiveBundleMessage(msg BundleMessage) {
	var uuids []string
	ra.clients.Range(func(k, v interface{}) bool {
		if BagHasEndpoint(msg.Recipients(), v.(bundle.EndpointID)) {
			uuids = append(uuids, k.(string))
		}
		return false // multiple clients might be registered for some endpoint
	})

	for _, uuid := range uuids {
		var bundles []bundle.Bundle
		if val, ok := ra.mailbox.Load(uuid); !ok {
			bundles = []bundle.Bundle{msg.Bundle}
		} else {
			bundles = append(val.([]bundle.Bundle), msg.Bundle)
		}

		ra.mailbox.Store(uuid, bundles)

		log.WithFields(log.Fields{
			"bundle": msg.Bundle.ID().String(),
			"uuid":   uuid,
		}).Info("REST Application Agent delivering message to a client's inbox")
	}
}

// randomUuid to be used for authentication. UUID does not complain RFC 4122.
func (_ *RestAgent) randomUuid() (uuid string, err error) {
	uuidBytes := make([]byte, 16)
	if _, err = rand.Read(uuidBytes); err == nil {
		uuid = fmt.Sprintf("%x-%x-%x-%x-%x",
			uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:16])
	}
	return
}

// handleRegister processes /register POST requests.
func (ra *RestAgent) handleRegister(w http.ResponseWriter, r *http.Request) {
	var (
		registerRequest  RestRegisterRequest
		registerResponse RestRegisterResponse
	)

	if jsonErr := json.NewDecoder(r.Body).Decode(&registerRequest); jsonErr != nil {
		registerResponse.Error = jsonErr.Error()
	} else if eid, eidErr := bundle.NewEndpointID(registerRequest.EndpointId); eidErr != nil {
		registerResponse.Error = eidErr.Error()
	} else if uuid, uuidErr := ra.randomUuid(); uuidErr != nil {
		registerResponse.Error = uuidErr.Error()
	} else {
		ra.clients.Store(uuid, eid)
		registerResponse.UUID = uuid
	}

	log.WithFields(log.Fields{
		"request":  registerRequest,
		"response": registerResponse,
	}).Info("Processing REST registration")

	if err := json.NewEncoder(w).Encode(registerResponse); err != nil {
		log.WithError(err).Warn("Failed to write REST registration response")
	}
}

// handleUnregister processes /unregister POST requests.
func (ra *RestAgent) handleUnregister(w http.ResponseWriter, r *http.Request) {
	var (
		unregisterRequest  RestUnregisterRequest
		unregisterResponse RestUnregisterResponse
	)

	if jsonErr := json.NewDecoder(r.Body).Decode(&unregisterRequest); jsonErr != nil {
		log.WithError(jsonErr).Warn("Failed to parse REST unregistration request")
	} else {
		log.WithField("uuid", unregisterRequest.UUID).Info("Unregister REST client")
		ra.clients.Delete(unregisterRequest.UUID)
		ra.mailbox.Delete(unregisterRequest.UUID)
	}

	if err := json.NewEncoder(w).Encode(unregisterResponse); err != nil {
		log.WithError(err).Warn("Failed to write REST unregistration response")
	}
}

func (ra *RestAgent) handleFetch(w http.ResponseWriter, r *http.Request) {
	var (
		fetchRequest  RestFetchRequest
		fetchResponse RestFetchResponse
	)

	if jsonErr := json.NewDecoder(r.Body).Decode(&fetchRequest); jsonErr != nil {
		log.WithError(jsonErr).Warn("Failed to parse REST fetch request")
		fetchResponse.Error = jsonErr.Error()
	} else if val, ok := ra.mailbox.Load(fetchRequest.UUID); ok {
		log.WithField("uuid", fetchRequest.UUID).Info("REST client fetches bundles")
		fetchResponse.Bundles = val.([]bundle.Bundle)

		ra.mailbox.Delete(fetchRequest.UUID)
	} else if !ok {
		log.WithField("uuid", fetchRequest.UUID).Debug("REST client has no new bundles to fetch")
		fetchResponse.Error = "No data"
	}

	if err := json.NewEncoder(w).Encode(fetchResponse); err != nil {
		log.WithError(err).Warn("Failed to write REST fetch response")
	}
}

func (ra *RestAgent) Endpoints() (eids []bundle.EndpointID) {
	ra.clients.Range(func(_, v interface{}) bool {
		eids = append(eids, v.(bundle.EndpointID))
		return false
	})
	return
}

func (ra *RestAgent) MessageReceiver() chan Message {
	return ra.receiver
}

func (ra *RestAgent) MessageSender() chan Message {
	return ra.sender
}
