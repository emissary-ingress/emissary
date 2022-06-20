// Copyright 2020 Envoyproxy Authors
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

// Package sotw provides an implementation of GRPC SoTW (State of The World) part of XDS server
package sotw

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	streamv3 "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/stream/v3"
)

type Server interface {
	StreamHandler(stream Stream, typeURL string) error
}

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
	OnStreamResponse(context.Context, int64, *discovery.DiscoveryRequest, *discovery.DiscoveryResponse)
}

// NewServer creates handlers from a config watcher and callbacks.
func NewServer(ctx context.Context, config cache.ConfigWatcher, callbacks Callbacks) Server {
	return &server{cache: config, callbacks: callbacks, ctx: ctx}
}

type server struct {
	cache     cache.ConfigWatcher
	callbacks Callbacks
	ctx       context.Context

	// streamCount for counting bi-di streams
	streamCount int64
}

// Generic RPC stream.
type Stream interface {
	grpc.ServerStream

	Send(*discovery.DiscoveryResponse) error
	Recv() (*discovery.DiscoveryRequest, error)
}

// watches for all xDS resource types
type watches struct {
	endpoints        chan cache.Response
	clusters         chan cache.Response
	routes           chan cache.Response
	scopedRoutes     chan cache.Response
	listeners        chan cache.Response
	secrets          chan cache.Response
	runtimes         chan cache.Response
	extensionConfigs chan cache.Response

	endpointCancel        func()
	clusterCancel         func()
	routeCancel           func()
	scopedRouteCancel     func()
	listenerCancel        func()
	secretCancel          func()
	runtimeCancel         func()
	extensionConfigCancel func()

	endpointNonce        string
	clusterNonce         string
	routeNonce           string
	scopedRouteNonce     string
	listenerNonce        string
	secretNonce          string
	runtimeNonce         string
	extensionConfigNonce string

	// Opaque resources share a muxed channel. Nonces and watch cancellations are indexed by type URL.
	responses     chan cache.Response
	cancellations map[string]func()
	nonces        map[string]string
}

// Discovery response that is sent over GRPC stream
// We need to record what resource names are already sent to a client
// So if the client requests a new name we can respond back
// regardless current snapshot version (even if it is not changed yet)
type lastDiscoveryResponse struct {
	nonce     string
	resources map[string]struct{}
}

// Initialize all watches
func (values *watches) Init() {
	// muxed channel needs a buffer to release go-routines populating it
	values.responses = make(chan cache.Response, 5)
	values.cancellations = make(map[string]func())
	values.nonces = make(map[string]string)
}

// Token response value used to signal a watch failure in muxed watches.
var errorResponse = &cache.RawResponse{}

// Cancel all watches
func (values *watches) Cancel() {
	if values.endpointCancel != nil {
		values.endpointCancel()
	}
	if values.clusterCancel != nil {
		values.clusterCancel()
	}
	if values.routeCancel != nil {
		values.routeCancel()
	}
	if values.scopedRouteCancel != nil {
		values.scopedRouteCancel()
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
	if values.extensionConfigCancel != nil {
		values.extensionConfigCancel()
	}
	for _, cancel := range values.cancellations {
		if cancel != nil {
			cancel()
		}
	}
}

// process handles a bi-di stream request
func (s *server) process(stream Stream, reqCh <-chan *discovery.DiscoveryRequest, defaultTypeURL string) error {
	// increment stream count
	streamID := atomic.AddInt64(&s.streamCount, 1)

	// unique nonce generator for req-resp pairs per xDS stream; the server
	// ignores stale nonces. nonce is only modified within send() function.
	var streamNonce int64

	streamState := streamv3.NewStreamState(false, map[string]string{})
	lastDiscoveryResponses := map[string]lastDiscoveryResponse{}

	// a collection of stack allocated watches per request type
	var values watches
	values.Init()
	defer func() {
		values.Cancel()
		if s.callbacks != nil {
			s.callbacks.OnStreamClosed(streamID)
		}
	}()

	// sends a response by serializing to protobuf Any
	send := func(resp cache.Response) (string, error) {
		if resp == nil {
			return "", errors.New("missing response")
		}

		out, err := resp.GetDiscoveryResponse()
		if err != nil {
			return "", err
		}

		// increment nonce
		streamNonce = streamNonce + 1
		out.Nonce = strconv.FormatInt(streamNonce, 10)

		lastResponse := lastDiscoveryResponse{
			nonce:     out.Nonce,
			resources: make(map[string]struct{}),
		}
		for _, r := range resp.GetRequest().ResourceNames {
			lastResponse.resources[r] = struct{}{}
		}
		lastDiscoveryResponses[resp.GetRequest().TypeUrl] = lastResponse

		if s.callbacks != nil {
			s.callbacks.OnStreamResponse(resp.GetContext(), streamID, resp.GetRequest(), out)
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
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.endpointNonce = nonce

		case resp, more := <-values.clusters:
			if !more {
				return status.Errorf(codes.Unavailable, "clusters watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.clusterNonce = nonce

		case resp, more := <-values.routes:
			if !more {
				return status.Errorf(codes.Unavailable, "routes watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.routeNonce = nonce

		case resp, more := <-values.scopedRoutes:
			if !more {
				return status.Errorf(codes.Unavailable, "scopedRoutes watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.scopedRouteNonce = nonce

		case resp, more := <-values.listeners:
			if !more {
				return status.Errorf(codes.Unavailable, "listeners watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.listenerNonce = nonce

		case resp, more := <-values.secrets:
			if !more {
				return status.Errorf(codes.Unavailable, "secrets watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.secretNonce = nonce

		case resp, more := <-values.runtimes:
			if !more {
				return status.Errorf(codes.Unavailable, "runtimes watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.runtimeNonce = nonce

		case resp, more := <-values.extensionConfigs:
			if !more {
				return status.Errorf(codes.Unavailable, "extensionConfigs watch failed")
			}
			nonce, err := send(resp)
			if err != nil {
				return err
			}
			values.extensionConfigNonce = nonce

		case resp, more := <-values.responses:
			if more {
				if resp == errorResponse {
					return status.Errorf(codes.Unavailable, "resource watch failed")
				}
				typeURL := resp.GetRequest().TypeUrl
				nonce, err := send(resp)
				if err != nil {
					return err
				}
				values.nonces[typeURL] = nonce
			}

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

			if lastResponse, ok := lastDiscoveryResponses[req.TypeUrl]; ok {
				if lastResponse.nonce == "" || lastResponse.nonce == nonce {
					// Let's record Resource names that a client has received.
					streamState.SetKnownResourceNames(req.TypeUrl, lastResponse.resources)
				}
			}

			// cancel existing watches to (re-)request a newer version
			switch {
			case req.TypeUrl == resource.EndpointType:
				if values.endpointNonce == "" || values.endpointNonce == nonce {
					if values.endpointCancel != nil {
						values.endpointCancel()
					}
					values.endpoints = make(chan cache.Response, 1)
					values.endpointCancel = s.cache.CreateWatch(req, streamState, values.endpoints)
				}
			case req.TypeUrl == resource.ClusterType:
				if values.clusterNonce == "" || values.clusterNonce == nonce {
					if values.clusterCancel != nil {
						values.clusterCancel()
					}
					values.clusters = make(chan cache.Response, 1)
					values.clusterCancel = s.cache.CreateWatch(req, streamState, values.clusters)
				}
			case req.TypeUrl == resource.RouteType:
				if values.routeNonce == "" || values.routeNonce == nonce {
					if values.routeCancel != nil {
						values.routeCancel()
					}
					values.routes = make(chan cache.Response, 1)
					values.routeCancel = s.cache.CreateWatch(req, streamState, values.routes)
				}
			case req.TypeUrl == resource.ScopedRouteType:
				if values.scopedRouteNonce == "" || values.scopedRouteNonce == nonce {
					if values.scopedRouteCancel != nil {
						values.scopedRouteCancel()
					}
					values.scopedRoutes = make(chan cache.Response, 1)
					values.scopedRouteCancel = s.cache.CreateWatch(req, streamState, values.scopedRoutes)
				}
			case req.TypeUrl == resource.ListenerType:
				if values.listenerNonce == "" || values.listenerNonce == nonce {
					if values.listenerCancel != nil {
						values.listenerCancel()
					}
					values.listeners = make(chan cache.Response, 1)
					values.listenerCancel = s.cache.CreateWatch(req, streamState, values.listeners)
				}
			case req.TypeUrl == resource.SecretType:
				if values.secretNonce == "" || values.secretNonce == nonce {
					if values.secretCancel != nil {
						values.secretCancel()
					}
					values.secrets = make(chan cache.Response, 1)
					values.secretCancel = s.cache.CreateWatch(req, streamState, values.secrets)
				}
			case req.TypeUrl == resource.RuntimeType:
				if values.runtimeNonce == "" || values.runtimeNonce == nonce {
					if values.runtimeCancel != nil {
						values.runtimeCancel()
					}
					values.runtimes = make(chan cache.Response, 1)
					values.runtimeCancel = s.cache.CreateWatch(req, streamState, values.runtimes)
				}
			case req.TypeUrl == resource.ExtensionConfigType:
				if values.extensionConfigNonce == "" || values.extensionConfigNonce == nonce {
					if values.extensionConfigCancel != nil {
						values.extensionConfigCancel()
					}
					values.extensionConfigs = make(chan cache.Response, 1)
					values.extensionConfigCancel = s.cache.CreateWatch(req, streamState, values.extensionConfigs)
				}
			default:
				typeURL := req.TypeUrl
				responseNonce, seen := values.nonces[typeURL]
				if !seen || responseNonce == nonce {
					if cancel, seen := values.cancellations[typeURL]; seen && cancel != nil {
						cancel()
					}
					values.cancellations[typeURL] = s.cache.CreateWatch(req, streamState, values.responses)
				}
			}
		}
	}
}

// StreamHandler converts a blocking read call to channels and initiates stream processing
func (s *server) StreamHandler(stream Stream, typeURL string) error {
	// a channel for receiving incoming requests
	reqCh := make(chan *discovery.DiscoveryRequest)
	go func() {
		defer close(reqCh)
		for {
			req, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case reqCh <- req:
			case <-stream.Context().Done():
				return
			case <-s.ctx.Done():
				return
			}
		}
	}()

	return s.process(stream, reqCh, typeURL)
}
