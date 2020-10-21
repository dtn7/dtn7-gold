// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import "github.com/dtn7/dtn7-go/pkg/bpv7"

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
type RestUnregisterResponse struct {
	Error string `json:"error"`
}

// RestFetchRequest describes a JSON to be POSTed to /fetch.
type RestFetchRequest struct {
	UUID string `json:"uuid"`
}

// RestFetchResponse describes a JSON response for /fetch.
type RestFetchResponse struct {
	Error   string        `json:"error"`
	Bundles []bpv7.Bundle `json:"bundles"`
}

// RestBuildRequest describes a JSON to be POSTed to /build.
type RestBuildRequest struct {
	UUID string                 `json:"uuid"`
	Args map[string]interface{} `json:"arguments"`
}

// RestBuildResponse describes a JSON response for /build.
type RestBuildResponse struct {
	Error string `json:"error"`
}
