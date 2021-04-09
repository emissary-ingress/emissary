package gateway

import (
	v2 "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	v2core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	v2endpoint "github.com/datawire/ambassador/pkg/api/envoy/api/v2/endpoint"
	"github.com/datawire/ambassador/pkg/kates"
)

// Compile_Endpoints transforms a kubernetes endpoints resource into a v2.ClusterLoadAssignment
func Compile_Endpoints(endpoints *kates.Endpoints) *CompiledConfig {
	var lbEndpoints []*v2endpoint.LbEndpoint
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			for _, port := range subset.Ports {
				lbEndpoints = append(lbEndpoints, makeLbEndpoint("TCP", addr.IP, int(port.Port)))
			}
		}
	}

	return &CompiledConfig{
		CompiledItem: NewCompiledItem(SourceFromResource(endpoints)),
		LoadAssignments: []*CompiledLoadAssignment{
			{
				CompiledItem: NewCompiledItem(SourceFromResource(endpoints)),
				LoadAssignment: &v2.ClusterLoadAssignment{
					ClusterName: endpoints.Name,
					Endpoints:   []*v2endpoint.LocalityLbEndpoints{{LbEndpoints: lbEndpoints}},
				},
			},
		},
	}
}

// makeLbEndpoint takes a protocol, ip, and port and makes an envoy LbEndpoint.
func makeLbEndpoint(protocol, ip string, port int) *v2endpoint.LbEndpoint {
	return &v2endpoint.LbEndpoint{
		HostIdentifier: &v2endpoint.LbEndpoint_Endpoint{
			Endpoint: &v2endpoint.Endpoint{
				Address: &v2core.Address{
					Address: &v2core.Address_SocketAddress{
						SocketAddress: &v2core.SocketAddress{
							Protocol:      v2core.SocketAddress_Protocol(v2core.SocketAddress_Protocol_value[protocol]),
							Address:       ip,
							PortSpecifier: &v2core.SocketAddress_PortValue{PortValue: uint32(port)},
							Ipv4Compat:    true,
						},
					},
				},
			},
		},
	}
}
