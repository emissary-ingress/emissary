package server_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"

	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	rsrc "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/stream/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v3"
)

func (config *mockConfigWatcher) CreateDeltaWatch(req *discovery.DeltaDiscoveryRequest, state stream.StreamState, out chan cache.DeltaResponse) func() {
	config.deltaCounts[req.TypeUrl] = config.deltaCounts[req.TypeUrl] + 1

	if len(config.deltaResponses[req.TypeUrl]) > 0 {
		res := config.deltaResponses[req.TypeUrl][0]
		// In subscribed, we only want to send back what's changed if we detect changes
		var subscribed []types.Resource
		r, _ := res.GetDeltaDiscoveryResponse()

		switch {
		case state.IsWildcard():
			for _, resource := range r.Resources {
				name := resource.GetName()
				res, _ := cache.MarshalResource(resource)

				nextVersion := cache.HashResource(res)
				prevVersion, found := state.GetResourceVersions()[name]
				if !found || (prevVersion != nextVersion) {
					state.GetResourceVersions()[name] = nextVersion
					subscribed = append(subscribed, resource)
				}
			}
		default:
			for _, resource := range r.Resources {
				res, _ := cache.MarshalResource(resource)
				nextVersion := cache.HashResource(res)
				for _, prevVersion := range state.GetResourceVersions() {
					if prevVersion != nextVersion {
						subscribed = append(subscribed, resource)
					}
					state.GetResourceVersions()[resource.GetName()] = nextVersion
				}
			}
		}

		out <- &cache.RawDeltaResponse{
			DeltaRequest:      req,
			Resources:         subscribed,
			SystemVersionInfo: "",
			NextVersionMap:    state.GetResourceVersions(),
		}
	} else {
		config.deltaWatches++
		return func() {
			config.deltaWatches--
		}
	}

	return nil
}

type mockDeltaStream struct {
	t         *testing.T
	ctx       context.Context
	recv      chan *discovery.DeltaDiscoveryRequest
	sent      chan *discovery.DeltaDiscoveryResponse
	nonce     int
	sendError bool
	grpc.ServerStream
}

func (stream *mockDeltaStream) Context() context.Context {
	return stream.ctx
}

func (stream *mockDeltaStream) Send(resp *discovery.DeltaDiscoveryResponse) error {
	// Check that nonce is incremented by one
	stream.nonce = stream.nonce + 1
	if resp.Nonce != fmt.Sprintf("%d", stream.nonce) {
		stream.t.Errorf("Nonce => got %q, want %d", resp.Nonce, stream.nonce)
	}
	// Check that resources are non-empty
	if len(resp.Resources) == 0 {
		stream.t.Error("Resources => got none, want non-empty")
	}
	if resp.TypeUrl == "" {
		stream.t.Error("TypeUrl => got none, want non-empty")
	}

	// Check that the per resource TypeURL is correctly set.
	for _, res := range resp.Resources {
		if res.Resource.TypeUrl != resp.TypeUrl {
			stream.t.Errorf("TypeUrl => got %q, want %q", res.Resource.TypeUrl, resp.TypeUrl)
		}
	}

	stream.sent <- resp
	if stream.sendError {
		return errors.New("send error")
	}
	return nil
}

func (stream *mockDeltaStream) Recv() (*discovery.DeltaDiscoveryRequest, error) {
	req, more := <-stream.recv
	if !more {
		return nil, errors.New("empty")
	}
	return req, nil
}

func makeMockDeltaStream(t *testing.T) *mockDeltaStream {
	return &mockDeltaStream{
		t:    t,
		ctx:  context.Background(),
		sent: make(chan *discovery.DeltaDiscoveryResponse, 10),
		recv: make(chan *discovery.DeltaDiscoveryRequest, 10),
	}
}

func makeDeltaResponses() map[string][]cache.DeltaResponse {
	return map[string][]cache.DeltaResponse{
		rsrc.EndpointType: {
			&cache.RawDeltaResponse{
				Resources:         []types.Resource{endpoint},
				DeltaRequest:      &discovery.DeltaDiscoveryRequest{TypeUrl: rsrc.EndpointType},
				SystemVersionInfo: "1",
			},
		},
		rsrc.ClusterType: {
			&cache.RawDeltaResponse{
				Resources:         []types.Resource{cluster},
				DeltaRequest:      &discovery.DeltaDiscoveryRequest{TypeUrl: rsrc.ClusterType},
				SystemVersionInfo: "2",
			},
		},
		rsrc.RouteType: {
			&cache.RawDeltaResponse{
				Resources:         []types.Resource{route},
				DeltaRequest:      &discovery.DeltaDiscoveryRequest{TypeUrl: rsrc.RouteType},
				SystemVersionInfo: "3",
			},
		},
		rsrc.ScopedRouteType: {
			&cache.RawDeltaResponse{
				Resources:         []types.Resource{scopedRoute},
				DeltaRequest:      &discovery.DeltaDiscoveryRequest{TypeUrl: rsrc.ScopedRouteType},
				SystemVersionInfo: "4",
			},
		},
		rsrc.ListenerType: {
			&cache.RawDeltaResponse{
				Resources:         []types.Resource{httpListener, httpScopedListener},
				DeltaRequest:      &discovery.DeltaDiscoveryRequest{TypeUrl: rsrc.ListenerType},
				SystemVersionInfo: "5",
			},
		},
		rsrc.SecretType: {
			&cache.RawDeltaResponse{
				SystemVersionInfo: "6",
				Resources:         []types.Resource{secret},
				DeltaRequest:      &discovery.DeltaDiscoveryRequest{TypeUrl: rsrc.SecretType},
			},
		},
		rsrc.RuntimeType: {
			&cache.RawDeltaResponse{
				SystemVersionInfo: "7",
				Resources:         []types.Resource{runtime},
				DeltaRequest:      &discovery.DeltaDiscoveryRequest{TypeUrl: rsrc.RuntimeType},
			},
		},
		rsrc.ExtensionConfigType: {
			&cache.RawDeltaResponse{
				SystemVersionInfo: "8",
				Resources:         []types.Resource{extensionConfig},
				DeltaRequest:      &discovery.DeltaDiscoveryRequest{TypeUrl: rsrc.ExtensionConfigType},
			},
		},
		// Pass-through type (types without explicit handling)
		opaqueType: {
			&cache.RawDeltaResponse{
				SystemVersionInfo: "9",
				Resources:         []types.Resource{opaque},
				DeltaRequest:      &discovery.DeltaDiscoveryRequest{TypeUrl: opaqueType},
			},
		},
	}
}

func process(typ string, resp *mockDeltaStream, s server.Server) error {
	var err error
	switch typ {
	case rsrc.EndpointType:
		err = s.DeltaEndpoints(resp)
	case rsrc.ClusterType:
		err = s.DeltaClusters(resp)
	case rsrc.RouteType:
		err = s.DeltaRoutes(resp)
	case rsrc.ScopedRouteType:
		err = s.DeltaScopedRoutes(resp)
	case rsrc.ListenerType:
		err = s.DeltaListeners(resp)
	case rsrc.SecretType:
		err = s.DeltaSecrets(resp)
	case rsrc.RuntimeType:
		err = s.DeltaRuntime(resp)
	case rsrc.ExtensionConfigType:
		err = s.DeltaExtensionConfigs(resp)
	case opaqueType:
		err = s.DeltaAggregatedResources(resp)
	}

	return err
}

func TestDeltaResponseHandlersWildcard(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.deltaResponses = makeDeltaResponses()
			s := server.NewServer(context.Background(), config, server.CallbackFuncs{})

			resp := makeMockDeltaStream(t)
			// This is a wildcard request since we don't specify a list of resource subscriptions
			resp.recv <- &discovery.DeltaDiscoveryRequest{Node: node, TypeUrl: typ}

			go func() {
				err := process(typ, resp, s)
				assert.NoError(t, err)
			}()

			select {
			case res := <-resp.sent:
				close(resp.recv)

				assert.Equal(t, 1, config.deltaCounts[typ])
				assert.Empty(t, res.GetSystemVersionInfo())
			case <-time.After(1 * time.Second):
				t.Fatalf("got no response")
			}
		})
	}
}

func TestDeltaResponseHandlers(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.deltaResponses = makeDeltaResponses()
			s := server.NewServer(context.Background(), config, server.CallbackFuncs{})

			resp := makeMockDeltaStream(t)
			// This is a wildcard request since we don't specify a list of resource subscriptions
			res, err := config.deltaResponses[typ][0].GetDeltaDiscoveryResponse()
			if err != nil {
				t.Error(err)
			}
			// We only subscribe to one resource to see if we get the appropriate number of resources back
			resp.recv <- &discovery.DeltaDiscoveryRequest{Node: node, TypeUrl: typ, ResourceNamesSubscribe: []string{res.Resources[0].Name}}

			go func() {
				err := process(typ, resp, s)
				assert.NoError(t, err)
			}()

			select {
			case res := <-resp.sent:
				close(resp.recv)

				assert.Equal(t, 1, config.deltaCounts[typ])
				assert.Empty(t, res.GetSystemVersionInfo())
			case <-time.After(1 * time.Second):
				t.Fatalf("got no response")
			}
		})
	}
}

func TestSendDeltaError(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.deltaResponses = makeDeltaResponses()
			s := server.NewServer(context.Background(), config, server.CallbackFuncs{})

			// make a request with an error
			resp := makeMockDeltaStream(t)
			resp.sendError = true
			resp.recv <- &discovery.DeltaDiscoveryRequest{
				Node:    node,
				TypeUrl: typ,
			}

			// check that response fails since we expect an error to come through
			err := s.DeltaAggregatedResources(resp)
			assert.Error(t, err)

			close(resp.recv)
		})
	}
}

func TestDeltaAggregatedHandlers(t *testing.T) {
	config := makeMockConfigWatcher()
	config.deltaResponses = makeDeltaResponses()
	resp := makeMockDeltaStream(t)

	reqs := []*discovery.DeltaDiscoveryRequest{
		{
			Node:    node,
			TypeUrl: rsrc.ListenerType,
		},
		{
			Node:    node,
			TypeUrl: rsrc.ClusterType,
		},
		{
			Node:                   node,
			TypeUrl:                rsrc.EndpointType,
			ResourceNamesSubscribe: []string{clusterName},
		},
		{
			TypeUrl:                rsrc.RouteType,
			ResourceNamesSubscribe: []string{routeName},
		},
		{
			TypeUrl:                rsrc.ScopedRouteType,
			ResourceNamesSubscribe: []string{scopedRouteName},
		},
		{
			TypeUrl:                rsrc.SecretType,
			ResourceNamesSubscribe: []string{secretName},
		},
	}

	for _, r := range reqs {
		resp.recv <- r
	}

	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	go func() {
		err := s.DeltaAggregatedResources(resp)
		assert.NoError(t, err)
	}()

	count := 0
	for {
		select {
		case <-resp.sent:
			count++
			if count >= len(reqs) {
				close(resp.recv)
				assert.Equal(
					t,
					map[string]int{
						rsrc.EndpointType:    1,
						rsrc.ClusterType:     1,
						rsrc.RouteType:       1,
						rsrc.ScopedRouteType: 1,
						rsrc.ListenerType:    1,
						rsrc.SecretType:      1},
					config.deltaCounts,
				)
				return
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("got %d messages on the stream, not 5", count)
		}
	}
}

func TestDeltaAggregateRequestType(t *testing.T) {
	config := makeMockConfigWatcher()
	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	resp := makeMockDeltaStream(t)
	resp.recv <- &discovery.DeltaDiscoveryRequest{Node: node}
	if err := s.DeltaAggregatedResources(resp); err == nil {
		t.Error("DeltaAggregatedResources() => got nil, want an error")
	}
}

func TestDeltaCancellations(t *testing.T) {
	config := makeMockConfigWatcher()
	resp := makeMockDeltaStream(t)
	for _, typ := range testTypes {
		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:    node,
			TypeUrl: typ,
		}
	}
	close(resp.recv)
	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	if err := s.DeltaAggregatedResources(resp); err != nil {
		t.Errorf("DeltaAggregatedResources() => got %v, want no error", err)
	}
	if config.watches != 0 {
		t.Errorf("Expect all watches canceled, got %q", config.watches)
	}
}

func TestDeltaOpaqueRequestsChannelMuxing(t *testing.T) {
	config := makeMockConfigWatcher()
	resp := makeMockDeltaStream(t)
	for i := 0; i < 10; i++ {
		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                   node,
			TypeUrl:                fmt.Sprintf("%s%d", opaqueType, i%2),
			ResourceNamesSubscribe: []string{fmt.Sprintf("%d", i)},
		}
	}
	close(resp.recv)
	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	if err := s.DeltaAggregatedResources(resp); err != nil {
		t.Errorf("DeltaAggregatedResources() => got %v, want no error", err)
	}
	if config.watches != 0 {
		t.Errorf("Expect all watches canceled, got %q", config.watches)
	}
}

func TestDeltaCallbackError(t *testing.T) {
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			config := makeMockConfigWatcher()
			config.deltaResponses = makeDeltaResponses()

			s := server.NewServer(context.Background(), config, server.CallbackFuncs{
				DeltaStreamOpenFunc: func(ctx context.Context, i int64, s string) error {
					return errors.New("stream open error")
				},
			})

			// make a request
			resp := makeMockDeltaStream(t)
			resp.recv <- &discovery.DeltaDiscoveryRequest{
				Node:    node,
				TypeUrl: typ,
			}

			// check that response fails since stream open returns error
			if err := s.DeltaAggregatedResources(resp); err == nil {
				t.Error("Stream() => got no error, want error")
			}

			close(resp.recv)
		})
	}
}
