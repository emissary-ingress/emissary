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

// Package test contains test utilities
package test

import (
	"google.golang.org/grpc"

	clusterservice "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	endpointservice "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	listenerservice "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	routeservice "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	accessloggrpc "github.com/datawire/ambassador/pkg/api/envoy/service/accesslog/v2"
	discoverygrpc "github.com/datawire/ambassador/pkg/api/envoy/service/discovery/v2"
	runtimeservice "github.com/datawire/ambassador/pkg/api/envoy/service/discovery/v2"
	secretservice "github.com/datawire/ambassador/pkg/api/envoy/service/discovery/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/server/v2"
)

// RegisterAccessLogServer starts an accessloggrpc service.
func RegisterAccessLogServer(grpcServer *grpc.Server, als *AccessLogService) {
	accessloggrpc.RegisterAccessLogServiceServer(grpcServer, als)
}

// RegisterServer registers with v2 services.
func RegisterServer(grpcServer *grpc.Server, server server.Server) {
	// register services
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
}
