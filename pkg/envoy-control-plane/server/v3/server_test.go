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

	"github.com/stretchr/testify/assert"

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
	deltaResources map[string]map[string]types.Resource
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
	assert.Equal(stream.t, resp.Nonce, fmt.Sprintf("%d", stream.nonce))
	// check that version is set
	assert.NotEmpty(stream.t, resp.VersionInfo)
	// check resources are non-empty
	assert.NotEmpty(stream.t, resp.Resources)
	// check that type URL matches in resources
	assert.NotEmpty(stream.t, resp.TypeUrl)

	for _, res := range resp.Resources {
		assert.Equal(stream.t, res.TypeUrl, resp.TypeUrl)
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
	virtualHostName     = "virtualHost0"
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
	route              = resource.MakeRouteConfig(routeName, clusterName)
	scopedRoute        = resource.MakeScopedRouteConfig(scopedRouteName, routeName, []string{"127.0.0.1"})
	virtualHost        = resource.MakeVirtualHost(virtualHostName, clusterName)
	httpListener       = resource.MakeRouteHTTPListener(resource.Ads, listenerName, 80, routeName)
	httpScopedListener = resource.MakeScopedRouteHTTPListener(resource.Ads, scopedListenerName, 80)
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
		rsrc.VirtualHostType: {
			&cache.RawResponse{
				Version:   "5",
				Resources: []types.ResourceWithTTL{{Resource: virtualHost}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.VirtualHostType},
			},
		},
		rsrc.ListenerType: {
			&cache.RawResponse{
				Version:   "6",
				Resources: []types.ResourceWithTTL{{Resource: httpListener}, {Resource: httpScopedListener}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.ListenerType},
			},
		},
		rsrc.SecretType: {
			&cache.RawResponse{
				Version:   "7",
				Resources: []types.ResourceWithTTL{{Resource: secret}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.SecretType},
			},
		},
		rsrc.RuntimeType: {
			&cache.RawResponse{
				Version:   "8",
				Resources: []types.ResourceWithTTL{{Resource: runtime}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.RuntimeType},
			},
		},
		rsrc.ExtensionConfigType: {
			&cache.RawResponse{
				Version:   "9",
				Resources: []types.ResourceWithTTL{{Resource: extensionConfig}},
				Request:   &discovery.DiscoveryRequest{TypeUrl: rsrc.ExtensionConfigType},
			},
		},
		// Pass-through type (xDS does not exist for this type)
		opaqueType: {
			&cache.RawResponse{
				Version:   "10",
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
			done := make(chan struct{})
			ctx, cancel := context.WithCancel(context.Background())

			config := makeMockConfigWatcher()
			config.responses = makeResponses()
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
				assert.NoError(t, err)
				close(done)
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

			cancel()
			<-done
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
	out, err := s.FetchEndpoints(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.NotNil(t, out)
	assert.NoError(t, err)

	out, err = s.FetchClusters(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.NotNil(t, out)
	assert.NoError(t, err)

	out, err = s.FetchRoutes(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.NotNil(t, out)
	assert.NoError(t, err)

	out, err = s.FetchListeners(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.NotNil(t, out)
	assert.NoError(t, err)

	out, err = s.FetchSecrets(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.NotNil(t, out)
	assert.NoError(t, err)

	out, err = s.FetchRuntime(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.NotNil(t, out)
	assert.NoError(t, err)

	// try again and expect empty results
	out, err = s.FetchEndpoints(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchClusters(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchRoutes(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchListeners(context.Background(), &discovery.DiscoveryRequest{Node: node})
	assert.Nil(t, out)
	assert.Error(t, err)

	// try empty requests: not valid in a real gRPC server
	out, err = s.FetchEndpoints(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchClusters(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchRoutes(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchListeners(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchSecrets(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchRuntime(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	// send error from callback
	callbackError = true
	out, err = s.FetchEndpoints(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchClusters(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchRoutes(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	out, err = s.FetchListeners(context.Background(), nil)
	assert.Nil(t, out)
	assert.Error(t, err)

	// verify fetch callbacks
	assert.Equal(t, requestCount, 10)
	assert.Equal(t, responseCount, 6)
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
			err := s.StreamAggregatedResources(resp)
			assert.Error(t, err)

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
				err := s.StreamAggregatedResources(resp)
				assert.NoError(t, err)
				// should be two watches called
				assert.False(t, !reflect.DeepEqual(map[string]int{typ: 2}, config.counts))
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
		TypeUrl:       rsrc.ExtensionConfigType,
		ResourceNames: []string{extensionConfigName},
	}
	resp.recv <- &discovery.DiscoveryRequest{
		TypeUrl:       rsrc.ScopedRouteType,
		ResourceNames: []string{scopedRouteName},
	}
	resp.recv <- &discovery.DiscoveryRequest{
		TypeUrl:       rsrc.VirtualHostType,
		ResourceNames: []string{virtualHostName},
	}

	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	go func() {
		err := s.StreamAggregatedResources(resp)
		assert.NoError(t, err)
	}()

	count := 0
	expectedCount := 7
	for {
		select {
		case <-resp.sent:
			count++
			if count >= expectedCount {
				close(resp.recv)
				assert.False(t, !reflect.DeepEqual(map[string]int{
					rsrc.EndpointType:        1,
					rsrc.ClusterType:         1,
					rsrc.RouteType:           1,
					rsrc.ScopedRouteType:     1,
					rsrc.VirtualHostType:     1,
					rsrc.ListenerType:        1,
					rsrc.ExtensionConfigType: 1,
				}, config.counts))

				// got all messages
				return
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("got %d messages on the stream, not %d", count, expectedCount)
		}
	}
}

func TestAggregateRequestType(t *testing.T) {
	config := makeMockConfigWatcher()
	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	resp := makeMockStream(t)
	resp.recv <- &discovery.DiscoveryRequest{Node: node}
	err := s.StreamAggregatedResources(resp)
	assert.Error(t, err)
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
	err := s.StreamAggregatedResources(resp)
	assert.NoError(t, err)
	assert.Equal(t, config.watches, 0)
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
	err := s.StreamAggregatedResources(resp)
	assert.NoError(t, err)
	assert.Equal(t, config.watches, 0)
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
			err := s.StreamAggregatedResources(resp)
			assert.Error(t, err)

			close(resp.recv)
		})
	}
}
