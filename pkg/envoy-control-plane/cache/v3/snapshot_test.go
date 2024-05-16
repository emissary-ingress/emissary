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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	rsrc "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/test/resource/v3"
)

// Tests the snapshot defined in simple_test.go to ensure it is consistent.
func TestTestSnapshotIsConsistent(t *testing.T) {
	snapshot := fixture.snapshot()

	if err := snapshot.Consistent(); err != nil {
		t.Errorf("got inconsistent snapshot for %#v\nerr=%s", snapshot, err.Error())
	}
}

func TestSnapshotWithOnlyEndpointIsInconsistent(t *testing.T) {
	if snap, _ := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.EndpointType: {testEndpoint},
	}); snap.Consistent() == nil {
		t.Errorf("got consistent snapshot %#v", snap)
	}
}

func TestClusterWithMissingEndpointIsInconsistent(t *testing.T) {
	if snap, _ := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.EndpointType: {resource.MakeEndpoint("missing", 8080)},
		rsrc.ClusterType:  {testCluster},
	}); snap.Consistent() == nil {
		t.Errorf("got consistent snapshot %#v", snap)
	}
}

func TestListenerWithMissingRoutesIsInconsistent(t *testing.T) {
	if snap, _ := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.ListenerType: {testListener},
	}); snap.Consistent() == nil {
		t.Errorf("got consistent snapshot %#v", snap)
	}
}

func TestListenerWithUnidentifiedRouteIsInconsistent(t *testing.T) {
	if snap, _ := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.RouteType:    {resource.MakeRouteConfig("test", clusterName)},
		rsrc.ListenerType: {testListener},
	}); snap.Consistent() == nil {
		t.Errorf("got consistent snapshot %#v", snap)
	}
}

func TestRouteListenerWithRouteIsConsistent(t *testing.T) {
	snap, _ := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.ListenerType: {
			resource.MakeRouteHTTPListener(resource.Xds, "listener1", 80, "testRoute0"),
		},
		rsrc.RouteType: {
			resource.MakeRouteConfig("testRoute0", clusterName),
		},
	})

	if err := snap.Consistent(); err != nil {
		t.Errorf("got inconsistent snapshot %s, %#v", err.Error(), snap)
	}
}

func TestScopedRouteListenerWithScopedRouteOnlyIsInconsistent(t *testing.T) {
	if snap, _ := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.ListenerType: {
			resource.MakeScopedRouteHTTPListener(resource.Xds, "listener0", 80),
		},
		rsrc.ScopedRouteType: {
			resource.MakeScopedRouteConfig("scopedRoute0", "testRoute0", []string{"1.2.3.4"}),
		},
	}); snap.Consistent() == nil {
		t.Errorf("got consistent snapshot %#v", snap)
	}
}

func TestScopedRouteListenerWithScopedRouteAndRouteIsConsistent(t *testing.T) {
	snap, _ := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.ListenerType: {
			resource.MakeScopedRouteHTTPListener(resource.Xds, "listener0", 80),
		},
		rsrc.ScopedRouteType: {
			resource.MakeScopedRouteConfig("scopedRoute0", "testRoute0", []string{"1.2.3.4"}),
		},
		rsrc.RouteType: {
			resource.MakeRouteConfig("testRoute0", clusterName),
		},
	})

	require.NoError(t, snap.Consistent(), "got inconsistent snapshot %#v", snap)
}

func TestScopedRouteListenerWithInlineScopedRouteAndRouteIsConsistent(t *testing.T) {
	snap, err := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.ListenerType: {
			resource.MakeScopedRouteHTTPListenerForRoute(resource.Xds, "listener0", 80, "testRoute0"),
		},
		rsrc.RouteType: {
			resource.MakeRouteConfig("testRoute0", clusterName),
		},
	})

	require.NoError(t, err)
	require.NoError(t, snap.Consistent())
}

func TestScopedRouteListenerWithInlineScopedRouteAndNoRouteIsInconsistent(t *testing.T) {
	snap, err := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.ListenerType: {
			resource.MakeScopedRouteHTTPListenerForRoute(resource.Xds, "listener0", 80, "testRoute0"),
		},
		rsrc.RouteType: {
			resource.MakeRouteConfig("testRoute1", clusterName),
		},
	})

	require.NoError(t, err)
	require.Error(t, snap.Consistent())
}

func TestMultipleListenersWithScopedRouteAndRouteIsConsistent(t *testing.T) {
	snap, _ := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		rsrc.ListenerType: {
			resource.MakeScopedRouteHTTPListener(resource.Xds, "listener0", 80),
			resource.MakeRouteHTTPListener(resource.Xds, "listener1", 80, "testRoute1"),
		},
		rsrc.ScopedRouteType: {
			resource.MakeScopedRouteConfig("scopedRoute0", "testRoute0", []string{"1.2.3.4"}),
		},
		rsrc.RouteType: {
			resource.MakeRouteConfig("testRoute0", clusterName),
			resource.MakeRouteConfig("testRoute1", clusterName),
		},
	})

	if err := snap.Consistent(); err != nil {
		t.Errorf("got inconsistent snapshot %s, %#v", err.Error(), snap)
	}
}

func TestSnapshotGetters(t *testing.T) {
	var nilsnap *cache.Snapshot
	if out := nilsnap.GetResources(rsrc.EndpointType); out != nil {
		t.Errorf("got non-empty resources for nil snapshot: %#v", out)
	}
	if out := nilsnap.Consistent(); out == nil {
		t.Errorf("nil snapshot should be inconsistent")
	}
	if out := nilsnap.GetVersion(rsrc.EndpointType); out != "" {
		t.Errorf("got non-empty version for nil snapshot: %#v", out)
	}

	snapshot := fixture.snapshot()
	if out := snapshot.GetResources("not a type"); out != nil {
		t.Errorf("got non-empty resources for unknown type: %#v", out)
	}
	if out := snapshot.GetVersion("not a type"); out != "" {
		t.Errorf("got non-empty version for unknown type: %#v", out)
	}
}

func TestNewSnapshotBadType(t *testing.T) {
	snap, err := cache.NewSnapshot(fixture.version, map[rsrc.Type][]types.Resource{
		"random.type": nil,
	})

	// Should receive an error from an unknown type
	require.Error(t, err)
	assert.Nil(t, snap)
}
