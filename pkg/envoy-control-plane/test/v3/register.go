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

	accessloggrpc "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/accesslog/v3"
	clusterservice "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/cluster/v3"
	discoverygrpc "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	endpointservice "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/endpoint/v3"
	listenerservice "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/listener/v3"
	routeservice "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/route/v3"
	runtimeservice "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/runtime/v3"
	secretservice "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/secret/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v3"
)

// RegisterAccessLogServer starts an accessloggrpc service.
func RegisterAccessLogServer(grpcServer *grpc.Server, als *AccessLogService) {
	accessloggrpc.RegisterAccessLogServiceServer(grpcServer, als)
}

// RegisterServer registers with v3 services.
func RegisterServer(grpcServer *grpc.Server, server server.Server) {
	// register services
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterScopedRoutesDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterVirtualHostDiscoveryServiceServer(grpcServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
}
