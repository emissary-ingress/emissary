// Copyright 2018 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package server_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v3"
)

func TestGateway(t *testing.T) {
	config := makeMockConfigWatcher()
	config.responses = map[string][]cache.Response{
		resource.ClusterType: {
			&cache.RawResponse{
				Version:   "2",
				Resources: []types.ResourceWithTTL{{Resource: cluster}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: resource.ClusterType},
			},
		},
		resource.RouteType: {
			&cache.RawResponse{
				Version:   "3",
				Resources: []types.ResourceWithTTL{{Resource: route}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: resource.RouteType},
			},
		},
		resource.ListenerType: {
			&cache.RawResponse{
				Version:   "4",
				Resources: []types.ResourceWithTTL{{Resource: httpListener}, {Resource: httpScopedListener}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: resource.ListenerType},
			},
		},
	}
	gtw := server.HTTPGateway{Server: server.NewServer(context.Background(), config, nil)}

	failCases := []struct {
		path   string
		body   io.Reader
		expect int
	}{
		{
			path:   "/hello/",
			expect: http.StatusNotFound,
		},
		{
			path:   resource.FetchEndpoints,
			expect: http.StatusBadRequest,
		},
		{
			path:   resource.FetchEndpoints,
			body:   iotest.TimeoutReader(strings.NewReader("hello")),
			expect: http.StatusBadRequest,
		},
		{
			path:   resource.FetchEndpoints,
			body:   strings.NewReader("hello"),
			expect: http.StatusBadRequest,
		},
		{
			// missing response
			path:   resource.FetchEndpoints,
			body:   strings.NewReader("{\"node\": {\"id\": \"test\"}}"),
			expect: http.StatusInternalServerError,
		},
	}
	for _, cs := range failCases {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, cs.path, cs.body)
		if err != nil {
			t.Fatal(err)
		}
		resp, code, err := gtw.ServeHTTP(req)
		if err == nil {
			t.Errorf("ServeHTTP succeeded, but should have failed")
		}
		if resp != nil {
			t.Errorf("handler returned wrong response")
		}
		if status := code; status != cs.expect {
			t.Errorf("handler returned wrong status: %d, want %d", status, cs.expect)
		}
	}

	for _, path := range []string{resource.FetchClusters, resource.FetchRoutes, resource.FetchListeners} {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, path, strings.NewReader("{\"node\": {\"id\": \"test\"}}"))
		if err != nil {
			t.Fatal(err)
		}
		resp, code, err := gtw.ServeHTTP(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp == nil {
			t.Errorf("handler returned wrong response")
		}
		if status := code; status != 200 {
			t.Errorf("handler returned wrong status: %d, want %d", status, 200)
		}
	}
}
