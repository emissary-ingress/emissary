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
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	rsrc "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/stream/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/test/resource/v3"
)

type group struct{}

const (
	key = "node"
)

func (group) ID(node *core.Node) string {
	if node != nil {
		return node.Id
	}
	return key
}

var (
	version  = "x"
	version2 = "y"

	snapshot, _ = cache.NewSnapshot(version, map[rsrc.Type][]types.Resource{
		rsrc.EndpointType:        {testEndpoint},
		rsrc.ClusterType:         {testCluster},
		rsrc.RouteType:           {testRoute},
		rsrc.ListenerType:        {testListener},
		rsrc.RuntimeType:         {testRuntime},
		rsrc.SecretType:          {testSecret[0]},
		rsrc.ExtensionConfigType: {testExtensionConfig},
	})

	ttl                = 2 * time.Second
	snapshotWithTTL, _ = cache.NewSnapshotWithTTLs(version, map[rsrc.Type][]types.ResourceWithTTL{
		rsrc.EndpointType:        {{Resource: testEndpoint, TTL: &ttl}},
		rsrc.ClusterType:         {{Resource: testCluster}},
		rsrc.RouteType:           {{Resource: testRoute}},
		rsrc.ListenerType:        {{Resource: testListener}},
		rsrc.RuntimeType:         {{Resource: testRuntime}},
		rsrc.SecretType:          {{Resource: testSecret[0]}},
		rsrc.ExtensionConfigType: {{Resource: testExtensionConfig}},
	})

	names = map[string][]string{
		rsrc.EndpointType: {clusterName},
		rsrc.ClusterType:  nil,
		rsrc.RouteType:    {routeName},
		rsrc.ListenerType: nil,
		rsrc.RuntimeType:  nil,
	}

	testTypes = []string{
		rsrc.EndpointType,
		rsrc.ClusterType,
		rsrc.RouteType,
		rsrc.ListenerType,
		rsrc.RuntimeType,
	}
)

type logger struct {
	t *testing.T
}

func (log logger) Debugf(format string, args ...interface{}) { log.t.Logf(format, args...) }
func (log logger) Infof(format string, args ...interface{})  { log.t.Logf(format, args...) }
func (log logger) Warnf(format string, args ...interface{})  { log.t.Logf(format, args...) }
func (log logger) Errorf(format string, args ...interface{}) { log.t.Logf(format, args...) }

func TestSnapshotCacheWithTTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := cache.NewSnapshotCacheWithHeartbeating(ctx, true, group{}, logger{t: t}, time.Second)

	if _, err := c.GetSnapshot(key); err == nil {
		t.Errorf("unexpected snapshot found for key %q", key)
	}

	if err := c.SetSnapshot(context.Background(), key, snapshotWithTTL); err != nil {
		t.Fatal(err)
	}

	snap, err := c.GetSnapshot(key)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snap, snapshotWithTTL) {
		t.Errorf("expect snapshot: %v, got: %v", snapshotWithTTL, snap)
	}

	wg := sync.WaitGroup{}
	// All the resources should respond immediately when version is not up to date.
	streamState := stream.NewStreamState(false, map[string]string{})
	for _, typ := range testTypes {
		wg.Add(1)
		t.Run(typ, func(t *testing.T) {
			defer wg.Done()
			value := make(chan cache.Response, 1)
			c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: typ, ResourceNames: names[typ]}, streamState, value)
			select {
			case out := <-value:
				if gotVersion, _ := out.GetVersion(); gotVersion != version {
					t.Errorf("got version %q, want %q", gotVersion, version)
				}
				if !reflect.DeepEqual(cache.IndexResourcesByName(out.(*cache.RawResponse).Resources), snapshotWithTTL.GetResourcesAndTTL(typ)) {
					t.Errorf("get resources %v, want %v", out.(*cache.RawResponse).Resources, snapshotWithTTL.GetResourcesAndTTL(typ))
				}
				// Update streamState
				streamState.SetKnownResourceNamesAsList(typ, out.GetRequest().GetResourceNames())
			case <-time.After(2 * time.Second):
				t.Errorf("failed to receive snapshot response")
			}
		})
	}
	wg.Wait()

	// Once everything is up to date, only the TTL'd resource should send out updates.
	wg = sync.WaitGroup{}
	updatesByType := map[string]int{}
	for _, typ := range testTypes {
		wg.Add(1)
		go func(typ string) {
			defer wg.Done()

			end := time.After(5 * time.Second)
			for {
				value := make(chan cache.Response, 1)
				cancel := c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: typ, ResourceNames: names[typ], VersionInfo: version},
					streamState, value)

				select {
				case out := <-value:
					if gotVersion, _ := out.GetVersion(); gotVersion != version {
						t.Errorf("got version %q, want %q", gotVersion, version)
					}
					if !reflect.DeepEqual(cache.IndexResourcesByName(out.(*cache.RawResponse).Resources), snapshotWithTTL.GetResourcesAndTTL(typ)) {
						t.Errorf("get resources %v, want %v", out.(*cache.RawResponse).Resources, snapshotWithTTL.GetResources(typ))
					}

					if !reflect.DeepEqual(cache.IndexResourcesByName(out.(*cache.RawResponse).Resources), snapshotWithTTL.GetResourcesAndTTL(typ)) {
						t.Errorf("get resources %v, want %v", out.(*cache.RawResponse).Resources, snapshotWithTTL.GetResources(typ))
					}

					updatesByType[typ]++

					streamState.SetKnownResourceNamesAsList(typ, out.GetRequest().ResourceNames)
				case <-end:
					cancel()
					return
				}
			}
		}(typ)
	}

	wg.Wait()

	if len(updatesByType) != 1 {
		t.Errorf("expected to only receive updates for TTL'd type, got %v", updatesByType)
	}
	// Avoid an exact match on number of triggers to avoid this being flaky.
	if updatesByType[rsrc.EndpointType] < 2 {
		t.Errorf("expected at least two TTL updates for endpoints, got %d", updatesByType[rsrc.EndpointType])
	}
}

func TestSnapshotCache(t *testing.T) {
	c := cache.NewSnapshotCache(true, group{}, logger{t: t})

	if _, err := c.GetSnapshot(key); err == nil {
		t.Errorf("unexpected snapshot found for key %q", key)
	}

	if err := c.SetSnapshot(context.Background(), key, snapshot); err != nil {
		t.Fatal(err)
	}

	snap, err := c.GetSnapshot(key)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snap, snapshot) {
		t.Errorf("expect snapshot: %v, got: %v", snapshot, snap)
	}

	// try to get endpoints with incorrect list of names
	// should not receive response
	value := make(chan cache.Response, 1)
	streamState := stream.NewStreamState(false, map[string]string{})
	c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: rsrc.EndpointType, ResourceNames: []string{"none"}},
		streamState, value)
	select {
	case out := <-value:
		t.Errorf("watch for endpoints and mismatched names => got %v, want none", out)
	case <-time.After(time.Second / 4):
	}

	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			value := make(chan cache.Response, 1)
			streamState := stream.NewStreamState(false, map[string]string{})
			c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: typ, ResourceNames: names[typ]},
				streamState, value)
			select {
			case out := <-value:
				if gotVersion, _ := out.GetVersion(); gotVersion != version {
					t.Errorf("got version %q, want %q", gotVersion, version)
				}
				if !reflect.DeepEqual(cache.IndexResourcesByName(out.(*cache.RawResponse).Resources), snapshot.GetResourcesAndTTL(typ)) {
					t.Errorf("get resources %v, want %v", out.(*cache.RawResponse).Resources, snapshot.GetResourcesAndTTL(typ))
				}
			case <-time.After(time.Second):
				t.Fatal("failed to receive snapshot response")
			}
		})
	}
}

func TestSnapshotCacheFetch(t *testing.T) {
	c := cache.NewSnapshotCache(true, group{}, logger{t: t})
	if err := c.SetSnapshot(context.Background(), key, snapshot); err != nil {
		t.Fatal(err)
	}

	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			resp, err := c.Fetch(context.Background(), &discovery.DiscoveryRequest{TypeUrl: typ, ResourceNames: names[typ]})
			if err != nil || resp == nil {
				t.Fatal("unexpected error or null response")
			}
			if gotVersion, _ := resp.GetVersion(); gotVersion != version {
				t.Errorf("got version %q, want %q", gotVersion, version)
			}
		})
	}

	// no response for missing snapshot
	if resp, err := c.Fetch(context.Background(),
		&discovery.DiscoveryRequest{TypeUrl: rsrc.ClusterType, Node: &core.Node{Id: "oof"}}); resp != nil || err == nil {
		t.Errorf("missing snapshot: response is not nil %v", resp)
	}

	// no response for latest version
	if resp, err := c.Fetch(context.Background(),
		&discovery.DiscoveryRequest{TypeUrl: rsrc.ClusterType, VersionInfo: version}); resp != nil || err == nil {
		t.Errorf("latest version: response is not nil %v", resp)
	}
}

func TestSnapshotCacheWatch(t *testing.T) {
	c := cache.NewSnapshotCache(true, group{}, logger{t: t})
	watches := make(map[string]chan cache.Response)
	streamState := stream.NewStreamState(false, map[string]string{})
	for _, typ := range testTypes {
		watches[typ] = make(chan cache.Response, 1)
		c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: typ, ResourceNames: names[typ]}, streamState, watches[typ])
	}
	if err := c.SetSnapshot(context.Background(), key, snapshot); err != nil {
		t.Fatal(err)
	}
	for _, typ := range testTypes {
		t.Run(typ, func(t *testing.T) {
			select {
			case out := <-watches[typ]:
				if gotVersion, _ := out.GetVersion(); gotVersion != version {
					t.Errorf("got version %q, want %q", gotVersion, version)
				}
				if !reflect.DeepEqual(cache.IndexResourcesByName(out.(*cache.RawResponse).Resources), snapshot.GetResourcesAndTTL(typ)) {
					t.Errorf("get resources %v, want %v", out.(*cache.RawResponse).Resources, snapshot.GetResourcesAndTTL(typ))
				}
				streamState.SetKnownResourceNamesAsList(typ, out.GetRequest().GetResourceNames())
			case <-time.After(time.Second):
				t.Fatal("failed to receive snapshot response")
			}
		})
	}

	// open new watches with the latest version
	for _, typ := range testTypes {
		watches[typ] = make(chan cache.Response, 1)
		c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: typ, ResourceNames: names[typ], VersionInfo: version},
			streamState, watches[typ])
	}
	if count := c.GetStatusInfo(key).GetNumWatches(); count != len(testTypes) {
		t.Errorf("watches should be created for the latest version: %d", count)
	}

	// set partially-versioned snapshot
	snapshot2 := snapshot
	snapshot2.Resources[types.Endpoint] = cache.NewResources(version2, []types.Resource{resource.MakeEndpoint(clusterName, 9090)})
	if err := c.SetSnapshot(context.Background(), key, snapshot2); err != nil {
		t.Fatal(err)
	}
	if count := c.GetStatusInfo(key).GetNumWatches(); count != len(testTypes)-1 {
		t.Errorf("watches should be preserved for all but one: %d", count)
	}

	// validate response for endpoints
	select {
	case out := <-watches[rsrc.EndpointType]:
		if gotVersion, _ := out.GetVersion(); gotVersion != version2 {
			t.Errorf("got version %q, want %q", gotVersion, version2)
		}
		if !reflect.DeepEqual(cache.IndexResourcesByName(out.(*cache.RawResponse).Resources), snapshot2.Resources[types.Endpoint].Items) {
			t.Errorf("got resources %v, want %v", out.(*cache.RawResponse).Resources, snapshot2.Resources[types.Endpoint].Items)
		}
	case <-time.After(time.Second):
		t.Fatal("failed to receive snapshot response")
	}
}

func TestConcurrentSetWatch(t *testing.T) {
	c := cache.NewSnapshotCache(false, group{}, logger{t: t})
	for i := 0; i < 50; i++ {
		t.Run(fmt.Sprintf("worker%d", i), func(t *testing.T) {
			t.Parallel()
			id := fmt.Sprintf("%d", i%2)
			value := make(chan cache.Response, 1)
			if i < 25 {
				snap := cache.Snapshot{}
				snap.Resources[types.Endpoint] = cache.NewResources(fmt.Sprintf("v%d", i), []types.Resource{resource.MakeEndpoint(clusterName, uint32(i))})
				if err := c.SetSnapshot(context.Background(), id, snap); err != nil {
					t.Fatalf("failed to set snapshot %q: %s", id, err)
				}
			} else {
				streamState := stream.NewStreamState(false, map[string]string{})
				cancel := c.CreateWatch(&discovery.DiscoveryRequest{
					Node:    &core.Node{Id: id},
					TypeUrl: rsrc.EndpointType,
				}, streamState, value)

				defer cancel()
			}
		})
	}
}

func TestSnapshotCacheWatchCancel(t *testing.T) {
	c := cache.NewSnapshotCache(true, group{}, logger{t: t})
	streamState := stream.NewStreamState(false, map[string]string{})
	for _, typ := range testTypes {
		value := make(chan cache.Response, 1)
		cancel := c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: typ, ResourceNames: names[typ]}, streamState, value)
		cancel()
	}
	// should be status info for the node
	if keys := c.GetStatusKeys(); len(keys) == 0 {
		t.Error("got 0, want status info for the node")
	}

	for _, typ := range testTypes {
		if count := c.GetStatusInfo(key).GetNumWatches(); count > 0 {
			t.Errorf("watches should be released for %s", typ)
		}
	}

	if empty := c.GetStatusInfo("missing"); empty != nil {
		t.Errorf("should not return a status for unknown key: got %#v", empty)
	}
}

func TestSnapshotCacheWatchTimeout(t *testing.T) {
	c := cache.NewSnapshotCache(true, group{}, logger{t: t})

	// Create a non-buffered channel that will block sends.
	watchCh := make(chan cache.Response)
	streamState := stream.NewStreamState(false, map[string]string{})
	c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: rsrc.EndpointType, ResourceNames: names[rsrc.EndpointType]},
		streamState, watchCh)

	// The first time we set the snapshot without consuming from the blocking channel, so this should time out.
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	err := c.SetSnapshot(ctx, key, snapshot)
	assert.EqualError(t, err, context.Canceled.Error())

	// Now reset the snapshot with a consuming channel. This verifies that if setting the snapshot fails,
	// we can retry by setting the same snapshot. In other words, we keep the watch open even if we failed
	// to respond to it within the deadline.
	watchTriggeredCh := make(chan cache.Response)
	go func() {
		response := <-watchCh
		watchTriggeredCh <- response
		close(watchTriggeredCh)
	}()

	err = c.SetSnapshot(context.WithValue(context.Background(), testKey{}, "bar"), key, snapshot)
	assert.NoError(t, err)

	// The channel should get closed due to the watch trigger.
	select {
	case response := <-watchTriggeredCh:
		// Verify that we pass the context through.
		assert.Equal(t, response.GetContext().Value(testKey{}), "bar")
	case <-time.After(time.Second):
		t.Fatalf("timed out")
	}
}

func TestSnapshotCreateWatchWithResourcePreviouslyNotRequested(t *testing.T) {
	clusterName2 := "clusterName2"
	routeName2 := "routeName2"
	listenerName2 := "listenerName2"
	c := cache.NewSnapshotCache(false, group{}, logger{t: t})

	snapshot2, _ := cache.NewSnapshot(version, map[rsrc.Type][]types.Resource{
		rsrc.EndpointType:        {testEndpoint, resource.MakeEndpoint(clusterName2, 8080)},
		rsrc.ClusterType:         {testCluster, resource.MakeCluster(resource.Ads, clusterName2)},
		rsrc.RouteType:           {testRoute, resource.MakeRoute(routeName2, clusterName2)},
		rsrc.ListenerType:        {testListener, resource.MakeRouteHTTPListener(resource.Ads, listenerName2, 80, routeName2)},
		rsrc.RuntimeType:         {},
		rsrc.SecretType:          {},
		rsrc.ExtensionConfigType: {},
	})
	if err := c.SetSnapshot(context.Background(), key, snapshot2); err != nil {
		t.Fatal(err)
	}
	watch := make(chan cache.Response)

	// Request resource with name=ClusterName
	go func() {
		c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: rsrc.EndpointType, ResourceNames: []string{clusterName}},
			stream.NewStreamState(false, map[string]string{}), watch)
	}()

	select {
	case out := <-watch:
		if gotVersion, _ := out.GetVersion(); gotVersion != version {
			t.Errorf("got version %q, want %q", gotVersion, version)
		}
		want := map[string]types.ResourceWithTTL{clusterName: snapshot2.Resources[types.Endpoint].Items[clusterName]}
		if !reflect.DeepEqual(cache.IndexResourcesByName(out.(*cache.RawResponse).Resources), want) {
			t.Errorf("got resources %v, want %v", out.(*cache.RawResponse).Resources, want)
		}
	case <-time.After(time.Second):
		t.Fatal("failed to receive snapshot response")
	}

	// Request additional resource with name=clusterName2 for same version
	go func() {
		state := stream.NewStreamState(false, map[string]string{})
		state.SetKnownResourceNames(rsrc.EndpointType, map[string]struct{}{clusterName: {}})
		c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: rsrc.EndpointType, VersionInfo: version,
			ResourceNames: []string{clusterName, clusterName2}}, state, watch)
	}()

	select {
	case out := <-watch:
		if gotVersion, _ := out.GetVersion(); gotVersion != version {
			t.Errorf("got version %q, want %q", gotVersion, version)
		}
		if !reflect.DeepEqual(cache.IndexResourcesByName(out.(*cache.RawResponse).Resources), snapshot2.Resources[types.Endpoint].Items) {
			t.Errorf("got resources %v, want %v", out.(*cache.RawResponse).Resources, snapshot2.Resources[types.Endpoint].Items)
		}
	case <-time.After(time.Second):
		t.Fatal("failed to receive snapshot response")
	}

	// Repeat request for with same version and make sure a watch is created
	state := stream.NewStreamState(false, map[string]string{})
	state.SetKnownResourceNames(rsrc.EndpointType, map[string]struct{}{clusterName: {}, clusterName2: {}})
	if cancel := c.CreateWatch(&discovery.DiscoveryRequest{TypeUrl: rsrc.EndpointType, VersionInfo: version,
		ResourceNames: []string{clusterName, clusterName2}}, state, watch); cancel == nil {
		t.Fatal("Should create a watch")
	} else {
		cancel()
	}
}

func TestSnapshotClear(t *testing.T) {
	c := cache.NewSnapshotCache(true, group{}, logger{t: t})
	if err := c.SetSnapshot(context.Background(), key, snapshot); err != nil {
		t.Fatal(err)
	}
	c.ClearSnapshot(key)
	if empty := c.GetStatusInfo(key); empty != nil {
		t.Errorf("cache should be cleared")
	}
	if keys := c.GetStatusKeys(); len(keys) != 0 {
		t.Errorf("keys should be empty")
	}
}
