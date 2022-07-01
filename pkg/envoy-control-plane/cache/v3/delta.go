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

	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/stream/v3"
)

// groups together resource-related arguments for the createDeltaResponse function
type resourceContainer struct {
	resourceMap   map[string]types.Resource
	versionMap    map[string]string
	systemVersion string
}

func createDeltaResponse(ctx context.Context, req *DeltaRequest, state stream.StreamState, resources resourceContainer) *RawDeltaResponse {
	// variables to build our response with
	nextVersionMap := make(map[string]string)
	filtered := make([]types.Resource, 0, len(resources.resourceMap))
	toRemove := make([]string, 0)

	// If we are handling a wildcard request, we want to respond with all resources
	switch {
	case state.IsWildcard():
		for name, r := range resources.resourceMap {
			// Since we've already precomputed the version hashes of the new snapshot,
			// we can just set it here to be used for comparison later
			version := resources.versionMap[name]
			nextVersionMap[name] = version
			prevVersion, found := state.GetResourceVersions()[name]
			if !found || (prevVersion != nextVersionMap[name]) {
				filtered = append(filtered, r)
			}
		}
	default:
		// Reply only with the requested resources
		for name, prevVersion := range state.GetResourceVersions() {
			if r, ok := resources.resourceMap[name]; ok {
				nextVersion := resources.versionMap[name]
				if prevVersion != nextVersion {
					filtered = append(filtered, r)
				}
				nextVersionMap[name] = nextVersion
			} else {
				// We track non-existent resources for non-wildcard streams until the client explicitly unsubscribes from them.
				nextVersionMap[name] = ""
			}
		}
	}

	// Compute resources for removal regardless of the request type
	for name, prevVersion := range state.GetResourceVersions() {
		// The prevVersion != "" check is in place to make sure we are only sending an update to the client once right after it is removed.
		// If the client decides to keep the subscription we skip the add for every subsequent response.
		if _, ok := resources.resourceMap[name]; !ok && prevVersion != "" {
			toRemove = append(toRemove, name)
		}
	}

	return &RawDeltaResponse{
		DeltaRequest:      req,
		Resources:         filtered,
		RemovedResources:  toRemove,
		NextVersionMap:    nextVersionMap,
		SystemVersionInfo: resources.systemVersion,
		Ctx:               ctx,
	}
}
