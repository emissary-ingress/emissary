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
	"fmt"
	"testing"

	"github.com/datawire/ambassador/pkg/envoy-control-plane/cache/types"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
)

const (
	testType = "google.protobuf.StringValue"
)

func testResource(s string) types.Resource {
	return &wrappers.StringValue{Value: s}
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

func checkWatchCount(t *testing.T, c *LinearCache, name string, count int) {
	t.Helper()
	if i := c.NumWatches(name); i != count {
		t.Errorf("unexpected number of watches for %q: got %d, want %d", name, i, count)
	}
}

func mustBlock(t *testing.T, w <-chan Response) {
	select {
	case <-w:
		t.Error("watch must block")
	default:
	}
}

func TestLinearInitialResources(t *testing.T) {
	c := NewLinearCache(testType, WithInitialResources(map[string]types.Resource{"a": testResource("a"), "b": testResource("b")}))
	w, _ := c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType})
	verifyResponse(t, w, "0", 1)
	w, _ = c.CreateWatch(&Request{TypeUrl: testType})
	verifyResponse(t, w, "0", 2)
}

func TestLinearCornerCases(t *testing.T) {
	c := NewLinearCache(testType)
	err := c.UpdateResource("a", nil)
	if err == nil {
		t.Error("expected error on nil resource")
	}
	// create an incorrect type URL request
	w, _ := c.CreateWatch(&Request{TypeUrl: "test"})
	select {
	case _, more := <-w:
		if more {
			t.Error("should be closed by the producer")
		}
	default:
		t.Error("channel should be closed")
	}
}

func TestLinearBasic(t *testing.T) {
	c := NewLinearCache(testType)

	// Create watches before a resource is ready
	w1, _ := c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"})
	mustBlock(t, w1)
	w, _ := c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"})
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 2)
	checkWatchCount(t, c, "b", 1)
	c.UpdateResource("a", testResource("a"))
	checkWatchCount(t, c, "a", 0)
	checkWatchCount(t, c, "b", 0)
	verifyResponse(t, w1, "1", 1)
	verifyResponse(t, w, "1", 1)

	// Request again, should get same response
	w, _ = c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"})
	checkWatchCount(t, c, "a", 0)
	verifyResponse(t, w, "1", 1)
	w, _ = c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"})
	checkWatchCount(t, c, "a", 0)
	verifyResponse(t, w, "1", 1)

	// Add another element and update the first, response should be different
	c.UpdateResource("b", testResource("b"))
	c.UpdateResource("a", testResource("aa"))
	w, _ = c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"})
	verifyResponse(t, w, "3", 1)
	w, _ = c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"})
	verifyResponse(t, w, "3", 2)
}

func TestLinearVersionPrefix(t *testing.T) {
	c := NewLinearCache(testType, WithVersionPrefix("instance1-"))

	w, _ := c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"})
	verifyResponse(t, w, "instance1-0", 0)

	c.UpdateResource("a", testResource("a"))
	w, _ = c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"})
	verifyResponse(t, w, "instance1-1", 1)

	w, _ = c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "instance1-1"})
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 1)
}

func TestLinearDeletion(t *testing.T) {
	c := NewLinearCache(testType, WithInitialResources(map[string]types.Resource{"a": testResource("a"), "b": testResource("b")}))
	w, _ := c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "0"})
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 1)
	c.DeleteResource("a")
	verifyResponse(t, w, "1", 0)
	checkWatchCount(t, c, "a", 0)
	w, _ = c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"})
	verifyResponse(t, w, "1", 1)
	checkWatchCount(t, c, "b", 0)
	c.DeleteResource("b")
	w, _ = c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "1"})
	verifyResponse(t, w, "2", 0)
	checkWatchCount(t, c, "b", 0)
}

func TestLinearWatchTwo(t *testing.T) {
	c := NewLinearCache(testType, WithInitialResources(map[string]types.Resource{"a": testResource("a"), "b": testResource("b")}))
	w, _ := c.CreateWatch(&Request{ResourceNames: []string{"a", "b"}, TypeUrl: testType, VersionInfo: "0"})
	mustBlock(t, w)
	w1, _ := c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "0"})
	mustBlock(t, w1)
	c.UpdateResource("a", testResource("aa"))
	// should only get the modified resource
	verifyResponse(t, w, "1", 1)
	verifyResponse(t, w1, "1", 2)
}

func TestLinearCancel(t *testing.T) {
	c := NewLinearCache(testType)
	c.UpdateResource("a", testResource("a"))

	// cancel watch-all
	w, cancel := c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "1"})
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 1)
	cancel()
	checkWatchCount(t, c, "a", 0)

	// cancel watch for "a"
	w, cancel = c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "1"})
	mustBlock(t, w)
	checkWatchCount(t, c, "a", 1)
	cancel()
	checkWatchCount(t, c, "a", 0)

	// open four watches for "a" and "b" and two for all, cancel one of each, make sure the second one is unaffected
	w, cancel = c.CreateWatch(&Request{ResourceNames: []string{"a"}, TypeUrl: testType, VersionInfo: "1"})
	w2, cancel2 := c.CreateWatch(&Request{ResourceNames: []string{"b"}, TypeUrl: testType, VersionInfo: "1"})
	w3, cancel3 := c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "1"})
	w4, cancel4 := c.CreateWatch(&Request{TypeUrl: testType, VersionInfo: "1"})
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

func TestLinearConcurrentSetWatch(t *testing.T) {
	c := NewLinearCache(testType)
	n := 50
	for i := 0; i < 2*n; i++ {
		func(i int) {
			t.Run(fmt.Sprintf("worker%d", i), func(t *testing.T) {
				t.Parallel()
				id := fmt.Sprintf("%d", i)
				if i%2 == 0 {
					t.Logf("update resource %q", id)
					c.UpdateResource(id, testResource(id))
				} else {
					id2 := fmt.Sprintf("%d", i-1)
					t.Logf("request resources %q and %q", id, id2)
					value, _ := c.CreateWatch(&Request{
						// Only expect one to become stale
						ResourceNames: []string{id, id2},
						VersionInfo:   "0",
						TypeUrl:       testType,
					})
					// wait until all updates apply
					verifyResponse(t, value, "", 1)
				}
			})
		}(i)
	}
}
