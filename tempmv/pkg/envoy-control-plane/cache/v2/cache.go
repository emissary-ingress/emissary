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

	discovery "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/cache/types"
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
	//
	// Value channel produces requested resources, once they are available.  If
	// the channel is closed prior to cancellation of the watch, an unrecoverable
	// error has occurred in the producer, and the consumer should close the
	// corresponding stream.
	//
	// Cancel is an optional function to release resources in the producer. If
	// provided, the consumer may call this function multiple times.
	CreateWatch(Request) (value chan Response, cancel func())
}

// Cache is a generic config cache with a watcher.
type Cache interface {
	ConfigWatcher

	// Fetch implements the polling method of the config cache using a non-empty request.
	Fetch(context.Context, Request) (Response, error)
}

// Response is a wrapper around Envoy's DiscoveryResponse.
type Response interface {
	// Get the Constructed DiscoveryResponse
	GetDiscoveryResponse() (*discovery.DiscoveryResponse, error)

	// Get te original Request for the Response.
	GetRequest() *discovery.DiscoveryRequest

	// Get the version in the Response.
	GetVersion() (string, error)
}

// RawResponse is a pre-serialized xDS response containing the raw resources to
// be included in the final Discovery Response.
type RawResponse struct {
	Response
	// Request is the original request.
	Request discovery.DiscoveryRequest

	// Version of the resources as tracked by the cache for the given type.
	// Proxy responds with this version as an acknowledgement.
	Version string

	// Resources to be included in the response.
	Resources []types.Resource

	// isResourceMarshaled indicates whether the resources have been marshaled.
	// This is internally maintained by go-control-plane to prevent future
	// duplication in marshaling efforts.
	isResourceMarshaled bool

	// marshaledResponse holds the serialized discovery response.
	marshaledResponse *discovery.DiscoveryResponse
}

// PassthroughResponse is a pre constructed xDS response that need not go through marshalling transformations.
type PassthroughResponse struct {
	Response
	// Request is the original request.
	Request discovery.DiscoveryRequest

	// The discovery response that needs to be sent as is, without any marshalling transformations.
	DiscoveryResponse *discovery.DiscoveryResponse
}

// GetDiscoveryResponse performs the marshalling the first time its called and uses the cached response subsequently.
// This is necessary because the marshalled response does not change across the calls.
// This caching behavior is important in high throughput scenarios because grpc marshalling has a cost and it drives the cpu utilization under load.
func (r RawResponse) GetDiscoveryResponse() (*discovery.DiscoveryResponse, error) {
	if r.isResourceMarshaled {
		return r.marshaledResponse, nil
	}

	marshaledResources := make([]*any.Any, len(r.Resources))

	for i, resource := range r.Resources {
		marshaledResource, err := MarshalResource(resource)
		if err != nil {
			return nil, err
		}
		marshaledResources[i] = &any.Any{
			TypeUrl: r.Request.TypeUrl,
			Value:   marshaledResource,
		}
	}

	r.isResourceMarshaled = true

	return &discovery.DiscoveryResponse{
		VersionInfo: r.Version,
		Resources:   marshaledResources,
		TypeUrl:     r.Request.TypeUrl,
	}, nil
}

// GetRequest returns the original Discovery Request.
func (r RawResponse) GetRequest() *discovery.DiscoveryRequest {
	return &r.Request
}

// GetVersion returns the response version.
func (r RawResponse) GetVersion() (string, error) {
	return r.Version, nil
}

// GetDiscoveryResponse returns the final passthrough Discovery Response.
func (r PassthroughResponse) GetDiscoveryResponse() (*discovery.DiscoveryResponse, error) {
	return r.DiscoveryResponse, nil
}

// GetRequest returns the original Discovery Request
func (r PassthroughResponse) GetRequest() *discovery.DiscoveryRequest {
	return &r.Request
}

// GetVersion returns the response version.
func (r PassthroughResponse) GetVersion() (string, error) {
	if r.DiscoveryResponse != nil {
		return r.DiscoveryResponse.VersionInfo, nil
	}
	return "", fmt.Errorf("DiscoveryResponse is nil")
}
