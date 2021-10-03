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

	discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v2"
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
	OnStreamResponse(int64, *discovery.DiscoveryRequest, *discovery.DiscoveryResponse)
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

	// Opaque resources share a muxed channel. Nonces and watch cancellations are indexed by type URL.
	responses     chan cache.Response
	cancellations map[string]func()
	nonces        map[string]string
	terminations  map[string]chan struct{}
}

// Initialize all watches
func (values *watches) Init() {
	// muxed channel needs a buffer to release go-routines populating it
	values.responses = make(chan cache.Response, 5)
	values.cancellations = make(map[string]func())
	values.nonces = make(map[string]string)
	values.terminations = make(map[string]chan struct{})
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
	if values.listenerCancel != nil {
		values.listenerCancel()
	}
	if values.secretCancel != nil {
		values.secretCancel()
	}
	if values.runtimeCancel != nil {
		values.runtimeCancel()
	}
	for _, cancel := range values.cancellations {
		if cancel != nil {
			cancel()
		}
	}
	for _, terminate := range values.terminations {
		close(terminate)
	}
}

// process handles a bi-di stream request
func (s *server) process(stream Stream, reqCh <-chan *discovery.DiscoveryRequest, defaultTypeURL string) error {
	// increment stream count
	streamID := atomic.AddInt64(&s.streamCount, 1)

	// unique nonce generator for req-resp pairs per xDS stream; the server
	// ignores stale nonces. nonce is only modified within send() function.
	var streamNonce int64

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
	send := func(resp cache.Response, typeURL string) (string, error) {
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

		case resp, more := <-values.responses:
			if more {
				if resp == errorResponse {
					return status.Errorf(codes.Unavailable, "resource watch failed")
				}
				typeUrl := resp.GetRequest().TypeUrl
				nonce, err := send(resp, typeUrl)
				if err != nil {
					return err
				}
				values.nonces[typeUrl] = nonce
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

			// cancel existing watches to (re-)request a newer version
			switch {
			case req.TypeUrl == resource.EndpointType:
				if values.endpointNonce == "" || values.endpointNonce == nonce {
					if values.endpointCancel != nil {
						values.endpointCancel()
					}
					values.endpoints, values.endpointCancel = s.cache.CreateWatch(req)
				}
			case req.TypeUrl == resource.ClusterType:
				if values.clusterNonce == "" || values.clusterNonce == nonce {
					if values.clusterCancel != nil {
						values.clusterCancel()
					}
					values.clusters, values.clusterCancel = s.cache.CreateWatch(req)
				}
			case req.TypeUrl == resource.RouteType:
				if values.routeNonce == "" || values.routeNonce == nonce {
					if values.routeCancel != nil {
						values.routeCancel()
					}
					values.routes, values.routeCancel = s.cache.CreateWatch(req)
				}
			case req.TypeUrl == resource.ListenerType:
				if values.listenerNonce == "" || values.listenerNonce == nonce {
					if values.listenerCancel != nil {
						values.listenerCancel()
					}
					values.listeners, values.listenerCancel = s.cache.CreateWatch(req)
				}
			case req.TypeUrl == resource.SecretType:
				if values.secretNonce == "" || values.secretNonce == nonce {
					if values.secretCancel != nil {
						values.secretCancel()
					}
					values.secrets, values.secretCancel = s.cache.CreateWatch(req)
				}
			case req.TypeUrl == resource.RuntimeType:
				if values.runtimeNonce == "" || values.runtimeNonce == nonce {
					if values.runtimeCancel != nil {
						values.runtimeCancel()
					}
					values.runtimes, values.runtimeCancel = s.cache.CreateWatch(req)
				}
			default:
				typeUrl := req.TypeUrl
				responseNonce, seen := values.nonces[typeUrl]
				if !seen || responseNonce == nonce {
					// We must signal goroutine termination to prevent a race between the cancel closing the watch
					// and the producer closing the watch.
					if terminate, exists := values.terminations[typeUrl]; exists {
						close(terminate)
					}
					if cancel, seen := values.cancellations[typeUrl]; seen && cancel != nil {
						cancel()
					}
					var watch chan cache.Response
					watch, values.cancellations[typeUrl] = s.cache.CreateWatch(req)
					// Muxing watches across multiple type URLs onto a single channel requires spawning
					// a go-routine. Golang does not allow selecting over a dynamic set of channels.
					terminate := make(chan struct{})
					values.terminations[typeUrl] = terminate
					go func() {
						select {
						case resp, more := <-watch:
							if more {
								values.responses <- resp
							} else {
								// Check again if the watch is cancelled.
								select {
								case <-terminate: // do nothing
								default:
									// We cannot close the responses channel since it can be closed twice.
									// Instead we send a fake error response.
									values.responses <- errorResponse
								}
							}
							break
						case <-terminate:
							break
						}
					}()
				}
			}
		}
	}
}

// StreamHandler converts a blocking read call to channels and initiates stream processing
func (s *server) StreamHandler(stream Stream, typeURL string) error {
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
