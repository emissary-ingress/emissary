package ambex

import (
	"fmt"
	"sort"
	"strings"

	v2 "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	v2core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	v2endpoint "github.com/datawire/ambassador/pkg/api/envoy/api/v2/endpoint"
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

// ToMap_v2 produces a map with the envoy v2 friendly forms of all the endpoint data.
func (e *Endpoints) ToMap_v2() map[string]*v2.ClusterLoadAssignment {
	result := map[string]*v2.ClusterLoadAssignment{}
	for name, eps := range e.Entries {
		var endpoints []*v2endpoint.LbEndpoint
		for _, ep := range eps {
			endpoints = append(endpoints, ep.ToLbEndpoint_v2())
		}
		loadAssignment := &v2.ClusterLoadAssignment{
			ClusterName: name,
			Endpoints:   []*v2endpoint.LocalityLbEndpoints{{LbEndpoints: endpoints}},
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

// ToLBEndpoint_v2 translates to envoy v2 frinedly form of the Endpoint data.
func (e *Endpoint) ToLbEndpoint_v2() *v2endpoint.LbEndpoint {
	return &v2endpoint.LbEndpoint{
		HostIdentifier: &v2endpoint.LbEndpoint_Endpoint{
			Endpoint: &v2endpoint.Endpoint{
				Address: &v2core.Address{
					Address: &v2core.Address_SocketAddress{
						SocketAddress: &v2core.SocketAddress{
							Protocol: v2core.SocketAddress_Protocol(v2core.SocketAddress_Protocol_value[e.Protocol]),
							Address:  e.Ip,
							PortSpecifier: &v2core.SocketAddress_PortValue{
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
