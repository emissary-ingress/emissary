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

// Package server provides an implementation of a streaming xDS server.
package server

import (
	"context"
	"errors"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/rest/v2"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/sotw/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	clusterservice "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	endpointservice "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	listenerservice "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	routeservice "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	discoverygrpc "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v2"
	runtimeservice "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v2"
	secretservice "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v2"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v2"
)

// Server is a collection of handlers for streaming discovery requests.
type Server interface {
	endpointservice.EndpointDiscoveryServiceServer
	clusterservice.ClusterDiscoveryServiceServer
	routeservice.RouteDiscoveryServiceServer
	listenerservice.ListenerDiscoveryServiceServer
	discoverygrpc.AggregatedDiscoveryServiceServer
	secretservice.SecretDiscoveryServiceServer
	runtimeservice.RuntimeDiscoveryServiceServer

	rest.Server
	sotw.Server
}

// Callbacks is a collection of callbacks inserted into the server operation.
// The callbacks are invoked synchronously.
type Callbacks interface {
	rest.Callbacks
	sotw.Callbacks
}

// CallbackFuncs is a convenience type for implementing the Callbacks interface.
type CallbackFuncs struct {
	StreamOpenFunc     func(context.Context, int64, string) error
	StreamClosedFunc   func(int64)
	StreamRequestFunc  func(int64, *discovery.DiscoveryRequest) error
	StreamResponseFunc func(int64, *discovery.DiscoveryRequest, *discovery.DiscoveryResponse)
	FetchRequestFunc   func(context.Context, *discovery.DiscoveryRequest) error
	FetchResponseFunc  func(*discovery.DiscoveryRequest, *discovery.DiscoveryResponse)
}

var _ Callbacks = CallbackFuncs{}

// OnStreamOpen invokes StreamOpenFunc.
func (c CallbackFuncs) OnStreamOpen(ctx context.Context, streamID int64, typeURL string) error {
	if c.StreamOpenFunc != nil {
		return c.StreamOpenFunc(ctx, streamID, typeURL)
	}

	return nil
}

// OnStreamClosed invokes StreamClosedFunc.
func (c CallbackFuncs) OnStreamClosed(streamID int64) {
	if c.StreamClosedFunc != nil {
		c.StreamClosedFunc(streamID)
	}
}

// OnStreamRequest invokes StreamRequestFunc.
func (c CallbackFuncs) OnStreamRequest(streamID int64, req *discovery.DiscoveryRequest) error {
	if c.StreamRequestFunc != nil {
		return c.StreamRequestFunc(streamID, req)
	}

	return nil
}

// OnStreamResponse invokes StreamResponseFunc.
func (c CallbackFuncs) OnStreamResponse(streamID int64, req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if c.StreamResponseFunc != nil {
		c.StreamResponseFunc(streamID, req, resp)
	}
}

// OnFetchRequest invokes FetchRequestFunc.
func (c CallbackFuncs) OnFetchRequest(ctx context.Context, req *discovery.DiscoveryRequest) error {
	if c.FetchRequestFunc != nil {
		return c.FetchRequestFunc(ctx, req)
	}

	return nil
}

// OnFetchResponse invoked FetchResponseFunc.
func (c CallbackFuncs) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	if c.FetchResponseFunc != nil {
		c.FetchResponseFunc(req, resp)
	}
}

// NewServer creates handlers from a config watcher and callbacks.
func NewServer(ctx context.Context, config cache.Cache, callbacks Callbacks) Server {
	return NewServerAdvanced(rest.NewServer(config, callbacks), sotw.NewServer(ctx, config, callbacks))
}

func NewServerAdvanced(restServer rest.Server, sotwServer sotw.Server) Server {
	return &server{rest: restServer, sotw: sotwServer}
}

type server struct {
	rest rest.Server
	sotw sotw.Server
}

func (s *server) StreamHandler(stream sotw.Stream, typeURL string) error {
	return s.sotw.StreamHandler(stream, typeURL)
}

func (s *server) StreamAggregatedResources(stream discoverygrpc.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	return s.StreamHandler(stream, resource.AnyType)
}

func (s *server) StreamEndpoints(stream endpointservice.EndpointDiscoveryService_StreamEndpointsServer) error {
	return s.StreamHandler(stream, resource.EndpointType)
}

func (s *server) StreamClusters(stream clusterservice.ClusterDiscoveryService_StreamClustersServer) error {
	return s.StreamHandler(stream, resource.ClusterType)
}

func (s *server) StreamRoutes(stream routeservice.RouteDiscoveryService_StreamRoutesServer) error {
	return s.StreamHandler(stream, resource.RouteType)
}

func (s *server) StreamListeners(stream listenerservice.ListenerDiscoveryService_StreamListenersServer) error {
	return s.StreamHandler(stream, resource.ListenerType)
}

func (s *server) StreamSecrets(stream secretservice.SecretDiscoveryService_StreamSecretsServer) error {
	return s.StreamHandler(stream, resource.SecretType)
}

func (s *server) StreamRuntime(stream runtimeservice.RuntimeDiscoveryService_StreamRuntimeServer) error {
	return s.StreamHandler(stream, resource.RuntimeType)
}

// Fetch is the universal fetch method.
func (s *server) Fetch(ctx context.Context, req *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	return s.rest.Fetch(ctx, req)
}

func (s *server) FetchEndpoints(ctx context.Context, req *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.Unavailable, "empty request")
	}
	req.TypeUrl = resource.EndpointType
	return s.Fetch(ctx, req)
}

func (s *server) FetchClusters(ctx context.Context, req *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.Unavailable, "empty request")
	}
	req.TypeUrl = resource.ClusterType
	return s.Fetch(ctx, req)
}

func (s *server) FetchRoutes(ctx context.Context, req *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.Unavailable, "empty request")
	}
	req.TypeUrl = resource.RouteType
	return s.Fetch(ctx, req)
}

func (s *server) FetchListeners(ctx context.Context, req *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.Unavailable, "empty request")
	}
	req.TypeUrl = resource.ListenerType
	return s.Fetch(ctx, req)
}

func (s *server) FetchSecrets(ctx context.Context, req *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.Unavailable, "empty request")
	}
	req.TypeUrl = resource.SecretType
	return s.Fetch(ctx, req)
}

func (s *server) FetchRuntime(ctx context.Context, req *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.Unavailable, "empty request")
	}
	req.TypeUrl = resource.RuntimeType
	return s.Fetch(ctx, req)
}

func (s *server) DeltaAggregatedResources(_ discoverygrpc.AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error {
	return errors.New("not implemented")
}

func (s *server) DeltaEndpoints(_ endpointservice.EndpointDiscoveryService_DeltaEndpointsServer) error {
	return errors.New("not implemented")
}

func (s *server) DeltaClusters(_ clusterservice.ClusterDiscoveryService_DeltaClustersServer) error {
	return errors.New("not implemented")
}

func (s *server) DeltaRoutes(_ routeservice.RouteDiscoveryService_DeltaRoutesServer) error {
	return errors.New("not implemented")
}

func (s *server) DeltaListeners(_ listenerservice.ListenerDiscoveryService_DeltaListenersServer) error {
	return errors.New("not implemented")
}

func (s *server) DeltaSecrets(_ secretservice.SecretDiscoveryService_DeltaSecretsServer) error {
	return errors.New("not implemented")
}

func (s *server) DeltaRuntime(_ runtimeservice.RuntimeDiscoveryService_DeltaRuntimeServer) error {
	return errors.New("not implemented")
}
