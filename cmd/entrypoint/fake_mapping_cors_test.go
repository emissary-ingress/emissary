package entrypoint_test

import (
	"testing"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	v3listener "github.com/datawire/ambassador/v2/pkg/api/envoy/config/listener/v3"
	route "github.com/datawire/ambassador/v2/pkg/api/envoy/config/route/v3"
	http "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/wellknown"

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

	listener := findListener(config, func(l *v3listener.Listener) bool {
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

	listener := findListener(config, func(l *v3listener.Listener) bool {
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

func findVirtualHostRoute(listener *v3listener.Listener, predicate func(*route.RouteAction) bool) *route.RouteAction {
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

func findListener(envoyConfig *bootstrap.Bootstrap, predicate func(*v3listener.Listener) bool) *v3listener.Listener {
	for _, listener := range envoyConfig.StaticResources.Listeners {
		if predicate(listener) {
			return listener
		}
	}
	return nil
}
