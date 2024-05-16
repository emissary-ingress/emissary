// Copyright 2020 Envoyproxy Authors
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

package cache

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	endpoint "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/endpoint/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/stream/v3"
)

const (
	testType = "google.protobuf.StringValue"
)

func testResource(s string) types.Resource {
	return wrapperspb.String(s)
}

func verifyResponse(t *testing.T, ch <-chan Response, version string, num int) {
	t.Helper()
	r := <-ch
	if r.GetRequest().GetTypeUrl() != testType {
		t.Errorf("unexpected empty request type URL: %q", r.GetRequest().GetTypeUrl())
	}
	if r.GetContext() == nil {
		t.Errorf("unexpected empty response context")
	}
	out, err := r.GetDiscoveryResponse()
	if err != nil {
		t.Fatal(err)
	}
	if out.GetVersionInfo() == "" {
		t.Error("unexpected response empty version")
	}
	if n := len(out.GetResources()); n != num {
		t.Errorf("unexpected number of responses: got %d, want %d", n, num)
	}
	if version != "" && out.GetVersionInfo() != version {
		t.Errorf("unexpected version: got %q, want %q", out.GetVersionInfo(), version)
	}
	if out.GetTypeUrl() != testType {
		t.Errorf("unexpected type URL: %q", out.GetTypeUrl())
	}
}

type resourceInfo struct {
	name    string
	version string
}

func validateDeltaResponse(t *testing.T, resp DeltaResponse, resources []resourceInfo, deleted []string) {
	t.Helper()

	if resp.GetDeltaRequest().GetTypeUrl() != testType {
		t.Errorf("unexpected empty request type URL: %q", resp.GetDeltaRequest().GetTypeUrl())
	}
	out, err := resp.GetDeltaDiscoveryResponse()
	if err != nil {
		t.Fatal(err)
	}
	if len(out.GetResources()) != len(resources) {
		t.Errorf("unexpected number of responses: got %d, want %d", len(out.GetResources()), len(resources))
	}
	for _, r := range resources {
		found := false
		for _, r1 := range out.GetResources() {
			if r1.GetName() == r.name && r1.GetVersion() == r.version {
				found = true
				break
			} else if r1.GetName() == r.name {
				t.Errorf("unexpected version for resource %q: got %q, want %q", r.name, r1.GetVersion(), r.version)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("resource with name %q not found in response", r.name)
		}
	}
	if out.GetTypeUrl() != testType {
		t.Errorf("unexpected type URL: %q", out.GetTypeUrl())
	}
	if len(out.GetRemovedResources()) != len(deleted) {
		t.Errorf("unexpected number of removed resurces: got %d, want %d", len(out.GetRemovedResources()), len(deleted))
	}
	for _, r := range deleted {
		found := false
		for _, rr := range out.GetRemovedResources() {
			if r == rr {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected resource %s to be deleted", r)
		}
	}
}

func verifyDeltaResponse(t *testing.T, ch <-chan DeltaResponse, resources []resourceInfo, deleted []string) {
	t.Helper()
	var r DeltaResponse
	select {
	case r = <-ch:
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for delta response")
		return
	}
	validateDeltaResponse(t, r, resources, deleted)
}

func checkWatchCount(t *testing.T, c *LinearCache, name string, count int) {
	t.Helper()
	if i := c.NumWatches(name); i != count {
		t.Errorf("unexpected number of watches for %q: got %d, want %d", name, i, count)
	}
}

func checkDeltaWatchCount(t *testing.T, c *LinearCache, count int) {
	t.Helper()
	if i := c.NumDeltaWatches(); i != count {
		t.Errorf("unexpected number of delta watches: got %d, want %d", i, count)
	}
}

func checkVersionMapNotSet(t *testing.T, c *LinearCache) {
	t.Helper()
	if c.versionMap != nil {
		t.Errorf("version map is set on the cache with %d elements", len(c.versionMap))
	}
}

func checkVersionMapSet(t *testing.T, c *LinearCache) {
	t.Helper()
	if c.versionMap == nil {
		t.Errorf("version map is not set on the cache")
	} else if len(c.versionMap) != len(c.resources) {
		t.Errorf("version map has the wrong number of elements: %d instead of %d expected", len(c.versionMap), len(c.resources))
	}
}

func mustBlock(t *testing.T, w <-chan Response) {
	select {
	case <-w:
		t.Error("watch must block")
	default:
	}
}

func mustBlockDelta(t *testing.T, w <-chan DeltaResponse) {
	select {
	case <-w:
		t.Error("watch must block")
	default:
	}
}

func hashResource(t *testing.T, resource types.Resource) string {
	marshaledResource, err := MarshalResource(resource)
	if err != nil {
		t.Fatal(err)
	}
	v := HashResource(marshaledResource)
	if v == "" {
		t.Fatal(errors.New("failed to build resource version"))
	}
	return v
}

func createWildcardDeltaWatch(c *LinearCache, w chan DeltaResponse) {
	state := stream.NewStreamState(true, nil)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	resp := <-w
	state.SetResourceVersions(resp.GetNextVersionMap())
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w) // Ensure the watch is set properly with cache values
}

func TestLinearInitialResources(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType, WithInitialResources(map[string]types.Resource{"a": testResource("a"), "b": testResource("b")}))
	w := make(chan Response, 1)
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType}, streamState, w)
	verifyResponse(t, w, "0", 1)
	c.CreateWatch(&Request{TypeUrl: testType}, streamState, w)
	verifyResponse(t, w, "0", 2)
	checkVersionMapNotSet(t, c)
}

func TestLinearCornerCases(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType)
	err := c.UpdateResource("a", nil)
	if err == nil {
		t.Error("expected error on nil resource")
	}
	// create an incorrect type URL request
	w := make(chan Response, 1)
	c.CreateWatch(&Request{TypeUrl: "test"}, streamState, w)
	select {
	case r := <-w:
		if r != nil {
			t.Error("response should be nil")
		}
	default:
		t.Error("should receive nil response")
	}
}

func TestLinearBasic(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType)

	// Create watches before a resource is ready
	w1 := make(chan Response, 1)
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"}, streamState, w1)
	mustBlock(t, w1)
	checkVersionMapNotSet(t, c)

	w := make(chan Response, 1)
	c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 2)
	checkWatchCount(t, c, "b", 1)
	require.NoError(t, c.UpdateResource("a", testResource("a")))
	checkWatchCount(t, c, "a", 0)
	checkWatchCount(t, c, "b", 0)
	verifyResponse(t, w1, "1", 1)
	verifyResponse(t, w, "1", 1)

	// Request again, should get same response
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	checkWatchCount(t, c, "a", 0)
	verifyResponse(t, w, "1", 1)
	c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	checkWatchCount(t, c, "a", 0)
	verifyResponse(t, w, "1", 1)

	// Add another element and update the first, response should be different
	require.NoError(t, c.UpdateResource("b", testResource("b")))
	require.NoError(t, c.UpdateResource("a", testResource("aa")))
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	verifyResponse(t, w, "3", 1)
	c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	verifyResponse(t, w, "3", 2)
	// Ensure the version map was not created as we only ever used stow watches
	checkVersionMapNotSet(t, c)
}

func TestLinearSetResources(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType)

	// Create new resources
	w1 := make(chan Response, 1)
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"}, streamState, w1)
	mustBlock(t, w1)
	w2 := make(chan Response, 1)
	c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"}, streamState, w2)
	mustBlock(t, w2)
	c.SetResources(map[string]types.Resource{
		"a": testResource("a"),
		"b": testResource("b"),
	})
	verifyResponse(t, w1, "1", 1)
	verifyResponse(t, w2, "1", 2) // the version was only incremented once for all resources

	// Add another element and update the first, response should be different
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "1"}, streamState, w1)
	mustBlock(t, w1)
	c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "1"}, streamState, w2)
	mustBlock(t, w2)
	c.SetResources(map[string]types.Resource{
		"a": testResource("aa"),
		"b": testResource("b"),
		"c": testResource("c"),
	})
	verifyResponse(t, w1, "2", 1)
	verifyResponse(t, w2, "2", 3)

	// Delete resource
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "2"}, streamState, w1)
	mustBlock(t, w1)
	c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "2"}, streamState, w2)
	mustBlock(t, w2)
	c.SetResources(map[string]types.Resource{
		"b": testResource("b"),
		"c": testResource("c"),
	})
	verifyResponse(t, w1, "", 0) // removing a resource from the set triggers existing watches for deleted resources
	verifyResponse(t, w2, "3", 2)
}

func TestLinearGetResources(t *testing.T) {
	c := NewLinearCache(testType)

	expectedResources := map[string]types.Resource{
		"a": testResource("a"),
		"b": testResource("b"),
	}

	c.SetResources(expectedResources)

	resources := c.GetResources()

	if !reflect.DeepEqual(expectedResources, resources) {
		t.Errorf("resources are not equal. got: %v want: %v", resources, expectedResources)
	}
}

func TestLinearVersionPrefix(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType, WithVersionPrefix("instance1-"))

	w := make(chan Response, 1)
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	verifyResponse(t, w, "instance1-0", 0)

	require.NoError(t, c.UpdateResource("a", testResource("a")))
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	verifyResponse(t, w, "instance1-1", 1)

	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "instance1-1"}, streamState, w)
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 1)
}

func TestLinearDeletion(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType, WithInitialResources(map[string]types.Resource{"a": testResource("a"), "b": testResource("b")}))
	w := make(chan Response, 1)
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 1)
	require.NoError(t, c.DeleteResource("a"))
	verifyResponse(t, w, "1", 0)
	checkWatchCount(t, c, "a", 0)
	c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	verifyResponse(t, w, "1", 1)
	checkWatchCount(t, c, "b", 0)
	require.NoError(t, c.DeleteResource("b"))
	c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "1"}, streamState, w)
	verifyResponse(t, w, "2", 0)
	checkWatchCount(t, c, "b", 0)
}

func TestLinearWatchTwo(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType, WithInitialResources(map[string]types.Resource{"a": testResource("a"), "b": testResource("b")}))
	w := make(chan Response, 1)
	c.CreateWatch(&Request{ResourceNames: []string{"a", "b"}, TypeUrl: testType, VersionInfo: "0"}, streamState, w)
	mustBlock(t, w)
	w1 := make(chan Response, 1)
	c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"}, streamState, w1)
	mustBlock(t, w1)
	require.NoError(t, c.UpdateResource("a", testResource("aa")))
	// should only get the modified resource
	verifyResponse(t, w, "1", 1)
	verifyResponse(t, w1, "1", 2)
}

func TestLinearCancel(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType)
	require.NoError(t, c.UpdateResource("a", testResource("a")))

	// cancel watch-all
	w := make(chan Response, 1)
	cancel := c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "1"}, streamState, w)
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 1)
	cancel()
	checkWatchCount(t, c, "a", 0)

	// cancel watch for "a"
	cancel = c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "1"}, streamState, w)
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 1)
	cancel()
	checkWatchCount(t, c, "a", 0)

	// open four watches for "a" and "b" and two for all, cancel one of each, make sure the second one is unaffected
	w2 := make(chan Response, 1)
	w3 := make(chan Response, 1)
	w4 := make(chan Response, 1)
	cancel = c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "1"}, streamState, w)
	cancel2 := c.CreateWatch(&Request{ResourceNames: []string{"b"}, TypeUrl: testType, VersionInfo: "1"}, streamState, w2)
	cancel3 := c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "1"}, streamState, w3)
	cancel4 := c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "1"}, streamState, w4)
	mustBlock(t, w)
	mustBlock(t, w2)
	mustBlock(t, w3)
	mustBlock(t, w4)
	checkWatchCount(t, c, "a", 3)
	checkWatchCount(t, c, "b", 3)
	cancel()
	checkWatchCount(t, c, "a", 2)
	checkWatchCount(t, c, "b", 3)
	cancel3()
	checkWatchCount(t, c, "a", 1)
	checkWatchCount(t, c, "b", 2)
	cancel2()
	cancel4()
	checkWatchCount(t, c, "a", 0)
	checkWatchCount(t, c, "b", 0)
}

// TODO(mattklein123): This test requires GOMAXPROCS or -parallel >= 100. This should be
// rewritten to not require that. This is not the case in the GH actions environment.
func TestLinearConcurrentSetWatch(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType)
	n := 50
	for i := 0; i < 2*n; i++ {
		func(i int) {
			t.Run(fmt.Sprintf("worker%d", i), func(t *testing.T) {
				t.Parallel()
				id := fmt.Sprintf("%d", i)
				if i%2 == 0 {
					t.Logf("update resource %q", id)
					require.NoError(t, c.UpdateResource(id, testResource(id)))
				} else {
					id2 := fmt.Sprintf("%d", i-1)
					t.Logf("request resources %q and %q", id, id2)
					value := make(chan Response, 1)
					c.CreateWatch(&Request{
						// Only expect one to become stale
						ResourceNames: []string{id, id2},
						VersionInfo:   "0",
						TypeUrl:       testType,
					}, streamState, value)
					// wait until all updates apply
					verifyResponse(t, value, "", 1)
				}
			})
		}(i)
	}
}

func TestLinearDeltaWildcard(t *testing.T) {
	c := NewLinearCache(testType)
	state1 := stream.NewStreamState(true, map[string]string{})
	w1 := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state1, w1)
	mustBlockDelta(t, w1)
	state2 := stream.NewStreamState(true, map[string]string{})
	w2 := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state2, w2)
	mustBlockDelta(t, w1)
	checkDeltaWatchCount(t, c, 2)

	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	hash := hashResource(t, a)
	err := c.UpdateResource("a", a)
	require.NoError(t, err)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w1, []resourceInfo{{"a", hash}}, nil)
	verifyDeltaResponse(t, w2, []resourceInfo{{"a", hash}}, nil)
}

func TestLinearDeltaExistingResources(t *testing.T) {
	c := NewLinearCache(testType)
	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	hashA := hashResource(t, a)
	err := c.UpdateResource("a", a)
	require.NoError(t, err)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err = c.UpdateResource("b", b)
	require.NoError(t, err)

	state := stream.NewStreamState(false, nil)
	state.SetSubscribedResourceNames(map[string]struct{}{"b": {}, "c": {}}) // watching b and c - not interested in a
	w := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}}, []string{})

	state = stream.NewStreamState(false, nil)
	state.SetSubscribedResourceNames(map[string]struct{}{"a": {}, "b": {}})
	w = make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}, {"a", hashA}}, nil)
}

func TestLinearDeltaInitialResourcesVersionSet(t *testing.T) {
	c := NewLinearCache(testType)
	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	hashA := hashResource(t, a)
	err := c.UpdateResource("a", a)
	require.NoError(t, err)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err = c.UpdateResource("b", b)
	require.NoError(t, err)

	state := stream.NewStreamState(false, map[string]string{"b": hashB})
	state.SetSubscribedResourceNames(map[string]struct{}{"a": {}, "b": {}})
	w := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"a", hashA}}, nil) // b is up to date and shouldn't be returned

	state = stream.NewStreamState(false, map[string]string{"a": hashA, "b": hashB})
	state.SetSubscribedResourceNames(map[string]struct{}{"a": {}, "b": {}})
	w = make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)
	b = &endpoint.ClusterLoadAssignment{ClusterName: "b", Endpoints: []*endpoint.LocalityLbEndpoints{{Priority: 10}}} // new version of b
	hashB = hashResource(t, b)
	err = c.UpdateResource("b", b)
	require.NoError(t, err)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}}, nil)
}

func TestLinearDeltaResourceUpdate(t *testing.T) {
	c := NewLinearCache(testType)
	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	hashA := hashResource(t, a)
	err := c.UpdateResource("a", a)
	require.NoError(t, err)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err = c.UpdateResource("b", b)
	require.NoError(t, err)
	// There is currently no delta watch
	checkVersionMapNotSet(t, c)

	state := stream.NewStreamState(false, nil)
	state.SetSubscribedResourceNames(map[string]struct{}{"a": {}, "b": {}})
	w := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}, {"a", hashA}}, nil)
	checkVersionMapSet(t, c)

	state = stream.NewStreamState(false, map[string]string{"a": hashA, "b": hashB})
	state.SetSubscribedResourceNames(map[string]struct{}{"a": {}, "b": {}})
	w = make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)

	a = &endpoint.ClusterLoadAssignment{ClusterName: "a", Endpoints: []*endpoint.LocalityLbEndpoints{ // resource update
		{Priority: 10},
	}}
	hashA = hashResource(t, a)
	err = c.UpdateResource("a", a)
	require.NoError(t, err)
	verifyDeltaResponse(t, w, []resourceInfo{{"a", hashA}}, nil)
	checkVersionMapSet(t, c)
}

func TestLinearDeltaResourceDelete(t *testing.T) {
	c := NewLinearCache(testType)
	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	hashA := hashResource(t, a)
	err := c.UpdateResource("a", a)
	require.NoError(t, err)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err = c.UpdateResource("b", b)
	require.NoError(t, err)

	state := stream.NewStreamState(false, nil)
	state.SetSubscribedResourceNames(map[string]struct{}{"a": {}, "b": {}})
	w := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}, {"a", hashA}}, nil)

	state = stream.NewStreamState(false, map[string]string{"a": hashA, "b": hashB})
	state.SetSubscribedResourceNames(map[string]struct{}{"a": {}, "b": {}})
	w = make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)

	a = &endpoint.ClusterLoadAssignment{ClusterName: "a", Endpoints: []*endpoint.LocalityLbEndpoints{ // resource update
		{Priority: 10},
	}}
	hashA = hashResource(t, a)
	c.SetResources(map[string]types.Resource{"a": a})
	verifyDeltaResponse(t, w, []resourceInfo{{"a", hashA}}, []string{"b"})
}

func TestLinearDeltaMultiResourceUpdates(t *testing.T) {
	c := NewLinearCache(testType)

	state := stream.NewStreamState(false, nil)
	state.SetSubscribedResourceNames(map[string]struct{}{"a": {}, "b": {}})
	w := make(chan DeltaResponse, 1)
	checkVersionMapNotSet(t, c)
	assert.Equal(t, 0, c.NumResources())

	// Initial update
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)
	// The version map should now be created, even if empty
	checkVersionMapSet(t, c)
	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	hashA := hashResource(t, a)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err := c.UpdateResources(map[string]types.Resource{"a": a, "b": b}, nil)
	require.NoError(t, err)
	resp := <-w
	validateDeltaResponse(t, resp, []resourceInfo{{"a", hashA}, {"b", hashB}}, nil)
	checkVersionMapSet(t, c)
	assert.Equal(t, 2, c.NumResources())
	state.SetResourceVersions(resp.GetNextVersionMap())

	// Multiple updates
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)
	a = &endpoint.ClusterLoadAssignment{ClusterName: "a", Endpoints: []*endpoint.LocalityLbEndpoints{ // resource update
		{Priority: 10},
	}}
	b = &endpoint.ClusterLoadAssignment{ClusterName: "b", Endpoints: []*endpoint.LocalityLbEndpoints{ // resource update
		{Priority: 15},
	}}
	hashA = hashResource(t, a)
	hashB = hashResource(t, b)
	err = c.UpdateResources(map[string]types.Resource{"a": a, "b": b}, nil)
	require.NoError(t, err)
	resp = <-w
	validateDeltaResponse(t, resp, []resourceInfo{{"a", hashA}, {"b", hashB}}, nil)
	checkVersionMapSet(t, c)
	assert.Equal(t, 2, c.NumResources())
	state.SetResourceVersions(resp.GetNextVersionMap())

	// Update/add/delete
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)
	a = &endpoint.ClusterLoadAssignment{ClusterName: "a", Endpoints: []*endpoint.LocalityLbEndpoints{ // resource update
		{Priority: 15},
	}}
	d := &endpoint.ClusterLoadAssignment{ClusterName: "d", Endpoints: []*endpoint.LocalityLbEndpoints{}} // resource created, but not watched
	hashA = hashResource(t, a)
	err = c.UpdateResources(map[string]types.Resource{"a": a, "d": d}, []string{"b"})
	require.NoError(t, err)
	assert.Contains(t, c.resources, "d", "resource with name d not found in cache")
	assert.NotContains(t, c.resources, "b", "resource with name b was found in cache")
	resp = <-w
	validateDeltaResponse(t, resp, []resourceInfo{{"a", hashA}}, []string{"b"})
	checkVersionMapSet(t, c)
	assert.Equal(t, 2, c.NumResources())
	state.SetResourceVersions(resp.GetNextVersionMap())

	// Re-add previously deleted watched resource
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)
	b = &endpoint.ClusterLoadAssignment{ClusterName: "b", Endpoints: []*endpoint.LocalityLbEndpoints{}} // recreate watched resource
	hashB = hashResource(t, b)
	err = c.UpdateResources(map[string]types.Resource{"b": b}, []string{"d"})
	require.NoError(t, err)
	assert.Contains(t, c.resources, "b", "resource with name b not found in cache")
	assert.NotContains(t, c.resources, "d", "resource with name d was found in cache")
	resp = <-w
	validateDeltaResponse(t, resp, []resourceInfo{{"b", hashB}}, nil) // d is not watched and should not be returned
	checkVersionMapSet(t, c)
	assert.Equal(t, 2, c.NumResources())
	state.SetResourceVersions(resp.GetNextVersionMap())

	// Wildcard create/update
	createWildcardDeltaWatch(c, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)
	b = &endpoint.ClusterLoadAssignment{ClusterName: "b", Endpoints: []*endpoint.LocalityLbEndpoints{ // resource update
		{Priority: 15},
	}}
	d = &endpoint.ClusterLoadAssignment{ClusterName: "d", Endpoints: []*endpoint.LocalityLbEndpoints{}} // resource create
	hashB = hashResource(t, b)
	hashD := hashResource(t, d)
	err = c.UpdateResources(map[string]types.Resource{"b": b, "d": d}, nil)
	require.NoError(t, err)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}, {"d", hashD}}, nil)
	checkVersionMapSet(t, c)
	assert.Equal(t, 3, c.NumResources())

	// Wildcard update/delete
	createWildcardDeltaWatch(c, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)
	a = &endpoint.ClusterLoadAssignment{ClusterName: "a", Endpoints: []*endpoint.LocalityLbEndpoints{ // resource update
		{Priority: 25},
	}}
	hashA = hashResource(t, a)
	err = c.UpdateResources(map[string]types.Resource{"a": a}, []string{"d"})
	require.NoError(t, err)
	assert.NotContains(t, c.resources, "d", "resource with name d was found in cache")
	verifyDeltaResponse(t, w, []resourceInfo{{"a", hashA}}, []string{"d"})

	checkDeltaWatchCount(t, c, 0)
	// Confirm that the map is still set even though there is currently no watch
	checkVersionMapSet(t, c)
	assert.Equal(t, 2, c.NumResources())
}

func TestLinearMixedWatches(t *testing.T) {
	c := NewLinearCache(testType)
	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	err := c.UpdateResource("a", a)
	require.NoError(t, err)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err = c.UpdateResource("b", b)
	require.NoError(t, err)
	assert.Equal(t, 2, c.NumResources())

	sotwState := stream.NewStreamState(false, nil)
	w := make(chan Response, 1)
	c.CreateWatch(&Request{ResourceNames: []string{"a", "b"}, TypeUrl: testType, VersionInfo: c.getVersion()}, sotwState, w)
	mustBlock(t, w)
	checkVersionMapNotSet(t, c)

	a = &endpoint.ClusterLoadAssignment{ClusterName: "a", Endpoints: []*endpoint.LocalityLbEndpoints{ // resource update
		{Priority: 25},
	}}
	hashA := hashResource(t, a)
	err = c.UpdateResources(map[string]types.Resource{"a": a}, nil)
	require.NoError(t, err)
	// This behavior is currently invalid for cds and lds, but due to a current limitation of linear cache sotw implementation
	verifyResponse(t, w, c.getVersion(), 1)
	checkVersionMapNotSet(t, c)

	c.CreateWatch(&Request{ResourceNames: []string{"a", "b"}, TypeUrl: testType, VersionInfo: c.getVersion()}, sotwState, w)
	mustBlock(t, w)
	checkVersionMapNotSet(t, c)

	deltaState := stream.NewStreamState(false, map[string]string{"a": hashA, "b": hashB})
	deltaState.SetSubscribedResourceNames(map[string]struct{}{"a": {}, "b": {}})
	wd := make(chan DeltaResponse, 1)

	// Initial update
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, deltaState, wd)
	mustBlockDelta(t, wd)
	checkDeltaWatchCount(t, c, 1)
	checkVersionMapSet(t, c)

	err = c.UpdateResources(nil, []string{"b"})
	require.NoError(t, err)
	checkVersionMapSet(t, c)

	verifyResponse(t, w, c.getVersion(), 0)
	verifyDeltaResponse(t, wd, nil, []string{"b"})
}
