// Copyright 2018 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package cache_test

import (
	"reflect"
	"testing"

	cluster "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/cluster/v3"
	route "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/route/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	rsrc "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/test/resource/v3"
)

const (
	clusterName         = "cluster0"
	routeName           = "route0"
	scopedRouteName     = "scopedRoute0"
	listenerName        = "listener0"
	scopedListenerName  = "scopedListener0"
	runtimeName         = "runtime0"
	tlsName             = "secret0"
	rootName            = "root0"
	extensionConfigName = "extensionConfig0"
)

var (
	testEndpoint        = resource.MakeEndpoint(clusterName, 8080)
	testCluster         = resource.MakeCluster(resource.Ads, clusterName)
	testRoute           = resource.MakeRoute(routeName, clusterName)
	testScopedRoute     = resource.MakeScopedRoute(scopedRouteName, routeName, []string{"1.2.3.4"})
	testListener        = resource.MakeRouteHTTPListener(resource.Ads, listenerName, 80, routeName)
	testScopedListener  = resource.MakeScopedRouteHTTPListener(resource.Ads, scopedListenerName, 80, scopedRouteName)
	testRuntime         = resource.MakeRuntime(runtimeName)
	testSecret          = resource.MakeSecrets(tlsName, rootName)
	testExtensionConfig = resource.MakeExtensionConfig(resource.Ads, extensionConfigName, routeName)
)

func TestValidate(t *testing.T) {
	if err := testEndpoint.Validate(); err != nil {
		t.Error(err)
	}
	if err := testCluster.Validate(); err != nil {
		t.Error(err)
	}
	if err := testRoute.Validate(); err != nil {
		t.Error(err)
	}
	if err := testScopedRoute.Validate(); err != nil {
		t.Error(err)
	}
	if err := testListener.Validate(); err != nil {
		t.Error(err)
	}
	if err := testRuntime.Validate(); err != nil {
		t.Error(err)
	}

	invalidRoute := &route.RouteConfiguration{
		Name: "test",
		VirtualHosts: []*route.VirtualHost{{
			Name:    "test",
			Domains: []string{},
		}},
	}

	if err := invalidRoute.Validate(); err == nil {
		t.Error("expected an error")
	}
	if err := invalidRoute.VirtualHosts[0].Validate(); err == nil {
		t.Error("expected an error")
	}
}

func TestGetResourceName(t *testing.T) {
	if name := cache.GetResourceName(testEndpoint); name != clusterName {
		t.Errorf("GetResourceName(%v) => got %q, want %q", testEndpoint, name, clusterName)
	}
	if name := cache.GetResourceName(testCluster); name != clusterName {
		t.Errorf("GetResourceName(%v) => got %q, want %q", testCluster, name, clusterName)
	}
	if name := cache.GetResourceName(testRoute); name != routeName {
		t.Errorf("GetResourceName(%v) => got %q, want %q", testRoute, name, routeName)
	}
	if name := cache.GetResourceName(testScopedRoute); name != scopedRouteName {
		t.Errorf("GetResourceName(%v) => got %q, want %q", testScopedRoute, name, scopedRouteName)
	}
	if name := cache.GetResourceName(testListener); name != listenerName {
		t.Errorf("GetResourceName(%v) => got %q, want %q", testListener, name, listenerName)
	}
	if name := cache.GetResourceName(testRuntime); name != runtimeName {
		t.Errorf("GetResourceName(%v) => got %q, want %q", testRuntime, name, runtimeName)
	}
	if name := cache.GetResourceName(nil); name != "" {
		t.Errorf("GetResourceName(nil) => got %q, want none", name)
	}
}

func TestGetResourceReferences(t *testing.T) {
	cases := []struct {
		in  types.Resource
		out map[rsrc.Type]map[string]bool
	}{
		{
			in:  nil,
			out: map[rsrc.Type]map[string]bool{},
		},
		{
			in:  testCluster,
			out: map[rsrc.Type]map[string]bool{rsrc.EndpointType: {clusterName: true}},
		},
		{
			in: &cluster.Cluster{Name: clusterName, ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
				EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{ServiceName: "test"}},
			out: map[rsrc.Type]map[string]bool{rsrc.EndpointType: {"test": true}},
		},
		{
			in:  resource.MakeScopedRouteHTTPListener(resource.Xds, listenerName, 80, scopedRouteName),
			out: map[rsrc.Type]map[string]bool{rsrc.ScopedRouteType: {scopedRouteName: true}},
		},
		{
			in:  resource.MakeRouteHTTPListener(resource.Ads, listenerName, 80, routeName),
			out: map[rsrc.Type]map[string]bool{rsrc.RouteType: {routeName: true}},
		},
		{
			in:  resource.MakeTCPListener(listenerName, 80, clusterName),
			out: map[rsrc.Type]map[string]bool{},
		},
		{
			in:  testRoute,
			out: map[rsrc.Type]map[string]bool{},
		},
		{
			in:  testScopedRoute,
			out: map[rsrc.Type]map[string]bool{rsrc.RouteType: {routeName: true}},
		},
		{
			in:  testEndpoint,
			out: map[rsrc.Type]map[string]bool{},
		},
		{
			in:  testRuntime,
			out: map[rsrc.Type]map[string]bool{},
		},
	}
	for _, cs := range cases {
		names := cache.GetResourceReferences(cache.IndexResourcesByName([]types.ResourceWithTTL{{Resource: cs.in}}))
		if !reflect.DeepEqual(names, cs.out) {
			t.Errorf("GetResourceReferences(%v) => got %v, want %v", cs.in, names, cs.out)
		}
	}
}

func TestGetAllResourceReferencesReturnsExpectedRefs(t *testing.T) {
	expected := map[rsrc.Type]map[string]bool{
		rsrc.RouteType:       {routeName: true},
		rsrc.ScopedRouteType: {scopedRouteName: true},
		rsrc.EndpointType:    {clusterName: true},
	}

	resources := [types.UnknownType]cache.Resources{}
	resources[types.Endpoint] = cache.NewResources("1", []types.Resource{testEndpoint})
	resources[types.Cluster] = cache.NewResources("1", []types.Resource{testCluster})
	resources[types.Route] = cache.NewResources("1", []types.Resource{testRoute})
	resources[types.Listener] = cache.NewResources("1", []types.Resource{testListener, testScopedListener})
	resources[types.ScopedRoute] = cache.NewResources("1", []types.Resource{testScopedRoute})
	actual := cache.GetAllResourceReferences(resources)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("GetAllResourceReferences(%v) => got %v, want %v", resources, actual, expected)
	}
}
