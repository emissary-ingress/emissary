package entrypoint_test

import (
	"testing"

	bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	v3listener "github.com/datawire/ambassador/v2/pkg/api/envoy/config/listener/v3"
	route "github.com/datawire/ambassador/v2/pkg/api/envoy/config/route/v3"
	http "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/wellknown"
	"github.com/stretchr/testify/assert"
)

// findListener finds the first listener in a given Envoy configuration that matches a
// given predicate. If no listener is found, nil is returned.
//
// Obviously, in a perfect world, the given predicate would be constructed to match only
// a single listener...
func findListener(envoyConfig *bootstrap.Bootstrap, predicate func(*v3listener.Listener) bool) *v3listener.Listener {
	for _, listener := range envoyConfig.StaticResources.Listeners {
		if predicate(listener) {
			return listener
		}
	}
	return nil
}

// findListenerByName finds uses findListener to find a listener with a given name.
func findListenerByName(envoyConfig *bootstrap.Bootstrap, name string) *v3listener.Listener {
	return findListener(envoyConfig, func(listener *v3listener.Listener) bool {
		return listener.Name == name
	})
}

// mustFindListenerByName looks for a listener with a given name, and asserts that it
// must be present.
func mustFindListenerByName(t *testing.T, envoyConfig *bootstrap.Bootstrap, name string) *v3listener.Listener {
	listener := findListenerByName(envoyConfig, name)
	assert.NotNil(t, listener)
	return listener
}

func findVirtualHostRoute(listener *v3listener.Listener, predicate func(*route.Route) bool) []*route.Route {
	routes := make([]*route.Route, 0)

	// fmt.Printf("---- findVirtualHostRoute\n")

	for _, chain := range listener.FilterChains {
		for _, filter := range chain.Filters {
			if filter.Name != wellknown.HTTPConnectionManager {
				continue
			}

			hcm := resource.GetHTTPConnectionManager(filter)

			if hcm != nil {
				rs, ok := hcm.RouteSpecifier.(*http.HttpConnectionManager_RouteConfig)

				if ok {
					for _, vh := range rs.RouteConfig.VirtualHosts {
						for _, vhr := range vh.Routes {
							// if !(strings.HasPrefix(vhr.Match.GetPrefix(), "/ambassador/")) {
							// 	fmt.Printf("ROUTE: #%v\n", vhr)
							// }

							if predicate(vhr) {
								routes = append(routes, vhr)
							}
						}
					}
				}
			}
		}
	}

	return routes
}

func findVirtualHostRouteAction(listener *v3listener.Listener, predicate func(*route.RouteAction) bool) *route.RouteAction {
	routes := findVirtualHostRoute(listener, func(r *route.Route) bool {
		routeAction, ok := r.Action.(*route.Route_Route)

		if ok {
			return predicate(routeAction.Route)
		}

		return false
	})

	if len(routes) == 0 {
		return nil
	}

	return routes[0].Action.(*route.Route_Route).Route
}
