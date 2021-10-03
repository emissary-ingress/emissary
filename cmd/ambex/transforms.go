package ambex

import (
	// standard library
	"context"
	"fmt"

	// third-party libraries
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	// envoy api v2
	apiv2 "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	apiv2_core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	apiv2_endpoint "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/endpoint"
	apiv2_listener "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/listener"
	apiv2_httpman "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/network/http_connection_manager/v2"

	// envoy api v3
	apiv3_cluster "github.com/datawire/ambassador/v2/pkg/api/envoy/config/cluster/v3"
	apiv3_core "github.com/datawire/ambassador/v2/pkg/api/envoy/config/core/v3"
	apiv3_endpoint "github.com/datawire/ambassador/v2/pkg/api/envoy/config/endpoint/v3"
	apiv3_listener "github.com/datawire/ambassador/v2/pkg/api/envoy/config/listener/v3"
	apiv3_route "github.com/datawire/ambassador/v2/pkg/api/envoy/config/route/v3"
	apiv3_httpman "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"

	// envoy control plane
	ecp_cache_types "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	ecp_v2_resource "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v2"
	ecp_v3_resource "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v3"
	ecp_wellknown "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/wellknown"

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
func ListenerToRdsListener(lnr *apiv2.Listener) (*apiv2.Listener, []*apiv2.RouteConfiguration, error) {
	l := proto.Clone(lnr).(*apiv2.Listener)
	var routes []*apiv2.RouteConfiguration
	for _, fc := range l.FilterChains {
		for _, f := range fc.Filters {
			if f.Name != ecp_wellknown.HTTPConnectionManager {
				// We only know how to create an rds listener for HttpConnectionManager
				// listeners. We must ignore all other listeners.
				continue
			}

			// Note that the hcm configuration is stored in a protobuf any, so √the
			// GetHTTPConnectionManager is actually returning an unmarshalled copy.
			hcm := ecp_v2_resource.GetHTTPConnectionManager(f)
			if hcm != nil {
				// RouteSpecifier is a protobuf oneof that corresponds to the rds, route_config, and
				// scoped_routes fields. Only one of those may be set at a time.
				rs, ok := hcm.RouteSpecifier.(*apiv2_httpman.HttpConnectionManager_RouteConfig)
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
					hcm.RouteSpecifier = &apiv2_httpman.HttpConnectionManager_Rds{
						Rds: &apiv2_httpman.Rds{
							ConfigSource: &apiv2_core.ConfigSource{
								ConfigSourceSpecifier: &apiv2_core.ConfigSource_Ads{
									Ads: &apiv2_core.AggregatedConfigSource{},
								},
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
				f.ConfigType = &apiv2_listener.Filter_TypedConfig{TypedConfig: any}
			}
		}
	}

	return l, routes, nil
}

// V3ListenerToRdsListener is the v3 variety of ListnerToRdsListener
func V3ListenerToRdsListener(lnr *apiv3_listener.Listener) (*apiv3_listener.Listener, []*apiv3_route.RouteConfiguration, error) {
	l := proto.Clone(lnr).(*apiv3_listener.Listener)
	var routes []*apiv3_route.RouteConfiguration
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
				rs, ok := hcm.RouteSpecifier.(*apiv3_httpman.HttpConnectionManager_RouteConfig)
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
					hcm.RouteSpecifier = &apiv3_httpman.HttpConnectionManager_Rds{
						Rds: &apiv3_httpman.Rds{
							ConfigSource: &apiv3_core.ConfigSource{
								ConfigSourceSpecifier: &apiv3_core.ConfigSource_Ads{
									Ads: &apiv3_core.AggregatedConfigSource{},
								},
								ResourceApiVersion: apiv3_core.ApiVersion_V3,
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
				f.ConfigType = &apiv3_listener.Filter_TypedConfig{TypedConfig: any}
			}
		}
	}

	return l, routes, nil
}

// JoinEdsClusters will perform an outer join operation between the eds clusters in the supplied
// clusterlist and the eds endpoint data in the supplied map. It will return a slice of
// ClusterLoadAssignments (cast to []ecp_cache_types.Resource) with endpoint data for all the eds clusters in
// the supplied list. If there is no map entry for a given cluster, an empty ClusterLoadAssignment
// will be synthesized. The result is a set of endpoints that are consistent (by the
// go-control-plane's definition of consistent) with the input clusters.
func JoinEdsClusters(ctx context.Context, clusters []ecp_cache_types.Resource, edsEndpoints map[string]*apiv2.ClusterLoadAssignment) (endpoints []ecp_cache_types.Resource) {
	for _, clu := range clusters {
		c := clu.(*apiv2.Cluster)
		// Don't mess with non EDS clusters.
		if c.EdsClusterConfig == nil {
			continue
		}

		// By default envoy will use the cluster name to lookup ClusterLoadAssignments unless the
		// ServiceName is supplied in the EdsClusterConfig.
		ref := c.EdsClusterConfig.ServiceName
		if ref == "" {
			ref = c.Name
		}

		var source string
		ep, ok := edsEndpoints[ref]
		if ok {
			source = "found"
		} else {
			ep = &apiv2.ClusterLoadAssignment{
				ClusterName: ref,
				Endpoints:   []*apiv2_endpoint.LocalityLbEndpoints{},
			}
			source = "synthesized"
		}

		dlog.Debugf(ctx, "%s envoy v2 ClusterLoadAssignment for cluster %s: %v", source, c.Name, ep)
		endpoints = append(endpoints, ep)
	}

	return
}

// JoinEdsClustersV3 will perform an outer join operation between the eds clusters in the supplied
// clusterlist and the eds endpoint data in the supplied map. It will return a slice of
// ClusterLoadAssignments (cast to []ecp_cache_types.Resource) with endpoint data for all the eds clusters in
// the supplied list. If there is no map entry for a given cluster, an empty ClusterLoadAssignment
// will be synthesized. The result is a set of endpoints that are consistent (by the
// go-control-plane's definition of consistent) with the input clusters.
func JoinEdsClustersV3(ctx context.Context, clusters []ecp_cache_types.Resource, edsEndpoints map[string]*apiv3_endpoint.ClusterLoadAssignment) (endpoints []ecp_cache_types.Resource) {
	for _, clu := range clusters {
		c := clu.(*apiv3_cluster.Cluster)
		// Don't mess with non EDS clusters.
		if c.EdsClusterConfig == nil {
			continue
		}

		// By default envoy will use the cluster name to lookup ClusterLoadAssignments unless the
		// ServiceName is supplied in the EdsClusterConfig.
		ref := c.EdsClusterConfig.ServiceName
		if ref == "" {
			ref = c.Name
		}

		var source string
		ep, ok := edsEndpoints[ref]
		if ok {
			source = "found"
		} else {
			ep = &apiv3_endpoint.ClusterLoadAssignment{
				ClusterName: ref,
				Endpoints:   []*apiv3_endpoint.LocalityLbEndpoints{},
			}
			source = "synthesized"
		}

		dlog.Debugf(ctx, "%s envoy v3 ClusterLoadAssignment for cluster %s: %v", source, c.Name, ep)
		endpoints = append(endpoints, ep)
	}

	return
}
