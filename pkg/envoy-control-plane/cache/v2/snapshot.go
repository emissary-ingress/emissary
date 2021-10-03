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
	"errors"
	"fmt"
	"time"

	"github.com/datawire/ambassador/pkg/envoy-control-plane/cache/types"
)

// Resources is a versioned group of resources.
type Resources struct {
	// Version information.
	Version string

	// Items in the group indexed by name.
	Items map[string]types.ResourceWithTtl
}

// IndexResourcesByName creates a map from the resource name to the resource.
func IndexResourcesByName(items []types.ResourceWithTtl) map[string]types.ResourceWithTtl {
	indexed := make(map[string]types.ResourceWithTtl)
	for _, item := range items {
		indexed[GetResourceName(item.Resource)] = item
	}
	return indexed
}

// NewResources creates a new resource group.
func NewResources(version string, items []types.Resource) Resources {
	itemsWithTtl := []types.ResourceWithTtl{}
	for _, item := range items {
		itemsWithTtl = append(itemsWithTtl, types.ResourceWithTtl{Resource: item})
	}
	return NewResourcesWithTtl(version, itemsWithTtl)
}

// NewResources creates a new resource group.
func NewResourcesWithTtl(version string, items []types.ResourceWithTtl) Resources {
	return Resources{
		Version: version,
		Items:   IndexResourcesByName(items),
	}
}

// Snapshot is an internally consistent snapshot of xDS resources.
// Consistency is important for the convergence as different resource types
// from the snapshot may be delivered to the proxy in arbitrary order.
type Snapshot struct {
	Resources [types.UnknownType]Resources
}

// NewSnapshot creates a snapshot from response types and a version.
func NewSnapshot(version string,
	endpoints []types.Resource,
	clusters []types.Resource,
	routes []types.Resource,
	listeners []types.Resource,
	runtimes []types.Resource,
	secrets []types.Resource) Snapshot {
	return NewSnapshotWithResources(version, SnapshotResources{
		Endpoints: endpoints,
		Clusters:  clusters,
		Routes:    routes,
		Listeners: listeners,
		Runtimes:  runtimes,
		Secrets:   secrets,
	})
}

// SnapshotResources contains the resources to construct a snapshot from.
type SnapshotResources struct {
	Endpoints        []types.Resource
	Clusters         []types.Resource
	Routes           []types.Resource
	Listeners        []types.Resource
	Runtimes         []types.Resource
	Secrets          []types.Resource
	ExtensionConfigs []types.Resource
}

// NewSnapshotWithResources creates a snapshot from response types and a version.
func NewSnapshotWithResources(version string, resources SnapshotResources) Snapshot {
	out := Snapshot{}
	out.Resources[types.Endpoint] = NewResources(version, resources.Endpoints)
	out.Resources[types.Cluster] = NewResources(version, resources.Clusters)
	out.Resources[types.Route] = NewResources(version, resources.Routes)
	out.Resources[types.Listener] = NewResources(version, resources.Listeners)
	out.Resources[types.Runtime] = NewResources(version, resources.Runtimes)
	out.Resources[types.Secret] = NewResources(version, resources.Secrets)
	out.Resources[types.ExtensionConfig] = NewResources(version, resources.ExtensionConfigs)
	return out
}

type ResourceWithTtl struct {
	Resources []types.Resource
	Ttl       *time.Duration
}

func NewSnapshotWithTtls(version string,
	endpoints []types.ResourceWithTtl,
	clusters []types.ResourceWithTtl,
	routes []types.ResourceWithTtl,
	listeners []types.ResourceWithTtl,
	runtimes []types.ResourceWithTtl,
	secrets []types.ResourceWithTtl) Snapshot {
	out := Snapshot{}
	out.Resources[types.Endpoint] = NewResourcesWithTtl(version, endpoints)
	out.Resources[types.Cluster] = NewResourcesWithTtl(version, clusters)
	out.Resources[types.Route] = NewResourcesWithTtl(version, routes)
	out.Resources[types.Listener] = NewResourcesWithTtl(version, listeners)
	out.Resources[types.Runtime] = NewResourcesWithTtl(version, runtimes)
	out.Resources[types.Secret] = NewResourcesWithTtl(version, secrets)
	return out
}

// Consistent check verifies that the dependent resources are exactly listed in the
// snapshot:
// - all EDS resources are listed by name in CDS resources
// - all RDS resources are listed by name in LDS resources
//
// Note that clusters and listeners are requested without name references, so
// Envoy will accept the snapshot list of clusters as-is even if it does not match
// all references found in xDS.
func (s *Snapshot) Consistent() error {
	if s == nil {
		return errors.New("nil snapshot")
	}
	endpoints := GetResourceReferences(s.Resources[types.Cluster].Items)
	if len(endpoints) != len(s.Resources[types.Endpoint].Items) {
		return fmt.Errorf("mismatched endpoint reference and resource lengths: %v != %d", endpoints, len(s.Resources[types.Endpoint].Items))
	}
	if err := superset(endpoints, s.Resources[types.Endpoint].Items); err != nil {
		return err
	}

	routes := GetResourceReferences(s.Resources[types.Listener].Items)
	if len(routes) != len(s.Resources[types.Route].Items) {
		return fmt.Errorf("mismatched route reference and resource lengths: %v != %d", routes, len(s.Resources[types.Route].Items))
	}
	return superset(routes, s.Resources[types.Route].Items)
}

// GetResources selects snapshot resources by type, returning the map of resources.
func (s *Snapshot) GetResources(typeURL string) map[string]types.Resource {
	resources := s.GetResourcesAndTtl(typeURL)
	if resources == nil {
		return nil
	}

	withoutTtl := make(map[string]types.Resource, len(resources))

	for k, v := range resources {
		withoutTtl[k] = v.Resource
	}

	return withoutTtl
}

// GetResourcesAndTtl selects snapshot resources by type, returning the map of resources and the associated TTL.
func (s *Snapshot) GetResourcesAndTtl(typeURL string) map[string]types.ResourceWithTtl {
	if s == nil {
		return nil
	}
	typ := GetResponseType(typeURL)
	if typ == types.UnknownType {
		return nil
	}
	return s.Resources[typ].Items
}

// GetVersion returns the version for a resource type.
func (s *Snapshot) GetVersion(typeURL string) string {
	if s == nil {
		return ""
	}
	typ := GetResponseType(typeURL)
	if typ == types.UnknownType {
		return ""
	}
	return s.Resources[typ].Version
}
