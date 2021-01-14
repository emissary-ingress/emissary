package ambex

import (
	"fmt"

	api "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	listener "github.com/datawire/ambassador/pkg/api/envoy/api/v2/listener"
	http "github.com/datawire/ambassador/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/resource/v2"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/proto"
)

// ListenerToRdsListener will take a listener definition and extract any inline RouteConfigurations
// replacing them with a reference to an RDS supplied route configuration. It does not modify the
// supplied listener, any configuration included in the result is copied from the input.
func ListenerToRdsListener(lnr *api.Listener) (*api.Listener, []*api.RouteConfiguration, error) {
	l := proto.Clone(lnr).(*api.Listener)
	var routes []*api.RouteConfiguration
	for _, fc := range l.FilterChains {
		for _, f := range fc.Filters {
			hcm := resource.GetHTTPConnectionManager(f)
			if hcm != nil {
				rs, ok := hcm.RouteSpecifier.(*http.HttpConnectionManager_RouteConfig)
				if ok {
					rc := rs.RouteConfig
					if rc.Name == "" {
						rc.Name = fmt.Sprintf("%s-routeconfig-%d", l.Name, len(routes))
					}
					routes = append(routes, rc)
					hcm.RouteSpecifier = &http.HttpConnectionManager_Rds{
						Rds: &http.Rds{
							ConfigSource: &core.ConfigSource{
								ConfigSourceSpecifier: &core.ConfigSource_Ads{
									Ads: &core.AggregatedConfigSource{},
								},
							},
							RouteConfigName: rc.Name,
						},
					}
				}
				any, err := ptypes.MarshalAny(hcm)
				if err != nil {
					return nil, nil, err
				}
				f.ConfigType = &listener.Filter_TypedConfig{TypedConfig: any}
			}
		}
	}

	return l, routes, nil
}
