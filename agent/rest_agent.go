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

	// map UUIDs to EIDs and received bundles
	clients sync.Map // uuid[string] -> bundle.EndpointID
	mailbox sync.Map // uuid[string] -> []bundle.Bundle
}

// NewRestAgent creates a new RESTful Application Agent.
func NewRestAgent(router *mux.Router) (ra *RestAgent) {
	ra = &RestAgent{
		router: router,
	}

	ra.router.HandleFunc("/register", ra.handleRegister).Methods(http.MethodPost)
	ra.router.HandleFunc("/unregister", ra.handleUnregister).Methods(http.MethodPost)

	return ra
}

// ServeHTTP is a http.Handler to be bound to a HTTP endpoint, e.g., /rest.
func (ra *RestAgent) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ra.router.ServeHTTP(w, r)
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

func (ra *RestAgent) Endpoints() (eids []bundle.EndpointID) {
	ra.clients.Range(func(_, v interface{}) bool {
		eids = append(eids, v.(bundle.EndpointID))
		return false
	})
	return
}

func (ra *RestAgent) MessageReceiver() chan Message {
	panic("implement me")
}

func (ra *RestAgent) MessageSender() chan Message {
	panic("implement me")
}
