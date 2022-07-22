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
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"

	core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	rsrc "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/stream/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/test/resource/v3"
)

type mockConfigWatcher struct {
	counts         map[string]int
	deltaCounts    map[string]int
	responses      map[string][]cache.Response
	deltaResponses map[string][]cache.DeltaResponse
	watches        int
	deltaWatches   int

	mu *sync.RWMutex
}

func (config *mockConfigWatcher) CreateWatch(req *discovery.DiscoveryRequest, state stream.StreamState, out chan cache.Response) func() {
	config.counts[req.TypeUrl] = config.counts[req.TypeUrl] + 1
	if len(config.responses[req.TypeUrl]) > 0 {
		out <- config.responses[req.TypeUrl][0]
		config.responses[req.TypeUrl] = config.responses[req.TypeUrl][1:]
	} else {
		config.watches++
		return func() {
			config.watches--
		}
	}
	return nil
}

func (config *mockConfigWatcher) Fetch(ctx context.Context, req *discovery.DiscoveryRequest) (cache.Response, error) {
	if len(config.responses[req.TypeUrl]) > 0 {
		out := config.responses[req.TypeUrl][0]
		config.responses[req.TypeUrl] = config.responses[req.TypeUrl][1:]
		return out, nil
	}
	return nil, errors.New("missing")
}

func makeMockConfigWatcher() *mockConfigWatcher {
	return &mockConfigWatcher{
		counts:      make(map[string]int),
		deltaCounts: make(map[string]int),
		mu:          &sync.RWMutex{},
	}
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
	clusterName         = "cluster0"
	routeName           = "route0"
	scopedRouteName     = "scopedRoute0"
	listenerName        = "listener0"
	scopedListenerName  = "scopedListener0"
	secretName          = "secret0"
	runtimeName         = "runtime0"
	extensionConfigName = "extensionConfig0"
)

var (
	node = &core.Node{
		Id:      "test-id",
		Cluster: "test-cluster",
	}
	endpoint           = resource.MakeEndpoint(clusterName, 8080)
	cluster            = resource.MakeCluster(resource.Ads, clusterName)
	route              = resource.MakeRoute(routeName, clusterName)
	scopedRoute        = resource.MakeScopedRoute(scopedRouteName, routeName, []string{"127.0.0.1"})
	httpListener       = resource.MakeRouteHTTPListener(resource.Ads, listenerName, 80, routeName)
	httpScopedListener = resource.MakeScopedRouteHTTPListener(resource.Ads, scopedListenerName, 80, scopedRouteName)
	secret             = resource.MakeSecrets(secretName, "test")[0]
	runtime            = resource.MakeRuntime(runtimeName)
	extensionConfig    = resource.MakeExtensionConfig(resource.Ads, extensionConfigName, routeName)
	opaque             = &core.Address{}
	opaqueType         = "unknown-type"
	testTypes          = []string{
		rsrc.EndpointType,
		rsrc.ClusterType,
		rsrc.RouteType,
		rsrc.ScopedRouteType,
		rsrc.ListenerType,
		rsrc.SecretType,
		rsrc.RuntimeType,
		rsrc.ExtensionConfigType,
		opaqueType,
	}
)

func makeResponses() map[string][]cache.Response {
	return map[string][]cache.Response{
		rsrc.EndpointType: {
			&cache.RawResponse{
				Version:   "1",
				Resources: []types.ResourceWithTTL{{Resource: endpoint}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.EndpointType},
			},
		},
		rsrc.ClusterType: {
			&cache.RawResponse{
				Version:   "2",
				Resources: []types.ResourceWithTTL{{Resource: cluster}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.ClusterType},
			},
		},
		rsrc.RouteType: {
			&cache.RawResponse{
				Version:   "3",
				Resources: []types.ResourceWithTTL{{Resource: route}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.RouteType},
			},
		},
		rsrc.ScopedRouteType: {
			&cache.RawResponse{
				Version:   "4",
				Resources: []types.ResourceWithTTL{{Resource: scopedRoute}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.ScopedRouteType},
			},
		},
		rsrc.ListenerType: {
			&cache.RawResponse{
				Version:   "5",
				Resources: []types.ResourceWithTTL{{Resource: httpListener}, {Resource: httpScopedListener}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.ListenerType},
			},
		},
		rsrc.SecretType: {
			&cache.RawResponse{
				Version:   "6",
				Resources: []types.ResourceWithTTL{{Resource: secret}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.SecretType},
			},
		},
		rsrc.RuntimeType: {
			&cache.RawResponse{
				Version:   "7",
				Resources: []types.ResourceWithTTL{{Resource: runtime}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.RuntimeType},
			},
		},
		rsrc.ExtensionConfigType: {
			&cache.RawResponse{
				Version:   "8",
				Resources: []types.ResourceWithTTL{{Resource: extensionConfig}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.ExtensionConfigType},
			},
		},
		// Pass-through type (xDS does not exist for this type)
		opaqueType: {
			&cache.RawResponse{
				Version:   "9",
				Resources: []types.ResourceWithTTL{{Resource: opaque}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: opaqueType},
			},
		},
	}
}

func TestServerShutdown(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.responses = makeResponses()
			shutdown := make(chan bool)
			ctx, cancel := context.WithCancel(context.Background())
			s := server.NewServer(ctx, config, server.CallbackFuncs{})

			// make a request
			resp := makeMockStream(t)
			resp.recv <- &discovery.DiscoveryRequest{Node: node, TypeUrl: typ}
			go func(rType string) {
				var err error
				switch rType {
				case rsrc.EndpointType:
					err = s.StreamEndpoints(resp)
				case rsrc.ClusterType:
					err = s.StreamClusters(resp)
				case rsrc.RouteType:
					err = s.StreamRoutes(resp)
				case rsrc.ScopedRouteType:
					err = s.StreamScopedRoutes(resp)
				case rsrc.ListenerType:
					err = s.StreamListeners(resp)
				case rsrc.SecretType:
					err = s.StreamSecrets(resp)
				case rsrc.RuntimeType:
					err = s.StreamRuntime(resp)
				case rsrc.ExtensionConfigType:
					err = s.StreamExtensionConfigs(resp)
				case opaqueType:
					err = s.StreamAggregatedResources(resp)
				}
				if err != nil {
					t.Errorf("Stream() => got %v, want no error", err)
				}
				shutdown <- true
			}(typ)

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
			s := server.NewServer(context.Background(), config, server.CallbackFuncs{})

			// make a request
			resp := makeMockStream(t)
			resp.recv <- &discovery.DiscoveryRequest{Node: node, TypeUrl: typ}
			go func(rType string) {
				var err error
				switch rType {
				case rsrc.EndpointType:
					err = s.StreamEndpoints(resp)
				case rsrc.ClusterType:
					err = s.StreamClusters(resp)
				case rsrc.RouteType:
					err = s.StreamRoutes(resp)
				case rsrc.ScopedRouteType:
					err = s.StreamScopedRoutes(resp)
				case rsrc.ListenerType:
					err = s.StreamListeners(resp)
				case rsrc.SecretType:
					err = s.StreamSecrets(resp)
				case rsrc.RuntimeType:
					err = s.StreamRuntime(resp)
				case rsrc.ExtensionConfigType:
					err = s.StreamExtensionConfigs(resp)
				case opaqueType:
					err = s.StreamAggregatedResources(resp)
				}
				if err != nil {
					t.Errorf("Stream() => got %v, want no error", err)
				}
			}(typ)

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

	requestCount := 0
	responseCount := 0
	callbackError := false

	cb := server.CallbackFuncs{
		StreamOpenFunc: func(ctx context.Context, i int64, s string) error {
			if callbackError {
				return errors.New("stream open error")
			}
			return nil
		},
		FetchRequestFunc: func(ctx context.Context, request *discovery.DiscoveryRequest) error {
			if callbackError {
				return errors.New("fetch request error")
			}
			requestCount++
			return nil
		},
		FetchResponseFunc: func(request *discovery.DiscoveryRequest, response *discovery.DiscoveryResponse) {
			responseCount++
		},
	}

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
	if out, err := s.FetchScopedRoutes(context.Background(), &discovery.DiscoveryRequest{Node: node}); out == nil || err != nil {
		t.Errorf("unexpected empty or error for scopedRoutes: %v", err)
	}
	if out, err := s.FetchListeners(context.Background(), &discovery.DiscoveryRequest{Node: node}); out == nil || err != nil {
		t.Errorf("unexpected empty or error for listeners: %v", err)
	}
	if out, err := s.FetchSecrets(context.Background(), &discovery.DiscoveryRequest{Node: node}); out == nil || err != nil {
		t.Errorf("unexpected empty or error for secrets: %v", err)
	}
	if out, err := s.FetchRuntime(context.Background(), &discovery.DiscoveryRequest{Node: node}); out == nil || err != nil {
		t.Errorf("unexpected empty or error for runtime: %v", err)
	}
	if out, err := s.FetchExtensionConfigs(context.Background(), &discovery.DiscoveryRequest{Node: node}); out == nil || err != nil {
		t.Errorf("unexpected empty or error for extensionConfigs: %v", err)
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
	if out, err := s.FetchScopedRoutes(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil {
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
	if out, err := s.FetchScopedRoutes(context.Background(), nil); out != nil {
		t.Errorf("expected empty on empty request: %v", err)
	}
	if out, err := s.FetchListeners(context.Background(), nil); out != nil {
		t.Errorf("expected empty on empty request: %v", err)
	}
	if out, err := s.FetchSecrets(context.Background(), nil); out != nil {
		t.Errorf("expected empty on empty request: %v", err)
	}
	if out, err := s.FetchRuntime(context.Background(), nil); out != nil {
		t.Errorf("expected empty on empty request: %v", err)
	}
	if out, err := s.FetchExtensionConfigs(context.Background(), nil); out != nil {
		t.Errorf("expected empty on empty request: %v", err)
	}

	// send error from callback
	callbackError = true
	if out, err := s.FetchEndpoints(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil || err == nil {
		t.Errorf("expected empty or error due to callback error")
	}
	if out, err := s.FetchClusters(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil || err == nil {
		t.Errorf("expected empty or error due to callback error")
	}
	if out, err := s.FetchRoutes(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil || err == nil {
		t.Errorf("expected empty or error due to callback error")
	}
	if out, err := s.FetchScopedRoutes(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil || err == nil {
		t.Errorf("expected empty or error due to callback error")
	}
	if out, err := s.FetchListeners(context.Background(), &discovery.DiscoveryRequest{Node: node}); out != nil || err == nil {
		t.Errorf("expected empty or error due to callback error")
	}

	// verify fetch callbacks
	if want := 13; requestCount != want {
		t.Errorf("unexpected number of fetch requests: got %d, want %d", requestCount, want)
	}
	if want := 8; responseCount != want {
		t.Errorf("unexpected number of fetch responses: got %d, want %d", responseCount, want)
	}
}

func TestSendError(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.responses = makeResponses()
			s := server.NewServer(context.Background(), config, server.CallbackFuncs{})

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
			s := server.NewServer(context.Background(), config, server.CallbackFuncs{})

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
	// Delta compress node
	resp.recv <- &discovery.DiscoveryRequest{
		TypeUrl: rsrc.ClusterType,
	}
	resp.recv <- &discovery.DiscoveryRequest{
		TypeUrl:       rsrc.EndpointType,
		ResourceNames: []string{clusterName},
	}
	resp.recv <- &discovery.DiscoveryRequest{
		TypeUrl:       rsrc.RouteType,
		ResourceNames: []string{routeName},
	}
	resp.recv <- &discovery.DiscoveryRequest{
		TypeUrl:       rsrc.ScopedRouteType,
		ResourceNames: []string{scopedRouteName},
	}

	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
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
			if count >= 5 {
				close(resp.recv)
				if want := map[string]int{
					rsrc.EndpointType:    1,
					rsrc.ClusterType:     1,
					rsrc.RouteType:       1,
					rsrc.ScopedRouteType: 1,
					rsrc.ListenerType:    1,
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
	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	resp := makeMockStream(t)
	resp.recv <- &discovery.DiscoveryRequest{Node: node}
	if err := s.StreamAggregatedResources(resp); err == nil {
		t.Error("StreamAggregatedResources() => got nil, want an error")
	}
}

func TestCancellations(t *testing.T) {
	config := makeMockConfigWatcher()
	resp := makeMockStream(t)
	for _, typ := range testTypes {
		resp.recv <- &discovery.DiscoveryRequest{
			Node:    node,
			TypeUrl: typ,
		}
	}
	close(resp.recv)
	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	if err := s.StreamAggregatedResources(resp); err != nil {
		t.Errorf("StreamAggregatedResources() => got %v, want no error", err)
	}
	if config.watches != 0 {
		t.Errorf("Expect all watches canceled, got %q", config.watches)
	}
}

func TestOpaqueRequestsChannelMuxing(t *testing.T) {
	config := makeMockConfigWatcher()
	resp := makeMockStream(t)
	for i := 0; i < 10; i++ {
		resp.recv <- &discovery.DiscoveryRequest{
			Node:    node,
			TypeUrl: fmt.Sprintf("%s%d", opaqueType, i%2),
			// each subsequent request is assumed to supercede the previous request
			ResourceNames: []string{fmt.Sprintf("%d", i)},
		}
	}
	close(resp.recv)
	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	if err := s.StreamAggregatedResources(resp); err != nil {
		t.Errorf("StreamAggregatedResources() => got %v, want no error", err)
	}
	if config.watches != 0 {
		t.Errorf("Expect all watches canceled, got %q", config.watches)
	}
}

func TestCallbackError(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.responses = makeResponses()

			s := server.NewServer(context.Background(), config, server.CallbackFuncs{
				StreamOpenFunc: func(ctx context.Context, i int64, s string) error {
					return errors.New("stream open error")
				},
			})

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
