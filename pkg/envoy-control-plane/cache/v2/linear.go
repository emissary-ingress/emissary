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
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
)

type watches = map[chan Response]struct{}

// LinearCache supports collectons of opaque resources. This cache has a
// single collection indexed by resource names and manages resource versions
// internally. It implements the cache interface for a single type URL and
// should be combined with other caches via type URL muxing. It can be used to
// supply EDS entries, for example, uniformly across a fleet of proxies.
type LinearCache struct {
	// Type URL specific to the cache.
	typeURL string
	// Collection of resources indexed by name.
	resources map[string]types.Resource
	// Watches open by clients, indexed by resource name. Whenever resources
	// are changed, the watch is triggered.
	watches map[string]watches
	// Set of watches for all resources in the collection
	watchAll watches
	// Continously incremented version
	version uint64
	// Version prefix to be sent to the clients
	versionPrefix string
	// Versions for each resource by name.
	versionVector map[string]uint64
	mu            sync.Mutex
}

var _ Cache = &LinearCache{}

// Options for modifying the behavior of the linear cache.
type LinearCacheOption func(*LinearCache)

// WithVersionPrefix sets a version prefix of the form "prefixN" in the version info.
// Version prefix can be used to distinguish replicated instances of the cache, in case
// a client re-connects to another instance.
func WithVersionPrefix(prefix string) LinearCacheOption {
	return func(cache *LinearCache) {
		cache.versionPrefix = prefix
	}
}

// WithInitialResources initializes the initial set of resources.
func WithInitialResources(resources map[string]types.Resource) LinearCacheOption {
	return func(cache *LinearCache) {
		cache.resources = resources
		for name := range resources {
			cache.versionVector[name] = 0
		}
	}
}

// NewLinearCache creates a new cache. See the comments on the struct definition.
func NewLinearCache(typeURL string, opts ...LinearCacheOption) *LinearCache {
	out := &LinearCache{
		typeURL:       typeURL,
		resources:     make(map[string]types.Resource),
		watches:       make(map[string]watches),
		watchAll:      make(watches),
		version:       0,
		versionVector: make(map[string]uint64),
	}
	for _, opt := range opts {
		opt(out)
	}
	return out
}

func (cache *LinearCache) respond(value chan Response, staleResources []string) {
	var resources []types.Resource
	// TODO: optimize the resources slice creations across different clients
	if len(staleResources) == 0 {
		resources = make([]types.Resource, 0, len(cache.resources))
		for _, resource := range cache.resources {
			resources = append(resources, resource)
		}
	} else {
		resources = make([]types.Resource, 0, len(staleResources))
		for _, name := range staleResources {
			resource := cache.resources[name]
			if resource != nil {
				resources = append(resources, resource)
			}
		}
	}
	value <- &RawResponse{
		Request:   &Request{TypeUrl: cache.typeURL},
		Resources: resources,
		Version:   cache.versionPrefix + strconv.FormatUint(cache.version, 10),
	}
}

func (cache *LinearCache) notifyAll(modified map[string]struct{}) {
	// de-duplicate watches that need to be responded
	notifyList := make(map[chan Response][]string)
	for name := range modified {
		for watch := range cache.watches[name] {
			notifyList[watch] = append(notifyList[watch], name)
		}
		delete(cache.watches, name)
	}
	for value, stale := range notifyList {
		cache.respond(value, stale)
	}
	for value := range cache.watchAll {
		cache.respond(value, nil)
	}
	cache.watchAll = make(watches)
}

// UpdateResource updates a resource in the collection.
func (cache *LinearCache) UpdateResource(name string, res types.Resource) error {
	if res == nil {
		return errors.New("nil resource")
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.version += 1
	cache.versionVector[name] = cache.version
	cache.resources[name] = res

	// TODO: batch watch closures to prevent rapid updates
	cache.notifyAll(map[string]struct{}{name: {}})

	return nil
}

// DeleteResource removes a resource in the collection.
func (cache *LinearCache) DeleteResource(name string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.version += 1
	delete(cache.versionVector, name)
	delete(cache.resources, name)

	// TODO: batch watch closures to prevent rapid updates
	cache.notifyAll(map[string]struct{}{name: {}})
	return nil
}

func (cache *LinearCache) CreateWatch(request *Request) (chan Response, func()) {
	value := make(chan Response, 1)
	if request.TypeUrl != cache.typeURL {
		close(value)
		return value, nil
	}
	// If the version is not up to date, check whether any requested resource has
	// been updated between the last version and the current version. This avoids the problem
	// of sending empty updates whenever an irrelevant resource changes.
	stale := false
	staleResources := []string{} // empty means all

	// strip version prefix if it is present
	var lastVersion uint64
	var err error
	if strings.HasPrefix(request.VersionInfo, cache.versionPrefix) {
		lastVersion, err = strconv.ParseUint(request.VersionInfo[len(cache.versionPrefix):], 0, 64)
	} else {
		err = errors.New("mis-matched version prefix")
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	if err != nil {
		stale = true
		staleResources = request.ResourceNames
	} else if len(request.ResourceNames) == 0 {
		stale = lastVersion != cache.version
	} else {
		for _, name := range request.ResourceNames {
			// When a resource is removed, its version defaults 0 and it is not considered stale.
			if lastVersion < cache.versionVector[name] {
				stale = true
				staleResources = append(staleResources, name)
			}
		}
	}
	if stale {
		cache.respond(value, staleResources)
		return value, nil
	}
	// Create open watches since versions are up to date.
	if len(request.ResourceNames) == 0 {
		cache.watchAll[value] = struct{}{}
		return value, func() {
			cache.mu.Lock()
			defer cache.mu.Unlock()
			delete(cache.watchAll, value)
		}
	}
	for _, name := range request.ResourceNames {
		set, exists := cache.watches[name]
		if !exists {
			set = make(watches)
			cache.watches[name] = set
		}
		set[value] = struct{}{}
	}
	return value, func() {
		cache.mu.Lock()
		defer cache.mu.Unlock()
		for _, name := range request.ResourceNames {
			set, exists := cache.watches[name]
			if exists {
				delete(set, value)
			}
			if len(set) == 0 {
				delete(cache.watches, name)
			}
		}
	}
}

func (cache *LinearCache) Fetch(ctx context.Context, request *Request) (Response, error) {
	return nil, errors.New("not implemented")
}

// Number of active watches for a resource name.
func (cache *LinearCache) NumWatches(name string) int {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return len(cache.watches[name]) + len(cache.watchAll)
}
