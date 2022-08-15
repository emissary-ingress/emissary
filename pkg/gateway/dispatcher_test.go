package gateway_test

import (
	// standard library
	"errors"
	"fmt"
	"testing"

	// third-party libraries
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"

	// envoy api v3
	v3core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	v3endpoint "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/endpoint/v3"
	v3listener "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/listener/v3"
	v3httpman "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"

	// envoy control plane
	ecp_cache_types "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	ecp_wellknown "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/wellknown"

	// first-party libraries
	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/gateway"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

func assertErrorContains(t *testing.T, err error, msg string) {
	t.Helper()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), msg)
	}
}

func TestDispatcherRegister(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_Foo))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	assert.NoError(t, disp.Upsert(foo))
	l := disp.GetListener(ctx, "bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
}

func TestDispatcherDuplicateRegister(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_Foo))
	require.NoError(t, err)
	err = disp.Register("Foo", wrapFooCompiler(compile_Foo))
	assertErrorContains(t, err, "duplicate")
}

func TestIsRegistered(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_Foo))
	require.NoError(t, err)
	assert.True(t, disp.IsRegistered("Foo"))
	assert.False(t, disp.IsRegistered("Bar"))
}

func TestDispatcherFaultIsolation1(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_Foo))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bang")
	foo.Spec.PanicArg = errors.New("bang bang!")
	err = disp.Upsert(foo)
	assertErrorContains(t, err, "error processing")
}

func TestDispatcherFaultIsolation2(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_Foo))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bang")
	foo.Spec.PanicArg = errors.New("bang bang!")
	err = disp.Upsert(foo)
	assertErrorContains(t, err, "error processing")
}

func TestDispatcherTransformError(t *testing.T) {
	t.Parallel()
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_FooWithErrors))
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

func compile_FooWithErrors(f *Foo) (*gateway.CompiledConfig, error) {
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
	}, nil
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
	ctx := dlog.NewTestContext(t, false)
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_Foo))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	assert.NoError(t, disp.Upsert(foo))
	l := disp.GetListener(ctx, "bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	disp.Delete(foo)
	l = disp.GetListener(ctx, "bar")
	require.Nil(t, l)
}

func TestDispatcherDeleteKey(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_Foo))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	assert.NoError(t, disp.Upsert(foo))
	l := disp.GetListener(ctx, "bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	disp.DeleteKey("Foo", "default", "foo")
	l = disp.GetListener(ctx, "bar")
	require.Nil(t, l)
}

func compile_Foo(f *Foo) (*gateway.CompiledConfig, error) {
	if f.Spec.Value == "bang" {
		return nil, f.Spec.PanicArg
	}
	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Listeners: []*gateway.CompiledListener{
			{
				Listener: &v3listener.Listener{Name: f.Spec.Value},
			},
		},
	}, nil
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
	ctx := dlog.NewTestContext(t, false)
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_FooWithRouteConfigName))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	assert.NoError(t, disp.Upsert(foo))
	l := disp.GetListener(ctx, "bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	r := disp.GetRouteConfiguration(ctx, "bar-routeconfig")
	require.NotNil(t, r)
	assert.Equal(t, "bar-routeconfig", r.Name)
}

func compile_FooWithRouteConfigName(f *Foo) (*gateway.CompiledConfig, error) {
	if f.Spec.Value == "bang" {
		return nil, f.Spec.PanicArg
	}

	name := f.Spec.Value
	rcName := fmt.Sprintf("%s-routeconfig", name)

	hcm := &v3httpman.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*v3httpman.HttpFilter{
			{Name: ecp_wellknown.CORS},
			{Name: ecp_wellknown.Router},
		},
		RouteSpecifier: &v3httpman.HttpConnectionManager_Rds{
			Rds: &v3httpman.Rds{
				ConfigSource: &v3core.ConfigSource{
					ConfigSourceSpecifier: &v3core.ConfigSource_Ads{
						Ads: &v3core.AggregatedConfigSource{},
					},
				},
				RouteConfigName: rcName,
			},
		},
	}
	hcmAny, err := anypb.New(hcm)
	if err != nil {
		return nil, err
	}

	l := &v3listener.Listener{
		Name: name,
		FilterChains: []*v3listener.FilterChain{
			{
				Filters: []*v3listener.Filter{
					{
						Name:       ecp_wellknown.HTTPConnectionManager,
						ConfigType: &v3listener.Filter_TypedConfig{TypedConfig: hcmAny},
					},
				},
			},
		},
	}

	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Listeners:    []*gateway.CompiledListener{{Listener: l}},
	}, nil
}

func TestDispatcherAssemblyWithEmptyRouteConfigName(t *testing.T) {
	t.Parallel()

	ctx := dlog.NewTestContext(t, false)

	disp := gateway.NewDispatcher()

	err := disp.Register("Foo", wrapFooCompiler(compileFooWithEmptyRouteConfigName))
	require.NoError(t, err)

	foo := makeFoo("default", "foo", "bar")
	err = disp.Upsert(foo)
	assert.NoError(t, err)

	// due to inconsistent SanptShot the listener returned should be nil
	listener := disp.GetListener(ctx, "bar")
	require.Nil(t, listener)
}

// compileFooWithEmptyRouteConfigName generates invalid RDS route configuration due to the
// RouteConfigname being empty. This will lead to an inconsistent snapshot, so calls
// to GetSnapShot will return a nil snapshot
func compileFooWithEmptyRouteConfigName(f *Foo) (*gateway.CompiledConfig, error) {
	name := f.Spec.Value

	hcm := &v3httpman.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*v3httpman.HttpFilter{
			{Name: ecp_wellknown.CORS},
			{Name: ecp_wellknown.Router},
		},
		RouteSpecifier: &v3httpman.HttpConnectionManager_Rds{
			Rds: &v3httpman.Rds{
				// explicitly adding RDS Config with no name to trigger snapshot Consistency to fail
				ConfigSource: &v3core.ConfigSource{
					ConfigSourceSpecifier: &v3core.ConfigSource_Ads{
						Ads: &v3core.AggregatedConfigSource{},
					},
				},
			},
		},
	}
	hcmAny, err := anypb.New(hcm)
	if err != nil {
		return nil, err
	}

	l := &v3listener.Listener{
		Name: name,
		FilterChains: []*v3listener.FilterChain{
			{
				Filters: []*v3listener.Filter{
					{
						Name:       ecp_wellknown.HTTPConnectionManager,
						ConfigType: &v3listener.Filter_TypedConfig{TypedConfig: hcmAny},
					},
				},
			},
		},
	}

	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Listeners:    []*gateway.CompiledListener{{Listener: l}},
	}, nil
}

func TestDispatcherAssemblyWithoutRds(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_FooWithoutRds))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	assert.NoError(t, disp.Upsert(foo))
	l := disp.GetListener(ctx, "bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	r := disp.GetRouteConfiguration(ctx, "bar")
	require.Nil(t, r)
}

func compile_FooWithoutRds(f *Foo) (*gateway.CompiledConfig, error) {
	name := f.Spec.Value

	hcm := &v3httpman.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*v3httpman.HttpFilter{
			{Name: ecp_wellknown.CORS},
			{Name: ecp_wellknown.Router},
		},
	}
	hcmAny, err := anypb.New(hcm)
	if err != nil {
		return nil, err
	}

	l := &v3listener.Listener{
		Name: name,
		FilterChains: []*v3listener.FilterChain{
			{
				Filters: []*v3listener.Filter{
					{
						Name: ecp_wellknown.RateLimit,
					},
					{
						Name:       ecp_wellknown.HTTPConnectionManager,
						ConfigType: &v3listener.Filter_TypedConfig{TypedConfig: hcmAny},
					},
				},
			},
		},
	}

	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Listeners:    []*gateway.CompiledListener{{Listener: l}},
	}, nil
}

func TestDispatcherAssemblyEndpointDefaulting(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_FooWithClusterRefs))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	err = disp.Upsert(foo)
	require.NoError(t, err)

	_, snapshot := disp.GetSnapshot(ctx)
	if snapshot == nil {
		assert.Fail(t, "unable to get a valid consistent snapshot")
		return // ensure that linter is happy due to possible nil pointer dereference below
	}

	found := false
	for _, r := range snapshot.Resources[ecp_cache_types.Endpoint].Items {
		cla := r.Resource.(*v3endpoint.ClusterLoadAssignment)
		if cla.ClusterName == "foo" && len(cla.Endpoints) == 0 {
			found = true
		}
	}
	if !found {
		assert.Fail(t, "no defaulted cluster load assignment")
	}
}

func wrapFooCompiler(inner func(*Foo) (*gateway.CompiledConfig, error)) func(kates.Object) (*gateway.CompiledConfig, error) {
	return func(untyped kates.Object) (*gateway.CompiledConfig, error) {
		return inner(untyped.(*Foo))
	}
}

func compile_FooWithClusterRefs(f *Foo) (*gateway.CompiledConfig, error) {
	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Routes: []*gateway.CompiledRoute{{
			ClusterRefs: []*gateway.ClusterRef{{Name: "foo"}},
		}},
	}, nil
}

func TestDispatcherAssemblyEndpointWatches(t *testing.T) {
	t.Parallel()
	ctx := dlog.NewTestContext(t, false)
	disp := gateway.NewDispatcher()
	err := disp.Register("Foo", wrapFooCompiler(compile_FooEndpointWatches))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	err = disp.Upsert(foo)
	require.NoError(t, err)
	disp.GetSnapshot(ctx)
	assert.True(t, disp.IsWatched("foo-ns", "foo"))
}

func compile_FooEndpointWatches(f *Foo) (*gateway.CompiledConfig, error) {
	return &gateway.CompiledConfig{
		CompiledItem: gateway.NewCompiledItem(gateway.SourceFromResource(f)),
		Routes: []*gateway.CompiledRoute{{
			CompiledItem: gateway.CompiledItem{Source: gateway.SourceFromResource(f), Namespace: "foo-ns"},
			ClusterRefs:  []*gateway.ClusterRef{{Name: "foo"}},
		}},
	}, nil
}
