package ambex

import (
	"fmt"
	"sort"
	"strings"

	apiv3_core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	apiv3_endpoint "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/endpoint/v3"
)

// The Endpoints struct is how Endpoint data gets communicated to ambex. This is a bit simpler than
// the envoy endpoint data structures, and also provides us a layer of indirection to buffer us from
// changes in envoy configuration, e.g. we can switch from v2 to v3 endpoint data, or add v3
// endpoint data fairly easily with this layer of indirection.
type Endpoints struct {
	Entries map[string][]*Endpoint
}

func (e *Endpoints) RoutesString() string {
	var routes []string
	for k, eps := range e.Entries {
		var addrs []string
		for _, ep := range eps {
			addr := fmt.Sprintf("%s:%s:%d", ep.Protocol, ep.Ip, ep.Port)
			addrs = append(addrs, addr)
		}
		routes = append(routes, fmt.Sprintf("%s=[%s]", k, strings.Join(addrs, ", ")))
	}
	sort.Strings(routes)
	return strings.Join(routes, "\n")
}

// ToMap_v3 produces a map with the envoy v3 friendly forms of all the endpoint data.
func (e *Endpoints) ToMap_v3() map[string]*apiv3_endpoint.ClusterLoadAssignment {
	result := map[string]*apiv3_endpoint.ClusterLoadAssignment{}
	for name, eps := range e.Entries {
		var endpoints []*apiv3_endpoint.LbEndpoint
		for _, ep := range eps {
			endpoints = append(endpoints, ep.ToLbEndpoint_v3())
		}
		loadAssignment := &apiv3_endpoint.ClusterLoadAssignment{
			ClusterName: name,
			Endpoints:   []*apiv3_endpoint.LocalityLbEndpoints{{LbEndpoints: endpoints}},
		}
		result[name] = loadAssignment
	}
	return result
}

// Endpoint contains the subset of fields we bother to expose.
type Endpoint struct {
	ClusterName string
	Ip          string
	Port        uint32
	Protocol    string
}

// ToLBEndpoint_v3 translates to envoy v3 frinedly form of the Endpoint data.
func (e *Endpoint) ToLbEndpoint_v3() *apiv3_endpoint.LbEndpoint {
	return &apiv3_endpoint.LbEndpoint{
		HostIdentifier: &apiv3_endpoint.LbEndpoint_Endpoint{
			Endpoint: &apiv3_endpoint.Endpoint{
				Address: &apiv3_core.Address{
					Address: &apiv3_core.Address_SocketAddress{
						SocketAddress: &apiv3_core.SocketAddress{
							Protocol: apiv3_core.SocketAddress_Protocol(apiv3_core.SocketAddress_Protocol_value[e.Protocol]),
							Address:  e.Ip,
							PortSpecifier: &apiv3_core.SocketAddress_PortValue{
								PortValue: e.Port,
							},
							Ipv4Compat: true,
						},
					},
				},
			},
		},
	}
}
