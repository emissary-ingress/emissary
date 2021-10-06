package gateway

import (
	"fmt"

	v2 "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	v2core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	v2endpoint "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/endpoint"
	"github.com/datawire/ambassador/v2/pkg/kates"
)

// Compile_Endpoints transforms a kubernetes endpoints resource into a v2.ClusterLoadAssignment
func Compile_Endpoints(endpoints *kates.Endpoints) (*CompiledConfig, error) {
	var clas []*CompiledLoadAssignment

	for _, subset := range endpoints.Subsets {
		for _, port := range subset.Ports {
			var lbEndpoints []*v2endpoint.LbEndpoint
			for _, addr := range subset.Addresses {
				lbEndpoints = append(lbEndpoints, makeLbEndpoint("TCP", addr.IP, int(port.Port)))
			}
			path := fmt.Sprintf("k8s/%s/%s/%d", endpoints.Namespace, endpoints.Name, port.Port)
			clas = append(clas, &CompiledLoadAssignment{
				CompiledItem: NewCompiledItem(SourceFromResource(endpoints)),
				LoadAssignment: &v2.ClusterLoadAssignment{
					ClusterName: path,
					Endpoints:   []*v2endpoint.LocalityLbEndpoints{{LbEndpoints: lbEndpoints}},
				},
			})
			if len(subset.Ports) == 1 {
				path := fmt.Sprintf("k8s/%s/%s", endpoints.Namespace, endpoints.Name)
				clas = append(clas, &CompiledLoadAssignment{
					CompiledItem: NewCompiledItem(SourceFromResource(endpoints)),
					LoadAssignment: &v2.ClusterLoadAssignment{
						ClusterName: path,
						Endpoints:   []*v2endpoint.LocalityLbEndpoints{{LbEndpoints: lbEndpoints}},
					},
				})
			}
		}
	}

	return &CompiledConfig{
		CompiledItem:    NewCompiledItem(SourceFromResource(endpoints)),
		LoadAssignments: clas,
	}, nil
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
