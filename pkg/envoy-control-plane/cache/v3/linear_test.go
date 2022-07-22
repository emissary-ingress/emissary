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
	if r.GetRequest().TypeUrl != testType {
		t.Errorf("unexpected empty request type URL: %q", r.GetRequest().TypeUrl)
	}
	out, err := r.GetDiscoveryResponse()
	if err != nil {
		t.Fatal(err)
	}
	if out.VersionInfo == "" {
		t.Error("unexpected response empty version")
	}
	if n := len(out.Resources); n != num {
		t.Errorf("unexpected number of responses: got %d, want %d", n, num)
	}
	if version != "" && out.VersionInfo != version {
		t.Errorf("unexpected version: got %q, want %q", out.VersionInfo, version)
	}
	if out.TypeUrl != testType {
		t.Errorf("unexpected type URL: %q", out.TypeUrl)
	}
}

type resourceInfo struct {
	name    string
	version string
}

func verifyDeltaResponse(t *testing.T, ch <-chan DeltaResponse, resources []resourceInfo, deleted []string) {
	t.Helper()
	r := <-ch
	if r.GetDeltaRequest().TypeUrl != testType {
		t.Errorf("unexpected empty request type URL: %q", r.GetDeltaRequest().TypeUrl)
	}
	out, err := r.GetDeltaDiscoveryResponse()
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Resources) != len(resources) {
		t.Errorf("unexpected number of responses: got %d, want %d", len(out.Resources), len(resources))
	}
	for _, r := range resources {
		found := false
		for _, r1 := range out.Resources {
			if r1.Name == r.name && r1.Version == r.version {
				found = true
				break
			} else if r1.Name == r.name {
				t.Errorf("unexpected version for resource %q: got %q, want %q", r.name, r1.Version, r.version)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("resource with name %q not found in response", r.name)
		}
	}
	if out.TypeUrl != testType {
		t.Errorf("unexpected type URL: %q", out.TypeUrl)
	}
	if len(out.RemovedResources) != len(deleted) {
		t.Errorf("unexpected number of removed resurces: got %d, want %d", len(out.RemovedResources), len(deleted))
	}
	for _, r := range deleted {
		found := false
		for _, rr := range out.RemovedResources {
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

func TestLinearInitialResources(t *testing.T) {
	streamState := stream.NewStreamState(false, map[string]string{})
	c := NewLinearCache(testType, WithInitialResources(map[string]types.Resource{"a": testResource("a"), "b": testResource("b")}))
	w := make(chan Response, 1)
	c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType}, streamState, w)
	verifyResponse(t, w, "0", 1)
	c.CreateWatch(&Request{TypeUrl: testType}, streamState, w)
	verifyResponse(t, w, "0", 2)
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
	assert.NoError(t, err)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w1, []resourceInfo{{"a", hash}}, nil)
	verifyDeltaResponse(t, w2, []resourceInfo{{"a", hash}}, nil)
}

func TestLinearDeltaExistingResources(t *testing.T) {
	c := NewLinearCache(testType)
	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	hashA := hashResource(t, a)
	err := c.UpdateResource("a", a)
	assert.NoError(t, err)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err = c.UpdateResource("b", b)
	assert.NoError(t, err)

	state := stream.NewStreamState(false, map[string]string{"b": "", "c": ""}) // watching b and c - not interested in a
	w := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}}, []string{})

	state = stream.NewStreamState(false, map[string]string{"a": "", "b": ""})
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
	assert.NoError(t, err)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err = c.UpdateResource("b", b)
	assert.NoError(t, err)

	state := stream.NewStreamState(false, map[string]string{"a": "", "b": hashB})
	w := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"a", hashA}}, nil) // b is up to date and shouldn't be returned

	state = stream.NewStreamState(false, map[string]string{"a": hashA, "b": hashB})
	w = make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)
	b = &endpoint.ClusterLoadAssignment{ClusterName: "b", Endpoints: []*endpoint.LocalityLbEndpoints{{Priority: 10}}} // new version of b
	hashB = hashResource(t, b)
	err = c.UpdateResource("b", b)
	assert.NoError(t, err)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}}, nil)
}

func TestLinearDeltaResourceUpdate(t *testing.T) {
	c := NewLinearCache(testType)
	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	hashA := hashResource(t, a)
	err := c.UpdateResource("a", a)
	assert.NoError(t, err)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err = c.UpdateResource("b", b)
	assert.NoError(t, err)

	state := stream.NewStreamState(false, map[string]string{"a": "", "b": ""})
	w := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}, {"a", hashA}}, nil)

	state = stream.NewStreamState(false, map[string]string{"a": hashA, "b": hashB})
	w = make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)

	a = &endpoint.ClusterLoadAssignment{ClusterName: "a", Endpoints: []*endpoint.LocalityLbEndpoints{ //resource update
		{Priority: 10},
	}}
	hashA = hashResource(t, a)
	err = c.UpdateResource("a", a)
	assert.NoError(t, err)
	verifyDeltaResponse(t, w, []resourceInfo{{"a", hashA}}, nil)
}

func TestLinearDeltaResourceDelete(t *testing.T) {
	c := NewLinearCache(testType)
	a := &endpoint.ClusterLoadAssignment{ClusterName: "a"}
	hashA := hashResource(t, a)
	err := c.UpdateResource("a", a)
	assert.NoError(t, err)
	b := &endpoint.ClusterLoadAssignment{ClusterName: "b"}
	hashB := hashResource(t, b)
	err = c.UpdateResource("b", b)
	assert.NoError(t, err)

	state := stream.NewStreamState(false, map[string]string{"a": "", "b": ""})
	w := make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	checkDeltaWatchCount(t, c, 0)
	verifyDeltaResponse(t, w, []resourceInfo{{"b", hashB}, {"a", hashA}}, nil)

	state = stream.NewStreamState(false, map[string]string{"a": hashA, "b": hashB})
	w = make(chan DeltaResponse, 1)
	c.CreateDeltaWatch(&DeltaRequest{TypeUrl: testType}, state, w)
	mustBlockDelta(t, w)
	checkDeltaWatchCount(t, c, 1)

	a = &endpoint.ClusterLoadAssignment{ClusterName: "a", Endpoints: []*endpoint.LocalityLbEndpoints{ //resource update
		{Priority: 10},
	}}
	hashA = hashResource(t, a)
	c.SetResources(map[string]types.Resource{"a": a})
	verifyDeltaResponse(t, w, []resourceInfo{{"a", hashA}}, []string{"b"})
}
