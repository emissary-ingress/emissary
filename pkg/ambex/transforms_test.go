package ambex

import (
	"testing"

	v3listener "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/listener/v3"
	v3route "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/route/v3"
)

//   {
//     "name": "...",
//     ...,
//     "filter_chains": [
//       {
//         "filter_chain_match": {...},
//         "filters": [
//           {
//             "name": "envoy.filters.network.http_connection_manager",
//             "typed_config": {
//               "@type": "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
//               "http_filters": [...],
//               "route_config": {
//                 "virtual_hosts": [
//                   {
//                     "name": "ambassador-listener-8443-*",
//                     "domains": ["*"],
//                     "routes": [...],
//                   }
//                 ]
//               }
//             }
//           }
//         ]
//       }
//     ]
//   }

func testTransform(t *testing.T) {
	testListener := v3listener.Listener{
		Name: "testListener",
		FilterChains: v3listener.FilterChain{
			Filters: v3listener.Filter{
				name: "envoy.filters.network.http_connection_manager",
				ConfigType: &v3listener.Filter_TypedConfig{
					"@type": "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
				},
				//How to get Routes here?
			},
		},
	}

	lnr, routes, err := V3ListenerToRdsListener(*testListener)

	//check if routes.GetName() = ambassador-listener-8443-routeconfig-376bf87fb310abb282f452533940481d-0
	//check if routes.GetVirtualHosts().GetName() = ambassador-listener-8443-*
	//check if routes.GetVirtualHosts().GetDomains() = *

}
