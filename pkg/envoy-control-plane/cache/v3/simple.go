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

package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/log"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/stream/v3"
)

// SnapshotCache is a snapshot-based cache that maintains a single versioned
// snapshot of responses per node. SnapshotCache consistently replies with the
// latest snapshot. For the protocol to work correctly in ADS mode, EDS/RDS
// requests are responded only when all resources in the snapshot xDS response
// are named as part of the request. It is expected that the CDS response names
// all EDS clusters, and the LDS response names all RDS routes in a snapshot,
// to ensure that Envoy makes the request for all EDS clusters or RDS routes
// eventually.
//
// SnapshotCache can operate as a REST or regular xDS backend. The snapshot
// can be partial, e.g. only include RDS or EDS resources.
type SnapshotCache interface {
	Cache

	// SetSnapshot sets a response snapshot for a node. For ADS, the snapshots
	// should have distinct versions and be internally consistent (e.g. all
	// referenced resources must be included in the snapshot).
	//
	// This method will cause the server to respond to all open watches, for which
	// the version differs from the snapshot version.
	SetSnapshot(node string, snapshot Snapshot) error

	// GetSnapshots gets the snapshot for a node.
	GetSnapshot(node string) (Snapshot, error)

	// ClearSnapshot removes all status and snapshot information associated with a node.
	ClearSnapshot(node string)

	// GetStatusInfo retrieves status information for a node ID.
	GetStatusInfo(string) StatusInfo

	// GetStatusKeys retrieves node IDs for all statuses.
	GetStatusKeys() []string
}

type heartbeatHandle struct {
	cancel func()
}

type snapshotCache struct {
	// watchCount and deltaWatchCount are atomic counters incremented for each watch respectively. They need to
	// be the first fields in the struct to guarantee 64-bit alignment,
	// which is a requirement for atomic operations on 64-bit operands to work on
	// 32-bit machines.
	watchCount      int64
	deltaWatchCount int64

	log log.Logger

	// ads flag to hold responses until all resources are named
	ads bool

	// snapshots are cached resources indexed by node IDs
	snapshots map[string]Snapshot

	// status information for all nodes indexed by node IDs
	status map[string]*statusInfo

	// hash is the hashing function for Envoy nodes
	hash NodeHash

	mu sync.RWMutex
}

// NewSnapshotCache initializes a simple cache.
//
// ADS flag forces a delay in responding to streaming requests until all
// resources are explicitly named in the request. This avoids the problem of a
// partial request over a single stream for a subset of resources which would
// require generating a fresh version for acknowledgement. ADS flag requires
// snapshot consistency. For non-ADS case (and fetch), multiple partial
// requests are sent across multiple streams and re-using the snapshot version
// is OK.
//
// Logger is optional.
func NewSnapshotCache(ads bool, hash NodeHash, logger log.Logger) SnapshotCache {
	return newSnapshotCache(ads, hash, logger)
}

func newSnapshotCache(ads bool, hash NodeHash, logger log.Logger) *snapshotCache {
	cache := &snapshotCache{
		log:       logger,
		ads:       ads,
		snapshots: make(map[string]Snapshot),
		status:    make(map[string]*statusInfo),
		hash:      hash,
	}

	return cache
}

// NewSnapshotCacheWithHeartbeating initializes a simple cache that sends periodic heartbeat
// responses for resources with a TTL.
//
// ADS flag forces a delay in responding to streaming requests until all
// resources are explicitly named in the request. This avoids the problem of a
// partial request over a single stream for a subset of resources which would
// require generating a fresh version for acknowledgement. ADS flag requires
// snapshot consistency. For non-ADS case (and fetch), multiple partial
// requests are sent across multiple streams and re-using the snapshot version
// is OK.
//
// Logger is optional.
//
// The context provides a way to cancel the heartbeating routine, while the heartbeatInterval
// parameter controls how often heartbeating occurs.
func NewSnapshotCacheWithHeartbeating(ctx context.Context, ads bool, hash NodeHash, logger log.Logger, heartbeatInterval time.Duration) SnapshotCache {
	cache := newSnapshotCache(ads, hash, logger)
	go func() {
		t := time.NewTicker(heartbeatInterval)

		for {
			select {
			case <-t.C:
				cache.mu.Lock()
				for node := range cache.status {
					// TODO(snowp): Omit heartbeats if a real response has been sent recently.
					cache.sendHeartbeats(ctx, node)
				}
				cache.mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
	return cache
}

func (cache *snapshotCache) sendHeartbeats(ctx context.Context, node string) {
	snapshot := cache.snapshots[node]
	if info, ok := cache.status[node]; ok {
		info.mu.Lock()
		for id, watch := range info.watches {
			// Respond with the current version regardless of whether the version has changed.
			version := snapshot.GetVersion(watch.Request.TypeUrl)
			resources := snapshot.GetResourcesAndTtl(watch.Request.TypeUrl)

			// TODO(snowp): Construct this once per type instead of once per watch.
			resourcesWithTtl := map[string]types.ResourceWithTtl{}
			for k, v := range resources {
				if v.Ttl != nil {
					resourcesWithTtl[k] = v
				}
			}

			if len(resourcesWithTtl) == 0 {
				continue
			}
			if cache.log != nil {
				cache.log.Debugf("respond open watch %d%v with heartbeat for version %q", id, watch.Request.ResourceNames, version)
			}

			cache.respond(watch.Request, watch.Response, resourcesWithTtl, version, true)

			// The watch must be deleted and we must rely on the client to ack this response to create a new watch.
			delete(info.watches, id)
		}
		info.mu.Unlock()
	}
}

// SetSnapshotCache updates a snapshot for a node.
func (cache *snapshotCache) SetSnapshot(node string, snapshot Snapshot) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// update the existing entry
	cache.snapshots[node] = snapshot

	// trigger existing watches for which version changed
	if info, ok := cache.status[node]; ok {
		info.mu.Lock()
		for id, watch := range info.watches {
			version := snapshot.GetVersion(watch.Request.TypeUrl)
			if version != watch.Request.VersionInfo {
				if cache.log != nil {
					cache.log.Debugf("respond open watch %d%v with new version %q", id, watch.Request.ResourceNames, version)
				}
				resources := snapshot.GetResourcesAndTtl(watch.Request.TypeUrl)
				cache.respond(watch.Request, watch.Response, resources, version, false)

				// discard the watch
				delete(info.watches, id)
			}
		}

		// We only calculate version hashes when using delta. We don't want to do this when using SOTW so we can avoid unecessary computational cost if not using delta.
		if len(info.deltaWatches) > 0 {
			err := snapshot.ConstructVersionMap()
			if err != nil {
				return err
			}
		}

		// process our delta watches
		for id, watch := range info.deltaWatches {
			res := respondDelta(
				watch.Request,
				watch.Response,
				watch.StreamState,
				snapshot,
				cache.log,
			)
			// If we detect a nil response here, that means there has been no state change
			// so we don't want to respond or remove any existing resource watches
			if res != nil {
				delete(info.deltaWatches, id)
			}
		}
		info.mu.Unlock()
	}

	return nil
}

// GetSnapshots gets the snapshot for a node, and returns an error if not found.
func (cache *snapshotCache) GetSnapshot(node string) (Snapshot, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	snap, ok := cache.snapshots[node]
	if !ok {
		return Snapshot{}, fmt.Errorf("no snapshot found for node %s", node)
	}
	return snap, nil
}

// ClearSnapshot clears snapshot and info for a node.
func (cache *snapshotCache) ClearSnapshot(node string) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	delete(cache.snapshots, node)
	delete(cache.status, node)
}

// nameSet creates a map from a string slice to value true.
func nameSet(names []string) map[string]bool {
	set := make(map[string]bool)
	for _, name := range names {
		set[name] = true
	}
	return set
}

// superset checks that all resources are listed in the names set.
func superset(names map[string]bool, resources map[string]types.ResourceWithTtl) error {
	for resourceName := range resources {
		if _, exists := names[resourceName]; !exists {
			return fmt.Errorf("%q not listed", resourceName)
		}
	}
	return nil
}

// CreateWatch returns a watch for an xDS request.
func (cache *snapshotCache) CreateWatch(request *Request) (chan Response, func()) {
	nodeID := cache.hash.ID(request.Node)

	cache.mu.Lock()
	defer cache.mu.Unlock()

	info, ok := cache.status[nodeID]
	if !ok {
		info = newStatusInfo(request.Node)
		cache.status[nodeID] = info
	}

	// update last watch request time
	info.mu.Lock()
	info.lastWatchRequestTime = time.Now()
	info.mu.Unlock()

	// allocate capacity 1 to allow one-time non-blocking use
	value := make(chan Response, 1)

	snapshot, exists := cache.snapshots[nodeID]
	version := snapshot.GetVersion(request.TypeUrl)

	// if the requested version is up-to-date or missing a response, leave an open watch
	if !exists || request.VersionInfo == version {
		watchID := cache.nextWatchID()
		if cache.log != nil {
			cache.log.Debugf("open watch %d for %s%v from nodeID %q, version %q", watchID,
				request.TypeUrl, request.ResourceNames, nodeID, request.VersionInfo)
		}
		info.mu.Lock()
		info.watches[watchID] = ResponseWatch{Request: request, Response: value}
		info.mu.Unlock()
		return value, cache.cancelWatch(nodeID, watchID)
	}

	// otherwise, the watch may be responded immediately
	resources := snapshot.GetResourcesAndTtl(request.TypeUrl)
	cache.respond(request, value, resources, version, false)

	return value, nil
}

func (cache *snapshotCache) nextWatchID() int64 {
	return atomic.AddInt64(&cache.watchCount, 1)
}

// cancellation function for cleaning stale watches
func (cache *snapshotCache) cancelWatch(nodeID string, watchID int64) func() {
	return func() {
		// uses the cache mutex
		cache.mu.Lock()
		defer cache.mu.Unlock()
		if info, ok := cache.status[nodeID]; ok {
			info.mu.Lock()
			delete(info.watches, watchID)
			info.mu.Unlock()
		}
	}
}

// Respond to a watch with the snapshot value. The value channel should have capacity not to block.
// TODO(kuat) do not respond always, see issue https://github.com/envoyproxy/go-control-plane/issues/46
func (cache *snapshotCache) respond(request *Request, value chan Response, resources map[string]types.ResourceWithTtl, version string, heartbeat bool) {
	// for ADS, the request names must match the snapshot names
	// if they do not, then the watch is never responded, and it is expected that envoy makes another request
	if len(request.ResourceNames) != 0 && cache.ads {
		if err := superset(nameSet(request.ResourceNames), resources); err != nil {
			if cache.log != nil {
				cache.log.Warnf("ADS mode: not responding to request: %v", err)
			}
			return
		}
	}
	if cache.log != nil {
		cache.log.Debugf("respond %s%v version %q with version %q",
			request.TypeUrl, request.ResourceNames, request.VersionInfo, version)
	}

	value <- createResponse(request, resources, version, heartbeat)
}

func createResponse(request *Request, resources map[string]types.ResourceWithTtl, version string, heartbeat bool) Response {
	filtered := make([]types.ResourceWithTtl, 0, len(resources))

	// Reply only with the requested resources. Envoy may ask each resource
	// individually in a separate stream. It is ok to reply with the same version
	// on separate streams since requests do not share their response versions.
	if len(request.ResourceNames) != 0 {
		set := nameSet(request.ResourceNames)
		for name, resource := range resources {
			if set[name] {
				filtered = append(filtered, resource)
			}
		}
	} else {
		for _, resource := range resources {
			filtered = append(filtered, resource)
		}
	}

	return &RawResponse{
		Request:   request,
		Version:   version,
		Resources: filtered,
		Heartbeat: heartbeat,
	}
}

// CreateDeltaWatch returns a watch for a delta xDS request which implements the Simple SnapshotCache.
func (cache *snapshotCache) CreateDeltaWatch(request *DeltaRequest, st *stream.StreamState) (chan DeltaResponse, func()) {
	nodeID := cache.hash.ID(request.Node)
	t := request.GetTypeUrl()

	cache.mu.Lock()
	defer cache.mu.Unlock()

	info, ok := cache.status[nodeID]
	if !ok {
		info = newStatusInfo(request.Node)
		cache.status[nodeID] = info
	}

	// update last watch request time
	info.SetLastDeltaWatchRequestTime(time.Now())

	value := make(chan DeltaResponse, 1)

	// find the current cache snapshot for the provided node
	snapshot, exists := cache.snapshots[nodeID]

	// There are three different cases that leads to a delayed watch trigger:
	// - no snapshot exists for the requested nodeID
	// - a snapshot exists, but we failed to initialize its version map
	// - we attempted to issue a response, but the caller is already up to date
	delayedResponse := !exists
	if exists {
		err := snapshot.ConstructVersionMap()
		if err != nil {
			cache.log.Errorf("failed to compute version for snapshot resources inline, waiting for next snapshot update")
			delayedResponse = true
		}
		delayedResponse = respondDelta(request, value, st, snapshot, cache.log) == nil
	}

	if delayedResponse {
		watchID := cache.nextDeltaWatchID()
		if cache.log != nil {
			cache.log.Infof("open delta watch ID:%d for %s Resources:%v from nodeID: %q, system version %q", watchID,
				t, st.ResourceVersions, nodeID, snapshot.GetVersion(t))
		}

		info.SetDeltaResponseWatch(watchID, DeltaResponseWatch{Request: request, Response: value, StreamState: st})

		return value, cache.cancelDeltaWatch(nodeID, watchID)
	}

	return value, nil
}

func (cache *snapshotCache) nextDeltaWatchID() int64 {
	return atomic.AddInt64(&cache.deltaWatchCount, 1)
}

// cancellation function for cleaning stale delta watches
func (cache *snapshotCache) cancelDeltaWatch(nodeID string, watchID int64) func() {
	return func() {
		cache.mu.Lock()
		defer cache.mu.Unlock()
		if info, ok := cache.status[nodeID]; ok {
			info.mu.Lock()
			delete(info.deltaWatches, watchID)
			info.mu.Unlock()
		}
	}
}

// Fetch implements the cache fetch function.
// Fetch is called on multiple streams, so responding to individual names with the same version works.
func (cache *snapshotCache) Fetch(ctx context.Context, request *Request) (Response, error) {
	nodeID := cache.hash.ID(request.Node)

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if snapshot, exists := cache.snapshots[nodeID]; exists {
		// Respond only if the request version is distinct from the current snapshot state.
		// It might be beneficial to hold the request since Envoy will re-attempt the refresh.
		version := snapshot.GetVersion(request.TypeUrl)
		if request.VersionInfo == version {
			if cache.log != nil {
				cache.log.Warnf("skip fetch: version up to date")
			}
			return nil, &types.SkipFetchError{}
		}

		resources := snapshot.GetResourcesAndTtl(request.TypeUrl)
		out := createResponse(request, resources, version, false)
		return out, nil
	}

	return nil, fmt.Errorf("missing snapshot for %q", nodeID)
}

// GetStatusInfo retrieves the status info for the node.
func (cache *snapshotCache) GetStatusInfo(node string) StatusInfo {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	info, exists := cache.status[node]
	if !exists {
		if cache.log != nil {
			cache.log.Warnf("node does not exist")
		}
		return nil
	}

	return info
}

// GetStatusKeys retrieves all node IDs in the status map.
func (cache *snapshotCache) GetStatusKeys() []string {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	out := make([]string, 0, len(cache.status))
	for id := range cache.status {
		out = append(out, id)
	}

	return out
}
