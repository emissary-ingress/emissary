package ambex

import (
	// standard library
	"context"
	"fmt"

	// third-party libraries
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	// envoy api v3
	v3cluster "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/cluster/v3"
	v3core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	v3endpoint "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/endpoint/v3"
	v3listener "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/listener/v3"
	v3route "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/route/v3"
	v3httpman "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"

	// envoy control plane
	ecp_cache_types "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	ecp_v3_resource "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	ecp_wellknown "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/wellknown"

	// first-party libraries
	"github.com/datawire/dlib/dlog"
)

// ListenerToRdsListener will take a listener definition and extract any inline RouteConfigurations
// replacing them with a reference to an RDS supplied route configuration. It does not modify the
// supplied listener, any configuration included in the result is copied from the input.
//
// If the input listener does not match the expected form it is simply copied, i.e. it is the
// identity transform for any inputs not matching the expected form.
//
// Example Input (that will get transformed in a non-identity way):
//   - a listener configured with an http connection manager
//   - that specifies an http router
//   - that supplies its RouteConfiguration inline via the route_config field
//
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
//
// Example Output:
//   - a duplicate listener that defines the "rds" field instead of the "route_config" field
//   - and a list of route configurations
//   - with route_config_name supplied in such a way as to correlate the two together
//
//   lnr, routes, err := ListenerToRdsListener(...)
//
//   lnr = {
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
//               "rds": {
//                 "config_source": {
//                   "ads": {}
//                 },
//                 "route_config_name": "ambassador-listener-8443-routeconfig-0"
//               }
//             }
//           }
//         ]
//       }
//     ]
//   }
//
//  routes = [
//    {
//      "name": "ambassador-listener-8443-routeconfig-0",
//      "virtual_hosts": [
//        {
//          "name": "ambassador-listener-8443-*",
//          "domains": ["*"],
//          "routes": [...],
//        }
//      ]
//    }
//  ]

// V3ListenerToRdsListener is the v3 variety of ListnerToRdsListener
func V3ListenerToRdsListener(lnr *v3listener.Listener) (*v3listener.Listener, []*v3route.RouteConfiguration, error) {
	l := proto.Clone(lnr).(*v3listener.Listener)
	var routes []*v3route.RouteConfiguration
	for _, fc := range l.FilterChains {
		for _, f := range fc.Filters {
			if f.Name != ecp_wellknown.HTTPConnectionManager {
				// We only know how to create an rds listener for HttpConnectionManager
				// listeners. We must ignore all other listeners.
				continue
			}

			// Note that the hcm configuration is stored in a protobuf any, so √the
			// GetHTTPConnectionManager is actually returning an unmarshalled copy.
			hcm := ecp_v3_resource.GetHTTPConnectionManager(f)
			if hcm != nil {
				// RouteSpecifier is a protobuf oneof that corresponds to the rds, route_config, and
				// scoped_routes fields. Only one of those may be set at a time.
				rs, ok := hcm.RouteSpecifier.(*v3httpman.HttpConnectionManager_RouteConfig)
				if ok {
					rc := rs.RouteConfig
					if rc.Name == "" {
						// Generate a unique name for the RouteConfiguration that we can use to
						// correlate the listener to the RDS record. We use the listener name plus
						// an index because there can be more than one route configuration
						// associated with a given listener.
						rc.Name = fmt.Sprintf("%s-routeconfig-%d", l.Name, len(routes))
					}
					routes = append(routes, rc)
					// Now that we have extracted and named the RouteConfiguration, we change the
					// RouteSpecifier from the inline RouteConfig variation to RDS via ADS. This
					// will cause it to use whatever ADS source is defined in the bootstrap
					// configuration.
					hcm.RouteSpecifier = &v3httpman.HttpConnectionManager_Rds{
						Rds: &v3httpman.Rds{
							ConfigSource: &v3core.ConfigSource{
								ConfigSourceSpecifier: &v3core.ConfigSource_Ads{
									Ads: &v3core.AggregatedConfigSource{},
								},
								ResourceApiVersion: v3core.ApiVersion_V3,
							},
							RouteConfigName: rc.Name,
						},
					}
				}

				// Because the hcm is a protobuf any, we need to remarshal it, we can't simply
				// expect the above modifications to take effect on our clone of the input. There is
				// also a protobuf oneof that includes the deprecated config and typed_config
				// fields.
				any, err := anypb.New(hcm)
				if err != nil {
					return nil, nil, err
				}
				f.ConfigType = &v3listener.Filter_TypedConfig{TypedConfig: any}
			}
		}
	}

	return l, routes, nil
}

// JoinEdsClustersV3 will perform an outer join operation between the eds clusters in the supplied
// clusterlist and the eds endpoint data in the supplied map. It will return a slice of
// ClusterLoadAssignments (cast to []ecp_cache_types.Resource) with endpoint data for all the eds clusters in
// the supplied list. If there is no map entry for a given cluster, an empty ClusterLoadAssignment
// will be synthesized. The result is a set of endpoints that are consistent (by the
// go-control-plane's definition of consistent) with the input clusters.
func JoinEdsClustersV3(ctx context.Context, clusters []ecp_cache_types.Resource, edsEndpoints map[string]*v3endpoint.ClusterLoadAssignment, edsBypass bool) (endpoints []ecp_cache_types.Resource) {
	for _, clu := range clusters {
		c := clu.(*v3cluster.Cluster)
		// Don't mess with non EDS clusters.
		if c.EdsClusterConfig == nil {
			continue
		}

		// By default, envoy will use the cluster name to lookup ClusterLoadAssignments unless the
		// ServiceName is supplied in the EdsClusterConfig.
		ref := c.EdsClusterConfig.ServiceName
		if ref == "" {
			ref = c.Name
		}

		// This change was introduced as a stop gap solution to mitigate the 503 issues when certificates are rotated.
		// The issue is CDS gets updated and waits for EDS to send ClusterLoadAssignment.
		// During this wait period calls that are coming through get hit with a 503 since the cluster is in a warming state.
		// The solution is to "hijack" the cluster and insert all the endpoints instead of relying on EDS.
		// Now there will be a discrepancy between envoy/envoy.json and the config envoy.
		if edsBypass {
			c.EdsClusterConfig = nil
			// Type 0 is STATIC
			c.ClusterDiscoveryType = &v3cluster.Cluster_Type{Type: 0}

			if ep, ok := edsEndpoints[ref]; ok {
				c.LoadAssignment = ep
			} else {
				c.LoadAssignment = &v3endpoint.ClusterLoadAssignment{
					ClusterName: ref,
					Endpoints:   []*v3endpoint.LocalityLbEndpoints{},
				}
			}
		} else {
			var source string
			ep, ok := edsEndpoints[ref]
			if ok {
				source = "found"
			} else {
				ep = &v3endpoint.ClusterLoadAssignment{
					ClusterName: ref,
					Endpoints:   []*v3endpoint.LocalityLbEndpoints{},
				}
				source = "synthesized"
			}

			dlog.Debugf(ctx, "%s envoy v3 ClusterLoadAssignment for cluster %s: %v", source, c.Name, ep)
			endpoints = append(endpoints, ep)
		}

	}

	return
}
