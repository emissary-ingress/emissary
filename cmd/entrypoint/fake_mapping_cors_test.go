package entrypoint_test

import (
	"testing"

	"github.com/datawire/ambassador/cmd/entrypoint"
	envoy "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	route "github.com/datawire/ambassador/pkg/api/envoy/api/v2/route"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	http "github.com/datawire/ambassador/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/resource/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/wellknown"

	"github.com/stretchr/testify/assert"
)

func TestMappingCORSOriginsSlice(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: foo
  namespace: default
spec:
  prefix: /foo
  service: foo.default
  cors:
    origins:
     - foo.example.com
     - bar.example.com
`)
	f.Upsert(makeService("default", "foo"))
	f.Flush()
	snap := f.GetSnapshot(HasMapping("default", "foo"))
	assert.NotNil(t, snap)

	config := f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
		return FindCluster(config, ClusterNameContains("cluster_foo_default_default")) != nil
	})

	listener := findListener(config, func(l *envoy.Listener) bool {
		return l.Name == "ambassador-listener-8080"
	})

	assert.NotNil(t, listener)

	routeAction := findVirtualHostRoute(listener, func(r *route.RouteAction) bool {
		return r.GetCluster() == "cluster_foo_default_default"
	})
	assert.NotNil(t, routeAction)
	assert.NotNil(t, routeAction.Cors)
	assert.Equal(t, len(routeAction.Cors.AllowOriginStringMatch), 2)
	for _, m := range routeAction.Cors.AllowOriginStringMatch {
		assert.Contains(t, []string{"bar.example.com", "foo.example.com"}, m.GetExact())

	}
}

func TestMappingCORSOriginsString(t *testing.T) {
	f := entrypoint.RunFake(t, entrypoint.FakeConfig{EnvoyConfig: true}, nil)
	f.UpsertYAML(`
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: foo
  namespace: default
spec:
  prefix: /foo
  service: foo.default
  cors:
    origins: "foo.example.com,bar.example.com"
`)
	f.Upsert(makeService("default", "foo"))
	f.Flush()
	snap := f.GetSnapshot(HasMapping("default", "foo"))
	assert.NotNil(t, snap)

	config := f.GetEnvoyConfig(func(config *bootstrap.Bootstrap) bool {
		return FindCluster(config, ClusterNameContains("cluster_foo_default_default")) != nil
	})

	listener := findListener(config, func(l *envoy.Listener) bool {
		return l.Name == "ambassador-listener-8080"
	})

	assert.NotNil(t, listener)

	routeAction := findVirtualHostRoute(listener, func(r *route.RouteAction) bool {
		return r.GetCluster() == "cluster_foo_default_default"
	})
	assert.NotNil(t, routeAction)
	assert.NotNil(t, routeAction.Cors)
	assert.Equal(t, len(routeAction.Cors.AllowOriginStringMatch), 2)
	for _, m := range routeAction.Cors.AllowOriginStringMatch {
		assert.Contains(t, []string{"bar.example.com", "foo.example.com"}, m.GetExact())

	}
}

func findVirtualHostRoute(listener *envoy.Listener, predicate func(*route.RouteAction) bool) *route.RouteAction {
	for _, fc := range listener.FilterChains {
		for _, filter := range fc.Filters {
			if filter.Name != wellknown.HTTPConnectionManager {
				continue
			}
			hcm := resource.GetHTTPConnectionManager(filter)
			if hcm != nil {
				rs, ok := hcm.RouteSpecifier.(*http.HttpConnectionManager_RouteConfig)
				if ok {
					for _, vh := range rs.RouteConfig.VirtualHosts {
						for _, vhr := range vh.Routes {
							routeAction, ok := vhr.Action.(*route.Route_Route)
							if ok {
								if predicate(routeAction.Route) {
									return routeAction.Route
								}
							}
						}
					}
				}
			}
		}

	}
	return nil

}

func findListener(envoyConfig *bootstrap.Bootstrap, predicate func(*envoy.Listener) bool) *envoy.Listener {
	for _, listener := range envoyConfig.StaticResources.Listeners {
		if predicate(listener) {
			return listener
		}
	}
	return nil
}
