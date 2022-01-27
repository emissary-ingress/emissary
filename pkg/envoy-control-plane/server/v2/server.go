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
	"strconv"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	clusterservice "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	endpointservice "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	listenerservice "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	routeservice "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
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

	// Fetch is the universal fetch method.
	Fetch(context.Context, *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error)
}

// Callbacks is a collection of callbacks inserted into the server operation.
// The callbacks are invoked synchronously.
type Callbacks interface {
	// OnStreamOpen is called once an xDS stream is open with a stream ID and the type URL (or "" for ADS).
	// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
	OnStreamOpen(context.Context, int64, string) error
	// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
	OnStreamClosed(int64)
	// OnStreamRequest is called once a request is received on a stream.
	// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
	OnStreamRequest(int64, *discovery.DiscoveryRequest) error
	// OnStreamResponse is called immediately prior to sending a response on a stream.
	OnStreamResponse(int64, *discovery.DiscoveryRequest, *discovery.DiscoveryResponse)
	// OnFetchRequest is called for each Fetch request. Returning an error will end processing of the
	// request and respond with an error.
	OnFetchRequest(context.Context, *discovery.DiscoveryRequest) error
	// OnFetchResponse is called immediately prior to sending a response.
	OnFetchResponse(*discovery.DiscoveryRequest, *discovery.DiscoveryResponse)
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
	return &server{cache: config, callbacks: callbacks, ctx: ctx}
}

type server struct {
	cache     cache.Cache
	callbacks Callbacks

	// streamCount for counting bi-di streams
	streamCount int64
	ctx         context.Context
}

type stream interface {
	grpc.ServerStream

	Send(*discovery.DiscoveryResponse) error
	Recv() (*discovery.DiscoveryRequest, error)
}

// watches for all xDS resource types
type watches struct {
	endpoints chan cache.Response
	clusters  chan cache.Response
	routes    chan cache.Response
	listeners chan cache.Response
	secrets   chan cache.Response
	runtimes  chan cache.Response

	endpointCancel func()
	clusterCancel  func()
	routeCancel    func()
	listenerCancel func()
	secretCancel   func()
	runtimeCancel  func()

	endpointNonce string
	clusterNonce  string
	routeNonce    string
	listenerNonce string
	secretNonce   string
	runtimeNonce  string
}

// Cancel all watches
func (values watches) Cancel() {
	if values.endpointCancel != nil {
		values.endpointCancel()
	}
	if values.clusterCancel != nil {
		values.clusterCancel()
	}
	if values.routeCancel != nil {
		values.routeCancel()
	}
	if values.listenerCancel != nil {
		values.listenerCancel()
	}
	if values.secretCancel != nil {
		values.secretCancel()
	}
	if values.runtimeCancel != nil {
		values.runtimeCancel()
	}
}

func createResponse(resp cache.Response, typeURL string) (*discovery.DiscoveryResponse, error) {
	if resp == nil {
		return nil, errors.New("missing response")
	}

	marshalledResponse, err := resp.GetDiscoveryResponse()
	if err != nil {
		return nil, err
	}

	return marshalledResponse, nil
}

// process handles a bi-di stream request
func (s *server) process(stream stream, reqCh <-chan *discovery.DiscoveryRequest, defaultTypeURL string) error {
	// increment stream count
	streamID := atomic.AddInt64(&s.streamCount, 1)

	// unique nonce generator for req-resp pairs per xDS stream; the server
	// ignores stale nonces. nonce is only modified within send() function.
	var streamNonce int64

	// a collection of watches per request type
	var values watches
	defer func() {
		values.Cancel()
		if s.callbacks != nil {
			s.callbacks.OnStreamClosed(streamID)
		}
	}()

	// sends a response by serializing to protobuf Any
	send := func(resp cache.Response, typeURL string) (string, error) {
		out, err := createResponse(resp, typeURL)
		if err != nil {
			return "", err
		}

		// increment nonce
		streamNonce = streamNonce + 1
		out.Nonce = strconv.FormatInt(streamNonce, 10)
		if s.callbacks != nil {
			s.callbacks.OnStreamResponse(streamID, resp.GetRequest(), out)
		}
		return out.Nonce, stream.Send(out)
	}

	if s.callbacks != nil {
		if err := s.callbacks.OnStreamOpen(stream.Context(), streamID, defaultTypeURL); err != nil {
			return err
		}
	}

	// node may only be set on the first discovery request
	var node = &core.Node{}

	for {
		select {
		case <-s.ctx.Done():
			return nil
		// config watcher can send the requested resources types in any order
		case resp, more := <-values.endpoints:
			if !more {
				return status.Errorf(codes.Unavailable, "endpoints watch failed")
			}
			nonce, err := send(resp, resource.EndpointType)
			if err != nil {
				return err
			}
			values.endpointNonce = nonce

		case resp, more := <-values.clusters:
			if !more {
				return status.Errorf(codes.Unavailable, "clusters watch failed")
			}
			nonce, err := send(resp, resource.ClusterType)
			if err != nil {
				return err
			}
			values.clusterNonce = nonce

		case resp, more := <-values.routes:
			if !more {
				return status.Errorf(codes.Unavailable, "routes watch failed")
			}
			nonce, err := send(resp, resource.RouteType)
			if err != nil {
				return err
			}
			values.routeNonce = nonce

		case resp, more := <-values.listeners:
			if !more {
				return status.Errorf(codes.Unavailable, "listeners watch failed")
			}
			nonce, err := send(resp, resource.ListenerType)
			if err != nil {
				return err
			}
			values.listenerNonce = nonce

		case resp, more := <-values.secrets:
			if !more {
				return status.Errorf(codes.Unavailable, "secrets watch failed")
			}
			nonce, err := send(resp, resource.SecretType)
			if err != nil {
				return err
			}
			values.secretNonce = nonce

		case resp, more := <-values.runtimes:
			if !more {
				return status.Errorf(codes.Unavailable, "runtimes watch failed")
			}
			nonce, err := send(resp, resource.RuntimeType)
			if err != nil {
				return err
			}
			values.runtimeNonce = nonce

		case req, more := <-reqCh:
			// input stream ended or errored out
			if !more {
				return nil
			}
			if req == nil {
				return status.Errorf(codes.Unavailable, "empty request")
			}

			// node field in discovery request is delta-compressed
			if req.Node != nil {
				node = req.Node
			} else {
				req.Node = node
			}

			// nonces can be reused across streams; we verify nonce only if nonce is not initialized
			nonce := req.GetResponseNonce()

			// type URL is required for ADS but is implicit for xDS
			if defaultTypeURL == resource.AnyType {
				if req.TypeUrl == "" {
					return status.Errorf(codes.InvalidArgument, "type URL is required for ADS")
				}
			} else if req.TypeUrl == "" {
				req.TypeUrl = defaultTypeURL
			}

			if s.callbacks != nil {
				if err := s.callbacks.OnStreamRequest(streamID, req); err != nil {
					return err
				}
			}

			// cancel existing watches to (re-)request a newer version
			switch {
			case req.TypeUrl == resource.EndpointType && (values.endpointNonce == "" || values.endpointNonce == nonce):
				if values.endpointCancel != nil {
					values.endpointCancel()
				}
				values.endpoints, values.endpointCancel = s.cache.CreateWatch(*req)
			case req.TypeUrl == resource.ClusterType && (values.clusterNonce == "" || values.clusterNonce == nonce):
				if values.clusterCancel != nil {
					values.clusterCancel()
				}
				values.clusters, values.clusterCancel = s.cache.CreateWatch(*req)
			case req.TypeUrl == resource.RouteType && (values.routeNonce == "" || values.routeNonce == nonce):
				if values.routeCancel != nil {
					values.routeCancel()
				}
				values.routes, values.routeCancel = s.cache.CreateWatch(*req)
			case req.TypeUrl == resource.ListenerType && (values.listenerNonce == "" || values.listenerNonce == nonce):
				if values.listenerCancel != nil {
					values.listenerCancel()
				}
				values.listeners, values.listenerCancel = s.cache.CreateWatch(*req)
			case req.TypeUrl == resource.SecretType && (values.secretNonce == "" || values.secretNonce == nonce):
				if values.secretCancel != nil {
					values.secretCancel()
				}
				values.secrets, values.secretCancel = s.cache.CreateWatch(*req)
			case req.TypeUrl == resource.RuntimeType && (values.runtimeNonce == "" || values.runtimeNonce == nonce):
				if values.runtimeCancel != nil {
					values.runtimeCancel()
				}
				values.runtimes, values.runtimeCancel = s.cache.CreateWatch(*req)
			}
		}
	}
}

// handler converts a blocking read call to channels and initiates stream processing
func (s *server) handler(stream stream, typeURL string) error {
	// a channel for receiving incoming requests
	reqCh := make(chan *discovery.DiscoveryRequest)
	reqStop := int32(0)
	go func() {
		for {
			req, err := stream.Recv()
			if atomic.LoadInt32(&reqStop) != 0 {
				return
			}
			if err != nil {
				close(reqCh)
				return
			}
			reqCh <- req
		}
	}()

	err := s.process(stream, reqCh, typeURL)

	// prevents writing to a closed channel if send failed on blocked recv
	// TODO(kuat) figure out how to unblock recv through gRPC API
	atomic.StoreInt32(&reqStop, 1)

	return err
}

func (s *server) StreamAggregatedResources(stream discoverygrpc.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	return s.handler(stream, resource.AnyType)
}

func (s *server) StreamEndpoints(stream endpointservice.EndpointDiscoveryService_StreamEndpointsServer) error {
	return s.handler(stream, resource.EndpointType)
}

func (s *server) StreamClusters(stream clusterservice.ClusterDiscoveryService_StreamClustersServer) error {
	return s.handler(stream, resource.ClusterType)
}

func (s *server) StreamRoutes(stream routeservice.RouteDiscoveryService_StreamRoutesServer) error {
	return s.handler(stream, resource.RouteType)
}

func (s *server) StreamListeners(stream listenerservice.ListenerDiscoveryService_StreamListenersServer) error {
	return s.handler(stream, resource.ListenerType)
}

func (s *server) StreamSecrets(stream secretservice.SecretDiscoveryService_StreamSecretsServer) error {
	return s.handler(stream, resource.SecretType)
}

func (s *server) StreamRuntime(stream runtimeservice.RuntimeDiscoveryService_StreamRuntimeServer) error {
	return s.handler(stream, resource.RuntimeType)
}

// Fetch is the universal fetch method.
func (s *server) Fetch(ctx context.Context, req *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	if s.callbacks != nil {
		if err := s.callbacks.OnFetchRequest(ctx, req); err != nil {
			return nil, err
		}
	}
	resp, err := s.cache.Fetch(ctx, *req)
	if err != nil {
		return nil, err
	}
	out, err := createResponse(resp, req.TypeUrl)
	if s.callbacks != nil {
		s.callbacks.OnFetchResponse(req, out)
	}
	return out, err
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
