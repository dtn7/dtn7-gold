// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"testing"

	"github.com/dtn7/dtn7-go/bundle"
)

func TestAppAgentContainsEndpoint(t *testing.T) {
	appAgent := newMockAgent([]bundle.EndpointID{bundle.MustNewEndpointID("dtn://foo/"), bundle.MustNewEndpointID("dtn://bar/")})

	tests := []struct {
		eids  []bundle.EndpointID
		valid bool
	}{
		{[]bundle.EndpointID{}, false},
		{[]bundle.EndpointID{bundle.MustNewEndpointID("dtn://foo/")}, true},
		{[]bundle.EndpointID{bundle.MustNewEndpointID("dtn://bar/")}, true},
		{[]bundle.EndpointID{bundle.MustNewEndpointID("dtn://foo/"), bundle.MustNewEndpointID("dtn://bar/")}, true},
		{[]bundle.EndpointID{bundle.MustNewEndpointID("dtn://bar/"), bundle.MustNewEndpointID("dtn://foo/")}, true},
		{[]bundle.EndpointID{bundle.MustNewEndpointID("dtn://bar/"), bundle.MustNewEndpointID("dtn://bar/")}, true},
		{[]bundle.EndpointID{bundle.MustNewEndpointID("dtn://baz/"), bundle.MustNewEndpointID("dtn://bar/")}, true},
		{[]bundle.EndpointID{bundle.MustNewEndpointID("dtn://baz/"), bundle.MustNewEndpointID("dtn://ban/")}, false},
		{[]bundle.EndpointID{bundle.MustNewEndpointID("dtn://baz/"), bundle.MustNewEndpointID("dtn://ban/"), bundle.MustNewEndpointID("dtn://bar/")}, true},
	}

	for _, test := range tests {
		contains := AppAgentContainsEndpoint(appAgent, test.eids)
		if contains != test.valid {
			t.Fatalf("errored for %v", test.eids)
		}
	}
}
