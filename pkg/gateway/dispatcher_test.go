package gateway_test

import (
	"fmt"
	"testing"

	v2 "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	listener "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/listener"
	v2http "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/wellknown"
	"github.com/datawire/ambassador/v2/pkg/gateway"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertErrorContains(t *testing.T, err error, msg string) {
	assert.Error(t, err)
	assert.Contains(t, err.Error(), msg)
}

func TestDispatcherRegister(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_Foo)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	disp.Upsert(foo)
	l := disp.GetListener("bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
}

func TestDispatcherDuplicateRegister(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_Foo)
	require.NoError(t, err)
	err = disp.Register("Foo", compile_Foo)
	assertErrorContains(t, err, "duplicate")
}

func TestIsRegistered(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_Foo)
	require.NoError(t, err)
	assert.True(t, disp.IsRegistered("Foo"))
	assert.False(t, disp.IsRegistered("Bar"))
}

func TestDispatcherFaultIsolation1(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_Foo)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bang")
	foo.Spec.PanicArg = "bang bang!"
	err = disp.Upsert(foo)
	assertErrorContains(t, err, "error processing")
}

func TestDispatcherFaultIsolation2(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_Foo)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bang")
	foo.Spec.PanicArg = fmt.Errorf("bang bang!")
	err = disp.Upsert(foo)
	assertErrorContains(t, err, "error processing")
}

func TestDispatcherTransformError(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_FooWithErrors)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	err = disp.Upsert(foo)
	require.NoError(t, err)

	errors := disp.GetErrors()
	require.Len(t, errors, 6)
	assert.Equal(t, "Foo foo.default", errors[0].Source.Location())
	assert.Equal(t, "this is an error", errors[0].Error)

	assert.Equal(t, "listener 1 in Foo foo.default", errors[1].Source.Location())
	assert.Equal(t, "this is a listener error", errors[1].Error)

	assert.Equal(t, "route in Foo foo.default", errors[2].Source.Location())
	assert.Equal(t, "this is a route error", errors[2].Error)

	assert.Equal(t, "clusterRef in Foo foo.default", errors[3].Source.Location())
	assert.Equal(t, "this is a clusterRef error", errors[3].Error)

	assert.Equal(t, "cluster in Foo foo.default", errors[4].Source.Location())
	assert.Equal(t, "this is a cluster error", errors[4].Error)

	assert.Equal(t, "load assignment in Foo foo.default", errors[5].Source.Location())
	assert.Equal(t, "this is a load assignment error", errors[5].Error)
}

func compile_FooWithErrors(f *Foo) *gateway.CompiledConfig {
	src := gateway.SourceFromResource(f)
	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItemError(src, "this is an error"),
		Listeners: []*gateway.CompiledListener{
			{CompiledItem: gateway.NewCompiledItemError(gateway.Sourcef("listener %d in %s", 1, src),
				"this is a listener error")},
		},
		Routes: []*gateway.CompiledRoute{
			{
				CompiledItem: gateway.NewCompiledItemError(gateway.Sourcef("route in %s", src), "this is a route error"),
				ClusterRefs:  []*gateway.ClusterRef{{CompiledItem: gateway.NewCompiledItemError(gateway.Sourcef("clusterRef in %s", src), "this is a clusterRef error")}},
			},
		},
		Clusters: []*gateway.CompiledCluster{
			{CompiledItem: gateway.NewCompiledItemError(gateway.Sourcef("cluster in %s", src),
				"this is a cluster error")},
		},
		LoadAssignments: []*gateway.CompiledLoadAssignment{
			{CompiledItem: gateway.NewCompiledItemError(gateway.Sourcef("load assignment in %s", src),
				"this is a load assignment error")},
		},
	}
}

func TestDispatcherNoTransform(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	foo := makeFoo("default", "foo", "bar")
	err := disp.Upsert(foo)
	assertErrorContains(t, err, "no transform for kind")
}

func TestDispatcherDelete(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_Foo)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	disp.Upsert(foo)
	l := disp.GetListener("bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	disp.Delete(foo)
	l = disp.GetListener("bar")
	require.Nil(t, l)
}

func TestDispatcherDeleteKey(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_Foo)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	disp.Upsert(foo)
	l := disp.GetListener("bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	disp.DeleteKey("Foo", "default", "foo")
	l = disp.GetListener("bar")
	require.Nil(t, l)
}

func compile_Foo(f *Foo) *gateway.CompiledConfig {
	if f.Spec.Value == "bang" {
		panic(f.Spec.PanicArg)
	}
	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Listeners: []*gateway.CompiledListener{
			{
				Listener: &v2.Listener{Name: f.Spec.Value},
			},
		},
	}
}

func TestDispatcherUpsertYamlErr(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.UpsertYaml("{")
	assertErrorContains(t, err, "error converting")
	err = disp.UpsertYaml(`
---
kind: Gatewayyyy
apiVersion: networking.x-k8s.io/v1alpha1
metadata:
  name: my-gateway
spec:
  listeners:
  - protocol: HTTP
    port: 8080
`)
	assertErrorContains(t, err, "no transform for kind")
}

func TestDispatcherAssemblyWithRouteConfg(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_FooWithRouteConfigName)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	disp.Upsert(foo)
	l := disp.GetListener("bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	r := disp.GetRouteConfiguration("bar-routeconfig")
	require.NotNil(t, r)
	assert.Equal(t, "bar-routeconfig", r.Name)
}

func compile_FooWithRouteConfigName(f *Foo) *gateway.CompiledConfig {
	if f.Spec.Value == "bang" {
		panic(f.Spec.PanicArg)
	}

	name := f.Spec.Value
	rcName := fmt.Sprintf("%s-routeconfig", name)

	hcm := &v2http.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*v2http.HttpFilter{
			{Name: wellknown.CORS},
			{Name: wellknown.Router},
		},
		RouteSpecifier: &v2http.HttpConnectionManager_Rds{
			Rds: &v2http.Rds{
				ConfigSource: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: rcName,
			},
		},
	}
	hcmAny, err := ptypes.MarshalAny(hcm)
	if err != nil {
		panic(err)
	}

	l := &v2.Listener{
		Name: name,
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name:       wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{TypedConfig: hcmAny},
					},
				},
			},
		},
	}

	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Listeners:    []*gateway.CompiledListener{{Listener: l}},
	}
}

func TestDispatcherAssemblyWithEmptyRouteConfigName(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_FooWithEmptyRouteConfigName)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	disp.Upsert(foo)
	l := disp.GetListener("bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	// This is a bit weird, but the go control plane's consistency check seems to imply that an
	// empty route config name is ok.
	r := disp.GetRouteConfiguration("")
	require.NotNil(t, r)
	assert.Equal(t, "", r.Name)
}

func compile_FooWithEmptyRouteConfigName(f *Foo) *gateway.CompiledConfig {
	name := f.Spec.Value

	hcm := &v2http.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*v2http.HttpFilter{
			{Name: wellknown.CORS},
			{Name: wellknown.Router},
		},
		RouteSpecifier: &v2http.HttpConnectionManager_Rds{
			Rds: &v2http.Rds{
				ConfigSource: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
			},
		},
	}
	hcmAny, err := ptypes.MarshalAny(hcm)
	if err != nil {
		panic(err)
	}

	l := &v2.Listener{
		Name: name,
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name:       wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{TypedConfig: hcmAny},
					},
				},
			},
		},
	}

	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Listeners:    []*gateway.CompiledListener{{Listener: l}},
	}
}

func TestDispatcherAssemblyWithoutRds(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_FooWithoutRds)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	disp.Upsert(foo)
	l := disp.GetListener("bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	r := disp.GetRouteConfiguration("bar")
	require.Nil(t, r)
}

func compile_FooWithoutRds(f *Foo) *gateway.CompiledConfig {
	name := f.Spec.Value

	hcm := &v2http.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*v2http.HttpFilter{
			{Name: wellknown.CORS},
			{Name: wellknown.Router},
		},
	}
	hcmAny, err := ptypes.MarshalAny(hcm)
	if err != nil {
		panic(err)
	}

	l := &v2.Listener{
		Name: name,
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: wellknown.RateLimit,
					},
					{
						Name:       wellknown.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{TypedConfig: hcmAny},
					},
				},
			},
		},
	}

	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Listeners:    []*gateway.CompiledListener{{Listener: l}},
	}
}

func TestDispatcherAssemblyEndpointDefaulting(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_FooWithClusterRefs)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	err = disp.Upsert(foo)
	require.NoError(t, err)
	_, snap := disp.GetSnapshot()
	found := false
	for _, r := range snap.Resources[types.Endpoint].Items {
		cla := r.(*v2.ClusterLoadAssignment)
		if cla.ClusterName == "foo" && len(cla.Endpoints) == 0 {
			found = true
		}
	}
	if !found {
		assert.Fail(t, "no defaulted cluster load assignment")
	}
}

func compile_FooWithClusterRefs(f *Foo) *gateway.CompiledConfig {
	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Routes: []*gateway.CompiledRoute{{
			ClusterRefs: []*gateway.ClusterRef{{Name: "foo"}},
		}},
	}
}

func TestDispatcherAssemblyEndpointWatches(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", compile_FooEndpointWatches)
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	err = disp.Upsert(foo)
	require.NoError(t, err)
	disp.GetSnapshot()
	assert.True(t, disp.IsWatched("foo-ns", "foo"))
}

func compile_FooEndpointWatches(f *Foo) *gateway.CompiledConfig {
	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Routes: []*gateway.CompiledRoute{{
			CompiledItem: gateway.CompiledItem{Source: gateway.SourceFromResource(f), Namespace: "foo-ns"},
			ClusterRefs:  []*gateway.ClusterRef{{Name: "foo"}},
		}},
	}
}
