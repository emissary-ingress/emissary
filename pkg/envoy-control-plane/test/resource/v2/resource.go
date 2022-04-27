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

// Package resource creates test xDS resources
package resource

import (
	"fmt"
	"time"

	pstruct "github.com/golang/protobuf/ptypes/struct"

	"github.com/golang/protobuf/ptypes"

	cluster "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2"
	endpoint "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2"
	listener "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2"
	route "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2"
	auth "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2/auth"
	core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2/core"
	endpointv2 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2/endpoint"
	listenerv2 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2/listener"
	routev2 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2/route"
	als "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/accesslog/v2"
	alf "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/filter/accesslog/v2"
	hcm "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	tcp "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/filter/network/tcp_proxy/v2"
	runtime "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v2"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v2"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v2"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/wellknown"
)

const (
	localhost = "127.0.0.1"

	// XdsCluster is the cluster name for the control server (used by non-ADS set-up)
	XdsCluster = "xds_cluster"

	// Ads mode for resources: one aggregated xDS service
	Ads = "ads"

	// Xds mode for resources: individual xDS services
	Xds = "xds"

	// Rest mode for resources: polling using Fetch
	Rest = "rest"
)

var (
	// RefreshDelay for the polling config source
	RefreshDelay = 500 * time.Millisecond
)

// MakeEndpoint creates a localhost endpoint on a given port.
func MakeEndpoint(clusterName string, port uint32) *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpointv2.LocalityLbEndpoints{{
			LbEndpoints: []*endpointv2.LbEndpoint{{
				HostIdentifier: &endpointv2.LbEndpoint_Endpoint{
					Endpoint: &endpointv2.Endpoint{
						Address: &core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.SocketAddress_TCP,
									Address:  localhost,
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: port,
									},
								},
							},
						},
					},
				},
			}},
		}},
	}
}

// MakeCluster creates a cluster using either ADS or EDS.
func MakeCluster(mode string, clusterName string) *cluster.Cluster {
	edsSource := configSource(mode)

	connectTimeout := 5 * time.Second
	return &cluster.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       ptypes.DurationProto(connectTimeout),
		ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
		EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
			EdsConfig: edsSource,
		},
	}
}

// MakeRoute creates an HTTP route that routes to a given cluster.
func MakeRoute(routeName, clusterName string) *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*routev2.VirtualHost{{
			Name:    routeName,
			Domains: []string{"*"},
			Routes: []*routev2.Route{{
				Match: &routev2.RouteMatch{
					PathSpecifier: &routev2.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &routev2.Route_Route{
					Route: &routev2.RouteAction{
						ClusterSpecifier: &routev2.RouteAction_Cluster{
							Cluster: clusterName,
						},
					},
				},
			}},
		}},
	}
}

// data source configuration
func configSource(mode string) *core.ConfigSource {
	source := &core.ConfigSource{}
	source.ResourceApiVersion = resource.DefaultAPIVersion
	switch mode {
	case Ads:
		source.ConfigSourceSpecifier = &core.ConfigSource_Ads{
			Ads: &core.AggregatedConfigSource{},
		}
	case Xds:
		source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				TransportApiVersion:       resource.DefaultAPIVersion,
				ApiType:                   core.ApiConfigSource_GRPC,
				SetNodeOnFirstMessageOnly: true,
				GrpcServices: []*core.GrpcService{{
					TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
						EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: XdsCluster},
					},
				}},
			},
		}
	case Rest:
		source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType:             core.ApiConfigSource_REST,
				TransportApiVersion: resource.DefaultAPIVersion,
				ClusterNames:        []string{XdsCluster},
				RefreshDelay:        ptypes.DurationProto(RefreshDelay),
			},
		}
	}
	return source
}

// MakeHTTPListener creates a listener using either ADS or RDS for the route.
func MakeHTTPListener(mode string, listenerName string, port uint32, route string) *listener.Listener {
	rdsSource := configSource(mode)

	// access log service configuration
	alsConfig := &als.HttpGrpcAccessLogConfig{
		CommonConfig: &als.CommonGrpcAccessLogConfig{
			LogName: "echo",
			GrpcService: &core.GrpcService{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
						ClusterName: XdsCluster,
					},
				},
			},
		},
	}
	alsConfigPbst, err := ptypes.MarshalAny(alsConfig)
	if err != nil {
		panic(err)
	}

	// HTTP filter configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    rdsSource,
				RouteConfigName: route,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: wellknown.Router,
		}},
		AccessLog: []*alf.AccessLog{{
			Name: wellknown.HTTPGRPCAccessLog,
			ConfigType: &alf.AccessLog_TypedConfig{
				TypedConfig: alsConfigPbst,
			},
		}},
	}
	pbst, err := ptypes.MarshalAny(manager)
	if err != nil {
		panic(err)
	}

	return &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  localhost,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
		FilterChains: []*listenerv2.FilterChain{{
			Filters: []*listenerv2.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listenerv2.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}
}

// MakeTCPListener creates a TCP listener for a cluster.
func MakeTCPListener(listenerName string, port uint32, clusterName string) *listener.Listener {
	// TCP filter configuration
	config := &tcp.TcpProxy{
		StatPrefix: "tcp",
		ClusterSpecifier: &tcp.TcpProxy_Cluster{
			Cluster: clusterName,
		},
	}
	pbst, err := ptypes.MarshalAny(config)
	if err != nil {
		panic(err)
	}
	return &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  localhost,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},
		FilterChains: []*listenerv2.FilterChain{{
			Filters: []*listenerv2.Filter{{
				Name: wellknown.TCPProxy,
				ConfigType: &listenerv2.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}
}

// MakeRuntime creates an RTDS layer with some fields.
func MakeRuntime(runtimeName string) *runtime.Runtime {
	return &runtime.Runtime{
		Name: runtimeName,
		Layer: &pstruct.Struct{
			Fields: map[string]*pstruct.Value{
				"field-0": {
					Kind: &pstruct.Value_NumberValue{NumberValue: 100},
				},
				"field-1": {
					Kind: &pstruct.Value_StringValue{StringValue: "foobar"},
				},
			},
		},
	}
}

// TestSnapshot holds parameters for a synthetic snapshot.
type TestSnapshot struct {
	// Xds indicates snapshot mode: ads, xds, or rest
	Xds string
	// Version for the snapshot.
	Version string
	// UpstreamPort for the single endpoint on the localhost.
	UpstreamPort uint32
	// BasePort is the initial port for the listeners.
	BasePort uint32
	// NumClusters is the total number of clusters to generate.
	NumClusters int
	// NumHTTPListeners is the total number of HTTP listeners to generate.
	NumHTTPListeners int
	// NumTCPListeners is the total number of TCP listeners to generate.
	// Listeners are assigned clusters in a round-robin fashion.
	NumTCPListeners int
	// NumRuntimes is the total number of RTDS layers to generate.
	NumRuntimes int
	// TLS enables SDS-enabled TLS mode on all listeners
	TLS bool
}

// Generate produces a snapshot from the parameters.
func (ts TestSnapshot) Generate() cache.Snapshot {
	clusters := make([]types.Resource, ts.NumClusters)
	endpoints := make([]types.Resource, ts.NumClusters)
	for i := 0; i < ts.NumClusters; i++ {
		name := fmt.Sprintf("cluster-%s-%d", ts.Version, i)
		clusters[i] = MakeCluster(ts.Xds, name)
		endpoints[i] = MakeEndpoint(name, ts.UpstreamPort)
	}

	routes := make([]types.Resource, ts.NumHTTPListeners)
	for i := 0; i < ts.NumHTTPListeners; i++ {
		name := fmt.Sprintf("route-%s-%d", ts.Version, i)
		routes[i] = MakeRoute(name, cache.GetResourceName(clusters[i%ts.NumClusters]))
	}

	total := ts.NumHTTPListeners + ts.NumTCPListeners
	listeners := make([]types.Resource, total)
	for i := 0; i < total; i++ {
		port := ts.BasePort + uint32(i)
		// listener name must be same since ports are shared and previous listener is drained
		name := fmt.Sprintf("listener-%d", port)
		var listener *listener.Listener
		if i < ts.NumHTTPListeners {
			listener = MakeHTTPListener(ts.Xds, name, port, cache.GetResourceName(routes[i]))
		} else {
			listener = MakeTCPListener(name, port, cache.GetResourceName(clusters[i%ts.NumClusters]))
		}

		if ts.TLS {
			for i, chain := range listener.FilterChains {
				tlsc := &auth.DownstreamTlsContext{
					CommonTlsContext: &auth.CommonTlsContext{
						TlsCertificateSdsSecretConfigs: []*auth.SdsSecretConfig{{
							Name:      tlsName,
							SdsConfig: configSource(ts.Xds),
						}},
						ValidationContextType: &auth.CommonTlsContext_ValidationContextSdsSecretConfig{
							ValidationContextSdsSecretConfig: &auth.SdsSecretConfig{
								Name:      rootName,
								SdsConfig: configSource(ts.Xds),
							},
						},
					},
				}
				mt, _ := ptypes.MarshalAny(tlsc)
				chain.TransportSocket = &core.TransportSocket{
					Name: "envoy.transport_sockets.tls",
					ConfigType: &core.TransportSocket_TypedConfig{
						TypedConfig: mt,
					},
				}
				listener.FilterChains[i] = chain
			}
		}

		listeners[i] = listener
	}

	runtimes := make([]types.Resource, ts.NumRuntimes)
	for i := 0; i < ts.NumRuntimes; i++ {
		name := fmt.Sprintf("runtime-%d", i)
		runtimes[i] = MakeRuntime(name)
	}

	var secrets []types.Resource
	if ts.TLS {
		for _, s := range MakeSecrets(tlsName, rootName) {
			secrets = append(secrets, s)
		}
	}

	out := cache.NewSnapshot(
		ts.Version,
		endpoints,
		clusters,
		routes,
		listeners,
		runtimes,
		secrets,
	)

	return out
}
