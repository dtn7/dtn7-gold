package agent

import "github.com/dtn7/dtn7-go/bundle"

// RestRegisterRequest describes a JSON to be POSTed to /register.
type RestRegisterRequest struct {
	EndpointId string `json:"endpoint_id"`
}

// RestRegisterResponse describes a JSON response for /register.
type RestRegisterResponse struct {
	Error string `json:"error"`
	UUID  string `json:"uuid"`
}

// RestUnregisterRequest describes a JSON to be POSTed to /unregister.
type RestUnregisterRequest struct {
	UUID string `json:"uuid"`
}

// RestUnregisterResponse describes a JSON response for /unregister.
type RestUnregisterResponse struct{}

// RestFetchRequest describes a JSON to be POSTed to /fetch.
type RestFetchRequest struct {
	UUID string `json:"uuid"`
}

// RestFetchResponse describes a JSON response for /fetch.
type RestFetchResponse struct {
	Error   string          `json:"error"`
	Bundles []bundle.Bundle `json:"bundles"`
}
