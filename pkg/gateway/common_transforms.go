package gateway

import (
	"fmt"

	apiv3_core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	apiv3_endpoint "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/endpoint/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

// Compile_Endpoints transforms a kubernetes endpoints resource into a apiv3_endpoint.ClusterLoadAssignment
func Compile_Endpoints(endpoints *kates.Endpoints) (*CompiledConfig, error) {
	var clas []*CompiledLoadAssignment

	for _, subset := range endpoints.Subsets {
		for _, port := range subset.Ports {
			var lbEndpoints []*apiv3_endpoint.LbEndpoint
			for _, addr := range subset.Addresses {
				lbEndpoints = append(lbEndpoints, makeLbEndpoint("TCP", addr.IP, int(port.Port)))
			}
			path := fmt.Sprintf("k8s/%s/%s/%d", endpoints.Namespace, endpoints.Name, port.Port)
			clas = append(clas, &CompiledLoadAssignment{
				CompiledItem: NewCompiledItem(SourceFromResource(endpoints)),
				LoadAssignment: &apiv3_endpoint.ClusterLoadAssignment{
					ClusterName: path,
					Endpoints:   []*apiv3_endpoint.LocalityLbEndpoints{{LbEndpoints: lbEndpoints}},
				},
			})
			if len(subset.Ports) == 1 {
				path := fmt.Sprintf("k8s/%s/%s", endpoints.Namespace, endpoints.Name)
				clas = append(clas, &CompiledLoadAssignment{
					CompiledItem: NewCompiledItem(SourceFromResource(endpoints)),
					LoadAssignment: &apiv3_endpoint.ClusterLoadAssignment{
						ClusterName: path,
						Endpoints:   []*apiv3_endpoint.LocalityLbEndpoints{{LbEndpoints: lbEndpoints}},
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
func makeLbEndpoint(protocol, ip string, port int) *apiv3_endpoint.LbEndpoint {
	return &apiv3_endpoint.LbEndpoint{
		HostIdentifier: &apiv3_endpoint.LbEndpoint_Endpoint{
			Endpoint: &apiv3_endpoint.Endpoint{
				Address: &apiv3_core.Address{
					Address: &apiv3_core.Address_SocketAddress{
						SocketAddress: &apiv3_core.SocketAddress{
							Protocol:      apiv3_core.SocketAddress_Protocol(apiv3_core.SocketAddress_Protocol_value[protocol]),
							Address:       ip,
							PortSpecifier: &apiv3_core.SocketAddress_PortValue{PortValue: uint32(port)},
							Ipv4Compat:    true,
						},
					},
				},
			},
		},
	}
}
