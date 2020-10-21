// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package agent

import (
	"testing"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
)

func TestAppAgentContainsEndpoint(t *testing.T) {
	appAgent := newMockAgent([]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://foo/"), bpv7.MustNewEndpointID("dtn://bar/")})

	tests := []struct {
		eids  []bpv7.EndpointID
		valid bool
	}{
		{[]bpv7.EndpointID{}, false},
		{[]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://foo/")}, true},
		{[]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://bar/")}, true},
		{[]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://foo/"), bpv7.MustNewEndpointID("dtn://bar/")}, true},
		{[]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://bar/"), bpv7.MustNewEndpointID("dtn://foo/")}, true},
		{[]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://bar/"), bpv7.MustNewEndpointID("dtn://bar/")}, true},
		{[]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://baz/"), bpv7.MustNewEndpointID("dtn://bar/")}, true},
		{[]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://baz/"), bpv7.MustNewEndpointID("dtn://ban/")}, false},
		{[]bpv7.EndpointID{bpv7.MustNewEndpointID("dtn://baz/"), bpv7.MustNewEndpointID("dtn://ban/"), bpv7.MustNewEndpointID("dtn://bar/")}, true},
	}

	for _, test := range tests {
		contains := AppAgentContainsEndpoint(appAgent, test.eids)
		if contains != test.valid {
			t.Fatalf("errored for %v", test.eids)
		}
	}
}
