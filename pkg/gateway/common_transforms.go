package gateway

import (
	"fmt"

	v3core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	v3endpoint "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/endpoint/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

// Compile_Endpoints transforms a kubernetes endpoints resource into a v2.ClusterLoadAssignment
func Compile_Endpoints(endpoints *kates.Endpoints) (*CompiledConfig, error) {
	var clas []*CompiledLoadAssignment

	for _, subset := range endpoints.Subsets {
		for _, port := range subset.Ports {
			var lbEndpoints []*v3endpoint.LbEndpoint
			for _, addr := range subset.Addresses {
				lbEndpoints = append(lbEndpoints, makeLbEndpoint("TCP", addr.IP, int(port.Port)))
			}
			path := fmt.Sprintf("k8s/%s/%s/%d", endpoints.Namespace, endpoints.Name, port.Port)
			clas = append(clas, &CompiledLoadAssignment{
				CompiledItem: NewCompiledItem(SourceFromResource(endpoints)),
				LoadAssignment: &v3endpoint.ClusterLoadAssignment{
					ClusterName: path,
					Endpoints:   []*v3endpoint.LocalityLbEndpoints{{LbEndpoints: lbEndpoints}},
				},
			})
			if len(subset.Ports) == 1 {
				path := fmt.Sprintf("k8s/%s/%s", endpoints.Namespace, endpoints.Name)
				clas = append(clas, &CompiledLoadAssignment{
					CompiledItem: NewCompiledItem(SourceFromResource(endpoints)),
					LoadAssignment: &v3endpoint.ClusterLoadAssignment{
						ClusterName: path,
						Endpoints:   []*v3endpoint.LocalityLbEndpoints{{LbEndpoints: lbEndpoints}},
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
func makeLbEndpoint(protocol, ip string, port int) *v3endpoint.LbEndpoint {
	return &v3endpoint.LbEndpoint{
		HostIdentifier: &v3endpoint.LbEndpoint_Endpoint{
			Endpoint: &v3endpoint.Endpoint{
				Address: &v3core.Address{
					Address: &v3core.Address_SocketAddress{
						SocketAddress: &v3core.SocketAddress{
							Protocol:      v3core.SocketAddress_Protocol(v3core.SocketAddress_Protocol_value[protocol]),
							Address:       ip,
							PortSpecifier: &v3core.SocketAddress_PortValue{PortValue: uint32(port)},
							Ipv4Compat:    true,
						},
					},
				},
			},
		},
	}
}
