package agent

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
