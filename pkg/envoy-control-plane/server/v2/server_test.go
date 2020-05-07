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
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc"

	discovery "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/cache/types"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/cache/v2"
	rsrc "github.com/datawire/ambassador/pkg/envoy-control-plane/resource/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/server/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/test/resource/v2"
)

type mockConfigWatcher struct {
	counts     map[string]int
	responses  map[string][]cache.Response
	closeWatch bool
}

func (config *mockConfigWatcher) CreateWatch(req discovery.DiscoveryRequest) (chan cache.Response, func()) {
	config.counts[req.TypeUrl] = config.counts[req.TypeUrl] + 1
	out := make(chan cache.Response, 1)
	if len(config.responses[req.TypeUrl]) > 0 {
		out <- config.responses[req.TypeUrl][0]
		config.responses[req.TypeUrl] = config.responses[req.TypeUrl][1:]
	} else if config.closeWatch {
		close(out)
	}
	return out, func() {}
}

func (config *mockConfigWatcher) Fetch(ctx context.Context, req discovery.DiscoveryRequest) (*cache.Response, error) {
	if len(config.responses[req.TypeUrl]) > 0 {
		out := config.responses[req.TypeUrl][0]
		config.responses[req.TypeUrl] = config.responses[req.TypeUrl][1:]
		return &out, nil
	}
	return nil, errors.New("missing")
}

func makeMockConfigWatcher() *mockConfigWatcher {
	return &mockConfigWatcher{
		counts: make(map[string]int),
	}
}

type callbacks struct {
	fetchReq      int
	fetchResp     int
	callbackError bool
}

func (c *callbacks) OnStreamOpen(context.Context, int64, string) error {
	if c.callbackError {
		return errors.New("stream open error")
	}
	return nil
}
func (c *callbacks) OnStreamClosed(int64)                                     {}
func (c *callbacks) OnStreamRequest(int64, *discovery.DiscoveryRequest) error { return nil }
func (c *callbacks) OnStreamResponse(int64, *discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {
}
func (c *callbacks) OnFetchRequest(context.Context, *discovery.DiscoveryRequest) error {
	if c.callbackError {
		return errors.New("fetch request error")
	}
	c.fetchReq++
	return nil
}
func (c *callbacks) OnFetchResponse(*discovery.DiscoveryRequest, *discovery.DiscoveryResponse) {
	c.fetchResp++
}

type mockStream struct {
	t         *testing.T
	ctx       context.Context
	recv      chan *discovery.DiscoveryRequest
	sent      chan *discovery.DiscoveryResponse
	nonce     int
	sendError bool
	grpc.ServerStream
}

func (stream *mockStream) Context() context.Context {
	return stream.ctx
}

func (stream *mockStream) Send(resp *discovery.DiscoveryResponse) error {
	// check that nonce is monotonically incrementing
	stream.nonce = stream.nonce + 1
	if resp.Nonce != fmt.Sprintf("%d", stream.nonce) {
		stream.t.Errorf("Nonce => got %q, want %d", resp.Nonce, stream.nonce)
	}
	// check that version is set
	if resp.VersionInfo == "" {
		stream.t.Error("VersionInfo => got none, want non-empty")
	}
	// check resources are non-empty
	if len(resp.Resources) == 0 {
		stream.t.Error("Resources => got none, want non-empty")
	}
	// check that type URL matches in resources
	if resp.TypeUrl == "" {
		stream.t.Error("TypeUrl => got none, want non-empty")
	}
	for _, res := range resp.Resources {
		if res.TypeUrl != resp.TypeUrl {
			stream.t.Errorf("TypeUrl => got %q, want %q", res.TypeUrl, resp.TypeUrl)
		}
	}
	stream.sent <- resp
	if stream.sendError {
		return errors.New("send error")
	}
	return nil
}

func (stream *mockStream) Recv() (*discovery.DiscoveryRequest, error) {
	req, more := <-stream.recv
	if !more {
		return nil, errors.New("empty")
	}
	return req, nil
}

func makeMockStream(t *testing.T) *mockStream {
	return &mockStream{
		t:    t,
		ctx:  context.Background(),
		sent: make(chan *discovery.DiscoveryResponse, 10),
		recv: make(chan *discovery.DiscoveryRequest, 10),
	}
}

const (
	clusterName  = "cluster0"
	routeName    = "route0"
	listenerName = "listener0"
)

var (
	node = &core.Node{
		Id:      "test-id",
		Cluster: "test-cluster",
	}
	endpoint  = resource.MakeEndpoint(clusterName, 8080)
	cluster   = resource.MakeCluster(resource.Ads, clusterName)
	route     = resource.MakeRoute(routeName, clusterName)
	listener  = resource.MakeHTTPListener(resource.Ads, listenerName, 80, routeName)
	testTypes = []string{
		rsrc.EndpointType,
		rsrc.ClusterType,
		rsrc.RouteType,
		rsrc.ListenerType,
	}
)

func makeResponses() map[string][]cache.Response {
	return map[string][]cache.Response{
		rsrc.EndpointType: []cache.Response{{
			Version:   "1",
			Resources: []types.Resource{endpoint},
		}},
		rsrc.ClusterType: []cache.Response{{
			Version:   "2",
			Resources: []types.Resource{cluster},
		}},
		rsrc.RouteType: []cache.Response{{
			Version:   "3",
			Resources: []types.Resource{route},
		}},
		rsrc.ListenerType: []cache.Response{{
			Version:   "4",
			Resources: []types.Resource{listener},
		}},
	}
}

func TestServerShutdown(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.responses = makeResponses()
			shutdown := make(chan bool)
			ctx, cancel := context.WithCancel(context.Background())
			s := server.NewServer(ctx, config, &callbacks{})

			// make a request
			resp := makeMockStream(t)
			resp.recv <- &discovery.DiscoveryRequest{Node: node}
			go func() {
				var err error
				switch typ {
				case rsrc.EndpointType:
					err = s.StreamEndpoints(resp)
				case rsrc.ClusterType:
					err = s.StreamClusters(resp)
				case rsrc.RouteType:
					err = s.StreamRoutes(resp)
				case rsrc.ListenerType:
					err = s.StreamListeners(resp)
				}
				if err != nil {
					t.Errorf("Stream() => got %v, want no error", err)
				}
				shutdown <- true
			}()

			go func() {
				defer cancel()
			}()

			select {
			case <-shutdown:
			case <-time.After(1 * time.Second):
				t.Fatalf("got no response")
			}
		})
	}
}

func TestResponseHandlers(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.responses = makeResponses()
			s := server.NewServer(context.Background(), config, &callbacks{})

			// make a request
			resp := makeMockStream(t)
			resp.recv <- &discovery.DiscoveryRequest{Node: node}
			go func() {
				var err error
				switch typ {
				case rsrc.EndpointType:
					err = s.StreamEndpoints(resp)
				case rsrc.ClusterType:
					err = s.StreamClusters(resp)
				case rsrc.RouteType:
					err = s.StreamRoutes(resp)
				case rsrc.ListenerType:
					err = s.StreamListeners(resp)
				}
				if err != nil {
					t.Errorf("Stream() => got %v, want no error", err)
				}
			}()

			// check a response
			select {
			case <-resp.sent:
				close(resp.recv)
				if want := map[string]int{typ: 1}; !reflect.DeepEqual(want, config.counts) {
					t.Errorf("watch counts => got %v, want %v", config.counts, want)
				}
			case <-time.After(1 * time.Second):
				t.Fatalf("got no response")
			}
		})
	}
}

func TestFetch(t *testing.T) {
	config := makeMockConfigWatcher()
	config.responses = makeResponses()
	cb := &callbacks{}
	s := server.NewServer(context.Background(), config, cb)
	if out, err := s.FetchEndpoints(context.Background(), &discovery.DiscoveryRequest{Node: node}); out == nil || err != nil {
		t.Errorf("unexpected empty or error for endpoints: %v", err)
	}
	if out, err := s.FetchClusters(context.Background(), &discovery.DiscoveryRequest{Node: node}); out == nil || err != nil {
		t.Errorf("unexpected empty or error for clusters: %v", err)
	}
	if out, err := s.FetchRoutes(context.Background(), &discovery.DiscoveryRequest{Node: node}); out == nil || err != nil {
		t.Errorf("unexpected empty or error for routes: %v", err)
	}
	if out, err := s.FetchListeners(context.Background(), &discovery.DiscoveryRequest{Node: node}); out == nil || err != nil {
		t.Errorf("unexpected empty or error for listeners: %v", err)
	}

	// try again and expect empty results
	if out, err := s.FetchEndpoints(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil {
		t.Errorf("expected empty or error for endpoints: %v", err)
	}
	if out, err := s.FetchClusters(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil {
		t.Errorf("expected empty or error for clusters: %v", err)
	}
	if out, err := s.FetchRoutes(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil {
		t.Errorf("expected empty or error for routes: %v", err)
	}
	if out, err := s.FetchListeners(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil {
		t.Errorf("expected empty or error for listeners: %v", err)
	}

	// try empty requests: not valid in a real gRPC server
	if out, err := s.FetchEndpoints(context.Background(), nil); out != nil {
		t.Errorf("expected empty on empty request: %v", err)
	}
	if out, err := s.FetchClusters(context.Background(), nil); out != nil {
		t.Errorf("expected empty on empty request: %v", err)
	}
	if out, err := s.FetchRoutes(context.Background(), nil); out != nil {
		t.Errorf("expected empty on empty request: %v", err)
	}
	if out, err := s.FetchListeners(context.Background(), nil); out != nil {
		t.Errorf("expected empty on empty request: %v", err)
	}

	// send error from callback
	cb.callbackError = true
	if out, err := s.FetchEndpoints(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil || err == nil {
		t.Errorf("expected empty or error due to callback error")
	}
	if out, err := s.FetchClusters(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil || err == nil {
		t.Errorf("expected empty or error due to callback error")
	}
	if out, err := s.FetchRoutes(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil || err == nil {
		t.Errorf("expected empty or error due to callback error")
	}
	if out, err := s.FetchListeners(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil || err == nil {
		t.Errorf("expected empty or error due to callback error")
	}

	// verify fetch callbacks
	if want := 8; cb.fetchReq != want {
		t.Errorf("unexpected number of fetch requests: got %d, want %d", cb.fetchReq, want)
	}
	if want := 4; cb.fetchResp != want {
		t.Errorf("unexpected number of fetch responses: got %d, want %d", cb.fetchResp, want)
	}
}

func TestWatchClosed(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.closeWatch = true
			s := server.NewServer(context.Background(), config, &callbacks{})

			// make a request
			resp := makeMockStream(t)
			resp.recv <- &discovery.DiscoveryRequest{
				Node:    node,
				TypeUrl: typ,
			}

			// check that response fails since watch gets closed
			if err := s.StreamAggregatedResources(resp); err == nil {
				t.Error("Stream() => got no error, want watch failed")
			}

			close(resp.recv)
		})
	}
}

func TestSendError(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.responses = makeResponses()
			s := server.NewServer(context.Background(), config, &callbacks{})

			// make a request
			resp := makeMockStream(t)
			resp.sendError = true
			resp.recv <- &discovery.DiscoveryRequest{
				Node:    node,
				TypeUrl: typ,
			}

			// check that response fails since send returns error
			if err := s.StreamAggregatedResources(resp); err == nil {
				t.Error("Stream() => got no error, want send error")
			}

			close(resp.recv)
		})
	}
}

func TestStaleNonce(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.responses = makeResponses()
			s := server.NewServer(context.Background(), config, &callbacks{})

			resp := makeMockStream(t)
			resp.recv <- &discovery.DiscoveryRequest{
				Node:    node,
				TypeUrl: typ,
			}
			stop := make(chan struct{})
			go func() {
				if err := s.StreamAggregatedResources(resp); err != nil {
					t.Errorf("StreamAggregatedResources() => got %v, want no error", err)
				}
				// should be two watches called
				if want := map[string]int{typ: 2}; !reflect.DeepEqual(want, config.counts) {
					t.Errorf("watch counts => got %v, want %v", config.counts, want)
				}
				close(stop)
			}()
			select {
			case <-resp.sent:
				// stale request
				resp.recv <- &discovery.DiscoveryRequest{
					Node:          node,
					TypeUrl:       typ,
					ResponseNonce: "xyz",
				}
				// fresh request
				resp.recv <- &discovery.DiscoveryRequest{
					VersionInfo:   "1",
					Node:          node,
					TypeUrl:       typ,
					ResponseNonce: "1",
				}
				close(resp.recv)
			case <-time.After(1 * time.Second):
				t.Fatalf("got %d messages on the stream, not 4", resp.nonce)
			}
			<-stop
		})
	}
}

func TestAggregatedHandlers(t *testing.T) {
	config := makeMockConfigWatcher()
	config.responses = makeResponses()
	resp := makeMockStream(t)

	resp.recv <- &discovery.DiscoveryRequest{
		Node:    node,
		TypeUrl: rsrc.ListenerType,
	}
	resp.recv <- &discovery.DiscoveryRequest{
		Node:    node,
		TypeUrl: rsrc.ClusterType,
	}
	resp.recv <- &discovery.DiscoveryRequest{
		Node:          node,
		TypeUrl:       rsrc.EndpointType,
		ResourceNames: []string{clusterName},
	}
	resp.recv <- &discovery.DiscoveryRequest{
		Node:          node,
		TypeUrl:       rsrc.RouteType,
		ResourceNames: []string{routeName},
	}

	s := server.NewServer(context.Background(), config, &callbacks{})
	go func() {
		if err := s.StreamAggregatedResources(resp); err != nil {
			t.Errorf("StreamAggregatedResources() => got %v, want no error", err)
		}
	}()

	count := 0
	for {
		select {
		case <-resp.sent:
			count++
			if count >= 4 {
				close(resp.recv)
				if want := map[string]int{
					rsrc.EndpointType: 1,
					rsrc.ClusterType:  1,
					rsrc.RouteType:    1,
					rsrc.ListenerType: 1,
				}; !reflect.DeepEqual(want, config.counts) {
					t.Errorf("watch counts => got %v, want %v", config.counts, want)
				}

				// got all messages
				return
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("got %d messages on the stream, not 4", count)
		}
	}
}

func TestAggregateRequestType(t *testing.T) {
	config := makeMockConfigWatcher()
	s := server.NewServer(context.Background(), config, &callbacks{})
	resp := makeMockStream(t)
	resp.recv <- &discovery.DiscoveryRequest{Node: node}
	if err := s.StreamAggregatedResources(resp); err == nil {
		t.Error("StreamAggregatedResources() => got nil, want an error")
	}
}

func TestCallbackError(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.responses = makeResponses()
			s := server.NewServer(context.Background(), config, &callbacks{callbackError: true})

			// make a request
			resp := makeMockStream(t)
			resp.recv <- &discovery.DiscoveryRequest{
				Node:    node,
				TypeUrl: typ,
			}

			// check that response fails since stream open returns error
			if err := s.StreamAggregatedResources(resp); err == nil {
				t.Error("Stream() => got no error, want error")
			}

			close(resp.recv)
		})
	}
}
