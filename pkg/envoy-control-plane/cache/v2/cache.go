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

// Package cache defines a configuration cache for the server.
package cache

import (
	"context"
	"fmt"
	"sync/atomic"

	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	ttl "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/ttl/v2"
	"github.com/golang/protobuf/ptypes/any"
)

// Request is an alias for the discovery request type.
type Request = discovery.DiscoveryRequest

// ConfigWatcher requests watches for configuration resources by a node, last
// applied version identifier, and resource names hint. The watch should send
// the responses when they are ready. The watch can be cancelled by the
// consumer, in effect terminating the watch for the request.
// ConfigWatcher implementation must be thread-safe.
type ConfigWatcher interface {
	// CreateWatch returns a new open watch from a non-empty request.
	// An individual consumer normally issues a single open watch by each type URL.
	//
	// Value channel produces requested resources, once they are available.  If
	// the channel is closed prior to cancellation of the watch, an unrecoverable
	// error has occurred in the producer, and the consumer should close the
	// corresponding stream.
	//
	// Cancel is an optional function to release resources in the producer. If
	// provided, the consumer may call this function multiple times.
	CreateWatch(*Request) (value chan Response, cancel func())
}

// ConfigFetcher fetches configuration resources from cache
type ConfigFetcher interface {
	// Fetch implements the polling method of the config cache using a non-empty request.
	Fetch(context.Context, *Request) (Response, error)
}

// Cache is a generic config cache with a watcher.
type Cache interface {
	ConfigWatcher
	ConfigFetcher
}

// Response is a wrapper around Envoy's DiscoveryResponse.
type Response interface {
	// Get the Constructed DiscoveryResponse
	GetDiscoveryResponse() (*discovery.DiscoveryResponse, error)

	// Get the original Request for the Response.
	GetRequest() *discovery.DiscoveryRequest

	// Get the version in the Response.
	GetVersion() (string, error)
}

// RawResponse is a pre-serialized xDS response containing the raw resources to
// be included in the final Discovery Response.
type RawResponse struct {
	// Request is the original request.
	Request *discovery.DiscoveryRequest

	// Version of the resources as tracked by the cache for the given type.
	// Proxy responds with this version as an acknowledgement.
	Version string

	// Resources to be included in the response.
	Resources []types.ResourceWithTtl

	// Whether this is a heartbeat response. For xDS versions that support TTL, this
	// will be converted into a response that doesn't contain the actual resource protobuf.
	// This allows for more lightweight updates that server only to update the TTL timer.
	Heartbeat bool

	// marshaledResponse holds an atomic reference to the serialized discovery response.
	marshaledResponse atomic.Value
}

var _ Response = &RawResponse{}

// PassthroughResponse is a pre constructed xDS response that need not go through marshalling transformations.
type PassthroughResponse struct {
	// Request is the original request.
	Request *discovery.DiscoveryRequest

	// The discovery response that needs to be sent as is, without any marshalling transformations.
	DiscoveryResponse *discovery.DiscoveryResponse
}

var _ Response = &PassthroughResponse{}

// GetDiscoveryResponse performs the marshalling the first time its called and uses the cached response subsequently.
// This is necessary because the marshalled response does not change across the calls.
// This caching behavior is important in high throughput scenarios because grpc marshalling has a cost and it drives the cpu utilization under load.
func (r *RawResponse) GetDiscoveryResponse() (*discovery.DiscoveryResponse, error) {

	marshaledResponse := r.marshaledResponse.Load()

	if marshaledResponse == nil {

		marshaledResources := make([]*any.Any, len(r.Resources))

		for i, resource := range r.Resources {
			maybeTtldResource, resourceType, err := ttl.MaybeCreateTtlResourceIfSupported(resource, GetResourceName(resource.Resource), r.Request.TypeUrl, r.Heartbeat)
			if err != nil {
				return nil, err
			}
			marshaledResource, err := MarshalResource(maybeTtldResource)
			if err != nil {
				return nil, err
			}
			marshaledResources[i] = &any.Any{
				TypeUrl: resourceType,
				Value:   marshaledResource,
			}
		}

		marshaledResponse = &discovery.DiscoveryResponse{
			VersionInfo: r.Version,
			Resources:   marshaledResources,
			TypeUrl:     r.Request.TypeUrl,
		}

		r.marshaledResponse.Store(marshaledResponse)
	}

	return marshaledResponse.(*discovery.DiscoveryResponse), nil
}

// GetRequest returns the original Discovery Request.
func (r *RawResponse) GetRequest() *discovery.DiscoveryRequest {
	return r.Request
}

// GetVersion returns the response version.
func (r *RawResponse) GetVersion() (string, error) {
	return r.Version, nil
}

// GetDiscoveryResponse returns the final passthrough Discovery Response.
func (r *PassthroughResponse) GetDiscoveryResponse() (*discovery.DiscoveryResponse, error) {
	return r.DiscoveryResponse, nil
}

// GetRequest returns the original Discovery Request
func (r *PassthroughResponse) GetRequest() *discovery.DiscoveryRequest {
	return r.Request
}

// GetVersion returns the response version.
func (r *PassthroughResponse) GetVersion() (string, error) {
	if r.DiscoveryResponse != nil {
		return r.DiscoveryResponse.VersionInfo, nil
	}
	return "", fmt.Errorf("DiscoveryResponse is nil")
}
