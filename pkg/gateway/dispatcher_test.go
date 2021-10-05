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

	// envoy api v2
	apiv2 "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	apiv2_core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	apiv2_listener "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/listener"
	apiv2_httpman "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/network/http_connection_manager/v2"

	// envoy control plane
	ecp_cache_types "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	ecp_wellknown "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/wellknown"

	// first-party libraries
	"github.com/datawire/ambassador/v2/pkg/gateway"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/dlib/dlog"
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
				Listener: &apiv2.Listener{Name: f.Spec.Value},
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

	hcm := &apiv2_httpman.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*apiv2_httpman.HttpFilter{
			{Name: ecp_wellknown.CORS},
			{Name: ecp_wellknown.Router},
		},
		RouteSpecifier: &apiv2_httpman.HttpConnectionManager_Rds{
			Rds: &apiv2_httpman.Rds{
				ConfigSource: &apiv2_core.ConfigSource{
					ConfigSourceSpecifier: &apiv2_core.ConfigSource_Ads{
						Ads: &apiv2_core.AggregatedConfigSource{},
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

	l := &apiv2.Listener{
		Name: name,
		FilterChains: []*apiv2_listener.FilterChain{
			{
				Filters: []*apiv2_listener.Filter{
					{
						Name:       ecp_wellknown.HTTPConnectionManager,
						ConfigType: &apiv2_listener.Filter_TypedConfig{TypedConfig: hcmAny},
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
	err := disp.Register("Foo", wrapFooCompiler(compile_FooWithEmptyRouteConfigName))
	require.NoError(t, err)
	foo := makeFoo("default", "foo", "bar")
	assert.NoError(t, disp.Upsert(foo))
	l := disp.GetListener(ctx, "bar")
	require.NotNil(t, l)
	assert.Equal(t, "bar", l.Name)
	// This is a bit weird, but the go control plane's consistency check seems to imply that an
	// empty route config name is ok.
	r := disp.GetRouteConfiguration(ctx, "")
	require.NotNil(t, r)
	assert.Equal(t, "", r.Name)
}

func compile_FooWithEmptyRouteConfigName(f *Foo) (*gateway.CompiledConfig, error) {
	name := f.Spec.Value

	hcm := &apiv2_httpman.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*apiv2_httpman.HttpFilter{
			{Name: ecp_wellknown.CORS},
			{Name: ecp_wellknown.Router},
		},
		RouteSpecifier: &apiv2_httpman.HttpConnectionManager_Rds{
			Rds: &apiv2_httpman.Rds{
				ConfigSource: &apiv2_core.ConfigSource{
					ConfigSourceSpecifier: &apiv2_core.ConfigSource_Ads{
						Ads: &apiv2_core.AggregatedConfigSource{},
					},
				},
			},
		},
	}
	hcmAny, err := anypb.New(hcm)
	if err != nil {
		return nil, err
	}

	l := &apiv2.Listener{
		Name: name,
		FilterChains: []*apiv2_listener.FilterChain{
			{
				Filters: []*apiv2_listener.Filter{
					{
						Name:       ecp_wellknown.HTTPConnectionManager,
						ConfigType: &apiv2_listener.Filter_TypedConfig{TypedConfig: hcmAny},
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

	hcm := &apiv2_httpman.HttpConnectionManager{
		StatPrefix: name,
		HttpFilters: []*apiv2_httpman.HttpFilter{
			{Name: ecp_wellknown.CORS},
			{Name: ecp_wellknown.Router},
		},
	}
	hcmAny, err := anypb.New(hcm)
	if err != nil {
		return nil, err
	}

	l := &apiv2.Listener{
		Name: name,
		FilterChains: []*apiv2_listener.FilterChain{
			{
				Filters: []*apiv2_listener.Filter{
					{
						Name: ecp_wellknown.RateLimit,
					},
					{
						Name:       ecp_wellknown.HTTPConnectionManager,
						ConfigType: &apiv2_listener.Filter_TypedConfig{TypedConfig: hcmAny},
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
	_, snap := disp.GetSnapshot(ctx)
	found := false
	for _, r := range snap.Resources[ecp_cache_types.Endpoint].Items {
		cla := r.(*apiv2.ClusterLoadAssignment)
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
