package server_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/goleak"

	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	rsrc "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/stream/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/test/resource/v3"
)

func (config *mockConfigWatcher) CreateDeltaWatch(req *discovery.DeltaDiscoveryRequest, state stream.StreamState, out chan cache.DeltaResponse) func() {
	config.deltaCounts[req.GetTypeUrl()] = config.deltaCounts[req.GetTypeUrl()] + 1

	// This is duplicated from pkg/cache/v3/delta.go as private there
	resourceMap := config.deltaResources[req.GetTypeUrl()]
	versionMap := map[string]string{}
	for name, resource := range resourceMap {
		marshaledResource, _ := cache.MarshalResource(resource)
		versionMap[name] = cache.HashResource(marshaledResource)
	}
	var nextVersionMap map[string]string
	var filtered []types.Resource
	var toRemove []string

	// If we are handling a wildcard request, we want to respond with all resources
	switch {
	case state.IsWildcard():
		if len(state.GetResourceVersions()) == 0 {
			filtered = make([]types.Resource, 0, len(resourceMap))
		}
		nextVersionMap = make(map[string]string, len(resourceMap))
		for name, r := range resourceMap {
			// Since we've already precomputed the version hashes of the new snapshot,
			// we can just set it here to be used for comparison later
			version := versionMap[name]
			nextVersionMap[name] = version
			prevVersion, found := state.GetResourceVersions()[name]
			if !found || (prevVersion != version) {
				filtered = append(filtered, r)
			}
		}

		// Compute resources for removal
		for name := range state.GetResourceVersions() {
			if _, ok := resourceMap[name]; !ok {
				toRemove = append(toRemove, name)
			}
		}
	default:
		nextVersionMap = make(map[string]string, len(state.GetSubscribedResourceNames()))
		// state.GetResourceVersions() may include resources no longer subscribed
		// In the current code this gets silently cleaned when updating the version map
		for name := range state.GetSubscribedResourceNames() {
			prevVersion, found := state.GetResourceVersions()[name]
			if r, ok := resourceMap[name]; ok {
				nextVersion := versionMap[name]
				if prevVersion != nextVersion {
					filtered = append(filtered, r)
				}
				nextVersionMap[name] = nextVersion
			} else if found {
				toRemove = append(toRemove, name)
			}
		}
	}

	if len(filtered)+len(toRemove) > 0 {
		out <- &cache.RawDeltaResponse{
			DeltaRequest:      req,
			Resources:         filtered,
			RemovedResources:  toRemove,
			SystemVersionInfo: "",
			NextVersionMap:    nextVersionMap,
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
	cancel func()
}

func (stream *mockDeltaStream) Context() context.Context {
	return stream.ctx
}

func (stream *mockDeltaStream) Send(resp *discovery.DeltaDiscoveryResponse) error {
	// Check that nonce is incremented by one
	stream.nonce++
	if resp.GetNonce() != fmt.Sprintf("%d", stream.nonce) {
		stream.t.Errorf("Nonce => got %q, want %d", resp.GetNonce(), stream.nonce)
	}
	// Check that resources are non-empty
	if len(resp.GetResources()) == 0 {
		stream.t.Error("Resources => got none, want non-empty")
	}
	if resp.GetTypeUrl() == "" {
		stream.t.Error("TypeUrl => got none, want non-empty")
	}

	// Check that the per resource TypeURL is correctly set.
	for _, res := range resp.GetResources() {
		if res.GetResource().GetTypeUrl() != resp.GetTypeUrl() {
			stream.t.Errorf("TypeUrl => got %q, want %q", res.GetResource().GetTypeUrl(), resp.GetTypeUrl())
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
	ctx, cancel := context.WithCancel(context.Background())
	return &mockDeltaStream{
		t:      t,
		ctx:    ctx,
		sent:   make(chan *discovery.DeltaDiscoveryResponse, 10),
		recv:   make(chan *discovery.DeltaDiscoveryRequest, 10),
		cancel: cancel,
	}
}

func makeDeltaResources() map[string]map[string]types.Resource {
	return map[string]map[string]types.Resource{
		rsrc.EndpointType: {
			endpoint.GetClusterName(): endpoint,
		},
		rsrc.ClusterType: {
			cluster.GetName(): cluster,
		},
		rsrc.RouteType: {
			route.GetName(): route,
		},
		rsrc.ScopedRouteType: {
			scopedRoute.GetName(): scopedRoute,
		},
		rsrc.VirtualHostType: {
			virtualHost.GetName(): virtualHost,
		},
		rsrc.ListenerType: {
			httpListener.GetName():       httpListener,
			httpScopedListener.GetName(): httpScopedListener,
		},
		rsrc.SecretType: {
			secret.GetName(): secret,
		},
		rsrc.RuntimeType: {
			runtime.GetName(): runtime,
		},
		rsrc.ExtensionConfigType: {
			extensionConfig.GetName(): extensionConfig,
		},
		// Pass-through type (types without explicit handling)
		opaqueType: {
			"opaque": opaque,
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
	case rsrc.VirtualHostType:
		err = s.DeltaVirtualHosts(resp)
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
			config.deltaResources = makeDeltaResources()
			s := server.NewServer(context.Background(), config, server.CallbackFuncs{})

			resp := makeMockDeltaStream(t)
			defer resp.cancel()
			// This is a wildcard request since we don't specify a list of resource subscriptions
			resp.recv <- &discovery.DeltaDiscoveryRequest{Node: node, TypeUrl: typ}

			go func() {
				err := process(typ, resp, s)
				require.NoError(t, err)
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
			config.deltaResources = makeDeltaResources()
			s := server.NewServer(context.Background(), config, server.CallbackFuncs{})

			resp := makeMockDeltaStream(t)
			defer resp.cancel()
			resourceNames := []string{}
			for resourceName := range config.deltaResources[typ] {
				resourceNames = append(resourceNames, resourceName)
			}
			// We only subscribe to one resource to see if we get the appropriate number of resources back
			resp.recv <- &discovery.DeltaDiscoveryRequest{Node: node, TypeUrl: typ, ResourceNamesSubscribe: resourceNames}

			go func() {
				err := process(typ, resp, s)
				require.NoError(t, err)
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
			config.deltaResources = makeDeltaResources()
			s := server.NewServer(context.Background(), config, server.CallbackFuncs{})

			// make a request with an error
			resp := makeMockDeltaStream(t)
			defer resp.cancel()
			resp.sendError = true
			resp.recv <- &discovery.DeltaDiscoveryRequest{
				Node:    node,
				TypeUrl: typ,
			}

			// check that response fails since we expect an error to come through
			err := s.DeltaAggregatedResources(resp)
			require.Error(t, err)

			close(resp.recv)
		})
	}
}

func TestDeltaAggregatedHandlers(t *testing.T) {
	config := makeMockConfigWatcher()
	config.deltaResources = makeDeltaResources()
	resp := makeMockDeltaStream(t)
	defer resp.cancel()

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
			TypeUrl:                rsrc.VirtualHostType,
			ResourceNamesSubscribe: []string{virtualHostName},
		},
		{
			TypeUrl:                rsrc.SecretType,
			ResourceNamesSubscribe: []string{secretName},
		},
	}

	for _, r := range reqs {
		resp.recv <- r
	}

	// We create the server with the optional ordered ADS flag so we guarantee resource
	// ordering over the stream.
	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	go func() {
		err := s.DeltaAggregatedResources(resp)
		require.NoError(t, err)
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
						rsrc.VirtualHostType: 1,
						rsrc.ListenerType:    1,
						rsrc.SecretType:      1,
					},
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
	defer goleak.VerifyNone(t)
	config := makeMockConfigWatcher()
	s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
	resp := makeMockDeltaStream(t)
	defer resp.cancel()
	resp.recv <- &discovery.DeltaDiscoveryRequest{Node: node}
	if err := s.DeltaAggregatedResources(resp); err == nil {
		t.Error("DeltaAggregatedResources() => got nil, want an error")
	}
	close(resp.recv)
}

func TestDeltaCancellations(t *testing.T) {
	defer goleak.VerifyNone(t)
	config := makeMockConfigWatcher()
	resp := makeMockDeltaStream(t)
	defer resp.cancel()
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
	defer goleak.VerifyNone(t)
	config := makeMockConfigWatcher()
	resp := makeMockDeltaStream(t)
	defer resp.cancel()
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
			defer goleak.VerifyNone(t)
			config := makeMockConfigWatcher()
			config.deltaResources = makeDeltaResources()

			s := server.NewServer(context.Background(), config, server.CallbackFuncs{
				DeltaStreamOpenFunc: func(context.Context, int64, string) error {
					return errors.New("stream open error")
				},
			})

			// make a request
			resp := makeMockDeltaStream(t)
			defer resp.cancel()
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

func TestDeltaWildcardSubscriptions(t *testing.T) {
	config := makeMockConfigWatcher()
	config.deltaResources = map[string]map[string]types.Resource{
		rsrc.EndpointType: {
			"endpoints0": resource.MakeEndpoint("endpoints0", 1234),
			"endpoints1": resource.MakeEndpoint("endpoints1", 1234),
			"endpoints2": resource.MakeEndpoint("endpoints2", 1234),
			"endpoints3": resource.MakeEndpoint("endpoints3", 1234),
		},
	}

	validateResponse := func(t *testing.T, replies <-chan *discovery.DeltaDiscoveryResponse, expectedResources, expectedRemovedResources []string) {
		t.Helper()
		select {
		case response := <-replies:
			assert.Equal(t, rsrc.EndpointType, response.GetTypeUrl())
			if assert.Equal(t, len(expectedResources), len(response.GetResources())) {
				var names []string
				for _, resource := range response.GetResources() {
					names = append(names, resource.GetName())
				}
				assert.ElementsMatch(t, names, expectedResources)
				assert.ElementsMatch(t, response.GetRemovedResources(), expectedRemovedResources)
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("got no response")
		}
	}

	updateResources := func(port uint32) {
		config.deltaResources[rsrc.EndpointType]["endpoints0"] = resource.MakeEndpoint("endpoints0", port)
		config.deltaResources[rsrc.EndpointType]["endpoints1"] = resource.MakeEndpoint("endpoints1", port)
		config.deltaResources[rsrc.EndpointType]["endpoints2"] = resource.MakeEndpoint("endpoints2", port)
		config.deltaResources[rsrc.EndpointType]["endpoints3"] = resource.MakeEndpoint("endpoints3", port)
	}

	t.Run("legacy still working", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		resp := makeMockDeltaStream(t)
		defer resp.cancel()
		defer close(resp.recv)
		s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
		go func() {
			err := s.DeltaAggregatedResources(resp)
			require.NoError(t, err)
		}()

		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:    node,
			TypeUrl: rsrc.EndpointType,
		}
		validateResponse(t, resp.sent, []string{"endpoints0", "endpoints1", "endpoints2", "endpoints3"}, nil)

		// Generate a change to ensure we receive updates if subscribed
		updateResources(2345)

		// In legacy mode, adding a new resource behaves the same as if providing a subscription to wildcard first
		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                   node,
			TypeUrl:                rsrc.EndpointType,
			ResourceNamesSubscribe: []string{"endpoints0"},
		}
		validateResponse(t, resp.sent, []string{"endpoints0", "endpoints1", "endpoints2", "endpoints3"}, nil)

		updateResources(1234)

		// We allow unsubscribing with the new method
		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                     node,
			TypeUrl:                  rsrc.EndpointType,
			ResourceNamesUnsubscribe: []string{"*"},
		}
		validateResponse(t, resp.sent, []string{"endpoints0"}, nil)
	})

	t.Run("* subscription/unsubscription support", func(t *testing.T) {
		resp := makeMockDeltaStream(t)
		defer resp.cancel()
		defer close(resp.recv)
		s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
		go func() {
			err := s.DeltaAggregatedResources(resp)
			require.NoError(t, err)
		}()
		updateResources(1234)

		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                   node,
			TypeUrl:                rsrc.EndpointType,
			ResourceNamesSubscribe: []string{"endpoints1"},
		}
		validateResponse(t, resp.sent, []string{"endpoints1"}, nil)

		updateResources(2345)

		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                   node,
			TypeUrl:                rsrc.EndpointType,
			ResourceNamesSubscribe: []string{"*"},
		}
		validateResponse(t, resp.sent, []string{"endpoints0", "endpoints1", "endpoints2", "endpoints3"}, nil)

		updateResources(1234)

		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                   node,
			TypeUrl:                rsrc.EndpointType,
			ResourceNamesSubscribe: []string{"endpoints2"},
		}
		validateResponse(t, resp.sent, []string{"endpoints0", "endpoints1", "endpoints2", "endpoints3"}, nil)

		updateResources(2345)

		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                     node,
			TypeUrl:                  rsrc.EndpointType,
			ResourceNamesUnsubscribe: []string{"*"},
		}
		validateResponse(t, resp.sent, []string{"endpoints1", "endpoints2"}, nil)
	})

	t.Run("resource specific subscriptions while using wildcard", func(t *testing.T) {
		resp := makeMockDeltaStream(t)
		defer resp.cancel()
		defer close(resp.recv)
		s := server.NewServer(context.Background(), config, server.CallbackFuncs{})
		go func() {
			err := s.DeltaAggregatedResources(resp)
			require.NoError(t, err)
		}()

		updateResources(1234)

		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                   node,
			TypeUrl:                rsrc.EndpointType,
			ResourceNamesSubscribe: []string{"*"},
		}
		validateResponse(t, resp.sent, []string{"endpoints0", "endpoints1", "endpoints2", "endpoints3"}, nil)

		updateResources(2345)

		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                   node,
			TypeUrl:                rsrc.EndpointType,
			ResourceNamesSubscribe: []string{"endpoints2", "endpoints4"}, // endpoints4 does not exist
		}
		validateResponse(t, resp.sent, []string{"endpoints0", "endpoints1", "endpoints2", "endpoints3"}, nil)

		// Don't update the resources now, test unsubscribing does send the resource again

		resp.recv <- &discovery.DeltaDiscoveryRequest{
			Node:                     node,
			TypeUrl:                  rsrc.EndpointType,
			ResourceNamesUnsubscribe: []string{"endpoints2", "endpoints4"}, // endpoints4 does not exist
		}
		validateResponse(t, resp.sent, []string{"endpoints2"}, []string{"endpoints4"})
	})
}

func TestDeltaMultipleStreams(t *testing.T) {
	// Unit test for issue identified in https://github.com/envoyproxy/go-control-plane/issues/913
	t.Run("return error to delta stream request; multiple streams.", func(t *testing.T) {
		defer goleak.VerifyNone(t)
		config := makeMockConfigWatcher()
		resp := makeMockDeltaStream(t)
		defer close(resp.recv)
		defer resp.cancel()
		s := server.NewServer(
			context.Background(),
			config,
			server.CallbackFuncs{
				StreamDeltaRequestFunc: func(int64, *discovery.DeltaDiscoveryRequest) error {
					return fmt.Errorf("error")
				},
			},
		)

		for i := 0; i < 2; i++ {
			resp.recv <- &discovery.DeltaDiscoveryRequest{
				Node:                   node,
				TypeUrl:                rsrc.EndpointType,
				ResourceNamesSubscribe: []string{"*"},
			}
		}

		err := s.DeltaAggregatedResources(resp)
		require.Error(t, err)
	})
}
