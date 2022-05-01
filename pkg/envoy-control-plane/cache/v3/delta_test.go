package cache_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	core "github.com/datawire/ambassador/v2/pkg/api/envoy/config/core/v3"
	discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v3"
	rsrc "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/stream/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/test/resource/v3"
)

func TestSnapshotCacheDeltaWatch(t *testing.T) {
	c := cache.NewSnapshotCache(false, group{}, logger{t: t})
	watches := make(map[string]chan cache.DeltaResponse)

	// Make our initial request as a wildcard to get all resources and make sure the wildcard requesting works as intended
	for _, typ := range testTypes {
		watches[typ], _ = c.CreateDeltaWatch(&discovery.DeltaDiscoveryRequest{
			Node: &core.Node{
				Id: "node",
			},
			TypeUrl:                typ,
			ResourceNamesSubscribe: names[typ],
		}, &stream.StreamState{IsWildcard: true, ResourceVersions: nil})
	}

	if err := c.SetSnapshot(key, snapshot); err != nil {
		t.Fatal(err)
	}

	vm := make(map[string]map[string]string)
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			select {
			case out := <-watches[typ]:
				if !reflect.DeepEqual(cache.IndexRawResourcesByName(out.(*cache.RawDeltaResponse).Resources), snapshot.GetResources(typ)) {
					t.Errorf("got resources %v, want %v", out.(*cache.RawDeltaResponse).Resources, snapshot.GetResources(typ))
				}
				vMap := out.GetNextVersionMap()
				vm[typ] = vMap
			case <-time.After(time.Second):
				t.Fatal("failed to receive snapshot response")
			}
		})
	}

	// On re-request we want to use non-wildcard so we can verify the logic path of not requesting
	// all resources as well as individual resource removals
	for _, typ := range testTypes {
		watches[typ], _ = c.CreateDeltaWatch(&discovery.DeltaDiscoveryRequest{
			Node: &core.Node{
				Id: "node",
			},
			TypeUrl:                typ,
			ResourceNamesSubscribe: names[typ],
		}, &stream.StreamState{IsWildcard: false, ResourceVersions: vm[typ]})
	}

	if count := c.GetStatusInfo(key).GetNumDeltaWatches(); count != len(testTypes) {
		t.Errorf("watches should be created for the latest version, saw %d watches expected %d", count, len(testTypes))
	}

	// set partially-versioned snapshot
	snapshot2 := snapshot
	snapshot2.Resources[types.Endpoint] = cache.NewResources(version2, []types.Resource{resource.MakeEndpoint(clusterName, 9090)})
	if err := c.SetSnapshot(key, snapshot2); err != nil {
		t.Fatal(err)
	}
	if count := c.GetStatusInfo(key).GetNumDeltaWatches(); count != len(testTypes)-1 {
		t.Errorf("watches should be preserved for all but one, got: %d open watches instead of the expected %d open watches", count, len(testTypes)-1)
	}

	// validate response for endpoints
	select {
	case out := <-watches[testTypes[0]]:
		if !reflect.DeepEqual(cache.IndexRawResourcesByName(out.(*cache.RawDeltaResponse).Resources), snapshot2.GetResources(rsrc.EndpointType)) {
			t.Fatalf("got resources %v, want %v", out.(*cache.RawDeltaResponse).Resources, snapshot2.GetResources(rsrc.EndpointType))
		}
		vMap := out.GetNextVersionMap()
		vm[testTypes[0]] = vMap
	case <-time.After(time.Second):
		t.Fatal("failed to receive snapshot response")
	}
}

func TestDeltaRemoveResources(t *testing.T) {
	c := cache.NewSnapshotCache(false, group{}, logger{t: t})
	watches := make(map[string]chan cache.DeltaResponse)

	for _, typ := range testTypes {
		// We don't specify any resource name subscriptions here because we want to make sure we test wildcard
		// functionality. This means we should receive all resources back without requesting a subscription by name.
		watches[typ], _ = c.CreateDeltaWatch(&discovery.DeltaDiscoveryRequest{
			Node: &core.Node{
				Id: "node",
			},
			TypeUrl: typ,
		}, &stream.StreamState{IsWildcard: true, ResourceVersions: nil})
	}

	if err := c.SetSnapshot(key, snapshot); err != nil {
		t.Fatal(err)
	}

	versionMap := make(map[string]map[string]string)
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			select {
			case out := <-watches[typ]:
				if !reflect.DeepEqual(cache.IndexRawResourcesByName(out.(*cache.RawDeltaResponse).Resources), snapshot.GetResources(typ)) {
					t.Errorf("got resources %v, want %v", out.(*cache.RawDeltaResponse).Resources, snapshot.GetResources(typ))
				}
				nextVersionMap := out.GetNextVersionMap()
				versionMap[typ] = nextVersionMap
			case <-time.After(time.Second):
				t.Fatal("failed to receive a snapshot response")
			}
		})
	}

	// We want to continue to do wildcard requests here so we can later
	// test the removal of certain resources from a partial snapshot
	for _, typ := range testTypes {
		watches[typ], _ = c.CreateDeltaWatch(&discovery.DeltaDiscoveryRequest{
			Node: &core.Node{
				Id: "node",
			},
			TypeUrl: typ,
		}, &stream.StreamState{IsWildcard: true, ResourceVersions: versionMap[typ]})
	}

	if count := c.GetStatusInfo(key).GetNumDeltaWatches(); count != len(testTypes) {
		t.Errorf("watches should be created for the latest version, saw %d watches expected %d", count, len(testTypes))
	}

	// set a partially versioned snapshot with no endpoints
	snapshot2 := snapshot
	snapshot2.Resources[types.Endpoint] = cache.NewResources(version2, []types.Resource{})
	if err := c.SetSnapshot(key, snapshot2); err != nil {
		t.Fatal(err)
	}

	// validate response for endpoints
	select {
	case out := <-watches[testTypes[0]]:
		if !reflect.DeepEqual(cache.IndexRawResourcesByName(out.(*cache.RawDeltaResponse).Resources), snapshot2.GetResources(rsrc.EndpointType)) {
			t.Fatalf("got resources %v, want %v", out.(*cache.RawDeltaResponse).Resources, snapshot2.GetResources(rsrc.EndpointType))
		}
		nextVersionMap := out.GetNextVersionMap()

		// make sure the version maps are different since we no longer are tracking any endpoint resources
		if reflect.DeepEqual(versionMap[testTypes[0]], nextVersionMap) {
			t.Fatalf("versionMap for the endpoint resource type did not change, received: %v, instead of an emtpy map", nextVersionMap)
		}
	case <-time.After(time.Second):
		t.Fatal("failed to receive snapshot response")
	}
}

func TestConcurrentSetDeltaWatch(t *testing.T) {
	c := cache.NewSnapshotCache(false, group{}, logger{t: t})
	for i := 0; i < 50; i++ {
		version := fmt.Sprintf("v%d", i)
		func(i int) {
			t.Run(fmt.Sprintf("worker%d", i), func(t *testing.T) {
				t.Parallel()
				id := fmt.Sprintf("%d", i%2)
				var cancel func()
				if i < 25 {
					snap := cache.Snapshot{}
					snap.Resources[types.Endpoint] = cache.NewResources(version, []types.Resource{resource.MakeEndpoint(clusterName, uint32(i))})
					c.SetSnapshot(id, snap)
				} else {
					if cancel != nil {
						cancel()
					}

					_, cancel = c.CreateDeltaWatch(&discovery.DeltaDiscoveryRequest{
						Node: &core.Node{
							Id: id,
						},
						TypeUrl:                rsrc.EndpointType,
						ResourceNamesSubscribe: []string{clusterName},
					}, &stream.StreamState{IsWildcard: true, ResourceVersions: nil})
				}
			})
		}(i)
	}
}

func TestSnapshotCacheDeltaWatchCancel(t *testing.T) {
	c := cache.NewSnapshotCache(true, group{}, logger{t: t})
	for _, typ := range testTypes {
		_, cancel := c.CreateDeltaWatch(&discovery.DeltaDiscoveryRequest{
			Node: &core.Node{
				Id: key,
			},
			TypeUrl:                typ,
			ResourceNamesSubscribe: names[typ],
		}, &stream.StreamState{IsWildcard: true, ResourceVersions: nil})

		// Cancel the watch
		cancel()
	}
	// c.GetStatusKeys() should return at least 1 because we register a node ID with the above watch creations
	if keys := c.GetStatusKeys(); len(keys) == 0 {
		t.Errorf("expected to see a status info registered for watch, saw %d entries", len(keys))
	}

	for _, typ := range testTypes {
		if count := c.GetStatusInfo(key).GetNumDeltaWatches(); count > 0 {
			t.Errorf("watches should be released for %s", typ)
		}
	}

	if s := c.GetStatusInfo("missing"); s != nil {
		t.Errorf("should not return a status for unknown key: got %#v", s)
	}
}
