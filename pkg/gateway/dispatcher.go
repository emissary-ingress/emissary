package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/pkg/errors"

	apiv2 "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	apiv2_core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	apiv2_endpoint "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/endpoint"
	apiv2_route "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/route"

	ecp_cache_types "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	ecp_v2_cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
	ecp_v2_resource "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v2"
	ecp_wellknown "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/wellknown"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/dlib/dlog"
)

// The Dispatcher struct allows transforms to be registered for different kinds of kubernetes
// resources and invokes those transforms to produce compiled envoy configurations. It also knows
// how to assemble the compiled envoy configuration into a complete snapshot.
//
// Currently the dispatch process is relatively simple, each resource is processed as an independent
// unit. This is sufficient for the gateway API since the currently implemented resources are
// conveniently defined in such a way as to make them independent.
//
// Consistency is guaranteed assuming transform functions don't use out of band communication to
// include information from other resources. This guarantee is achieved because each transform is
// only passed a single resource and can therefore only use information from that one
// resource. Changes to any other resource cannot impact the result of that transform.
//
// Not all the edgestack resources are defined as conveniently, so the Dispatcher design is expected
// to be extended in two ways to handle resources with more complex interdependencies:
//
//   1. Grouping -- This feature would cover resources that need to be processed as a group,
//      e.g. Mappings that get grouped together based on prefix. Instead of dispatching at the
//      granularity of a single resource, the dispatcher will track groups of resources that need to
//      be processed together via a logical "hash" function provided at registration. Whenever any
//      item in a given bucket changes, the dispatcher will transform the entire bucket.
//
//   2. Dependencies -- This feature would cover resources that need to lookup the contents of other
//      resources in order to properly implement their transform. This would be done by passing the
//      transform function a Query API. Any resources queried by the transform would be
//      automatically tracked as a dependency of that resource. The dependencies would then be used
//      to perform invalidation whenever a resource is Upsert()ed.
//
type Dispatcher struct {
	// Map from kind to transform function.
	transforms map[string]reflect.Value
	configs    map[string]*CompiledConfig

	version         string
	changeCount     int
	snapshot        *ecp_v2_cache.Snapshot
	endpointWatches map[string]bool
}

type ResourceRef struct {
	Kind      string
	Namespace string
	Name      string
}

// resourceKey produces a fully qualified key for a kubernetes resource.
func resourceKey(resource kates.Object) string {
	gvk := resource.GetObjectKind().GroupVersionKind()
	return resourceKeyFromParts(gvk.Kind, resource.GetNamespace(), resource.GetName())
}

func resourceKeyFromParts(kind, namespace, name string) string {
	return fmt.Sprintf("%s:%s:%s", kind, namespace, name)
}

// NewDispatcher creates a new and empty *Dispatcher struct.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		transforms: map[string]reflect.Value{},
		configs:    map[string]*CompiledConfig{},
	}
}

// Register registers a transform function for the specified kubernetes resource. The transform
// argument must be a function that takes a single resource of the supplied "kind" and returns a
// single CompiledConfig object, i.e.: `func(Kind) CompiledConfig`
func (d *Dispatcher) Register(kind string, transform interface{}) error {
	_, ok := d.transforms[kind]
	if ok {
		return errors.Errorf("duplicate transform: %+v", transform)
	}

	xform := reflect.ValueOf(transform)

	d.transforms[kind] = xform

	return nil
}

// IsRegistered returns true if the given kind can be processed by this dispatcher.
func (d *Dispatcher) IsRegistered(kind string) bool {
	_, ok := d.transforms[kind]
	return ok
}

// Upsert processes the given kubernetes resource whether it is new or just updated.
func (d *Dispatcher) Upsert(resource kates.Object) error {
	gvk := resource.GetObjectKind().GroupVersionKind()
	xform, ok := d.transforms[gvk.Kind]
	if !ok {
		return errors.Errorf("no transform for kind: %q", gvk.Kind)
	}

	key := resourceKey(resource)

	var config *CompiledConfig
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				e, ok := r.(error)
				if ok {
					err = errors.Wrapf(e, "internal error processing %s", key)
				} else {
					err = errors.Errorf("internal error processing %s: %+v", key, e)
				}
			}
		}()
		result := xform.Call([]reflect.Value{reflect.ValueOf(resource)})
		config = result[0].Interface().(*CompiledConfig)
	}()

	if err != nil {
		return err
	}

	d.configs[key] = config
	// Clear out the snapshot so we regenerate one.
	d.snapshot = nil
	return nil
}

// Delete processes the deletion of the given kubernetes resource.
func (d *Dispatcher) Delete(resource kates.Object) {
	key := resourceKey(resource)
	delete(d.configs, key)

	// Clear out the snapshot so we regenerate one.
	d.snapshot = nil
}

func (d *Dispatcher) DeleteKey(kind, namespace, name string) {
	key := resourceKeyFromParts(kind, namespace, name)
	delete(d.configs, key)
	d.snapshot = nil
}

// UpsertYaml parses the supplied yaml and invokes Upsert on the result.
func (d *Dispatcher) UpsertYaml(manifests string) error {
	objs, err := kates.ParseManifests(manifests)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		err := d.Upsert(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetErrors returns all compiled items with errors.
func (d *Dispatcher) GetErrors() []*CompiledItem {
	var result []*CompiledItem
	for _, config := range d.configs {
		if config.Error != "" {
			result = append(result, &config.CompiledItem)
		}
		for _, l := range config.Listeners {
			if l.Error != "" {
				result = append(result, &l.CompiledItem)
			}
		}
		for _, r := range config.Routes {
			if r.Error != "" {
				result = append(result, &r.CompiledItem)
			}
			for _, cr := range r.ClusterRefs {
				if cr.Error != "" {
					result = append(result, &cr.CompiledItem)
				}
			}
		}
		for _, c := range config.Clusters {
			if c.Error != "" {
				result = append(result, &c.CompiledItem)
			}
		}
		for _, la := range config.LoadAssignments {
			if la.Error != "" {
				result = append(result, &la.CompiledItem)
			}
		}
	}
	return result
}

// GetSnapshot returns a version and a snapshot.
func (d *Dispatcher) GetSnapshot(ctx context.Context) (string, *ecp_v2_cache.Snapshot) {
	if d.snapshot == nil {
		d.buildSnapshot(ctx)
	}
	return d.version, d.snapshot
}

// GetListener returns a *apiv2.Listener with the specified name or nil if none exists.
func (d *Dispatcher) GetListener(ctx context.Context, name string) *apiv2.Listener {
	_, snap := d.GetSnapshot(ctx)
	for _, rsrc := range snap.Resources[ecp_cache_types.Listener].Items {
		l := rsrc.(*apiv2.Listener)
		if l.Name == name {
			return l
		}
	}
	return nil

}

// GetRouteConfiguration returns a *apiv2.RouteConfiguration with the specified name or nil if none
// exists.
func (d *Dispatcher) GetRouteConfiguration(ctx context.Context, name string) *apiv2.RouteConfiguration {
	_, snap := d.GetSnapshot(ctx)
	for _, rsrc := range snap.Resources[ecp_cache_types.Route].Items {
		r := rsrc.(*apiv2.RouteConfiguration)
		if r.Name == name {
			return r
		}
	}
	return nil
}

// IsWatched is a temporary hack for dealing with the way endpoint data currenttly flows from
// watcher -> ambex.n
func (d *Dispatcher) IsWatched(namespace, name string) bool {
	key := fmt.Sprintf("%s:%s", namespace, name)
	_, ok := d.endpointWatches[key]
	return ok
}

func (d *Dispatcher) buildClusterMap() (map[string]string, map[string]bool) {
	refs := map[string]string{}
	watches := map[string]bool{}
	for _, config := range d.configs {
		for _, route := range config.Routes {
			for _, ref := range route.ClusterRefs {
				refs[ref.Name] = ref.EndpointPath
				if route.Namespace != "" {
					key := fmt.Sprintf("%s:%s", route.Namespace, ref.Name)
					watches[key] = true
				}
			}
		}
	}
	return refs, watches
}

func (d *Dispatcher) buildEndpointMap() map[string]*apiv2.ClusterLoadAssignment {
	endpoints := map[string]*apiv2.ClusterLoadAssignment{}
	for _, config := range d.configs {
		for _, la := range config.LoadAssignments {
			endpoints[la.LoadAssignment.ClusterName] = la.LoadAssignment
		}
	}
	return endpoints
}

func (d *Dispatcher) buildRouteConfigurations() ([]ecp_cache_types.Resource, []ecp_cache_types.Resource) {
	listeners := []ecp_cache_types.Resource{}
	routes := []ecp_cache_types.Resource{}
	for _, config := range d.configs {
		for _, lst := range config.Listeners {
			listeners = append(listeners, lst.Listener)
			r := d.buildRouteConfiguration(lst)
			if r != nil {
				routes = append(routes, r)
			}
		}
	}
	return listeners, routes
}

func (d *Dispatcher) buildRouteConfiguration(lst *CompiledListener) *apiv2.RouteConfiguration {
	rdsName, isRds := getRdsName(lst.Listener)
	if !isRds {
		return nil
	}

	var routes []*apiv2_route.Route
	for _, config := range d.configs {
		for _, route := range config.Routes {
			if lst.Predicate(route) {
				routes = append(routes, route.Routes...)
			}
		}
	}

	return &apiv2.RouteConfiguration{
		Name: rdsName,
		VirtualHosts: []*apiv2_route.VirtualHost{
			{
				Name:    rdsName,
				Domains: lst.Domains,
				Routes:  routes,
			},
		},
	}
}

// getRdsName returns the RDS route configuration name configured for the listener and a flag
// indicating whether the listener uses Rds.
func getRdsName(l *apiv2.Listener) (string, bool) {
	for _, fc := range l.FilterChains {
		for _, f := range fc.Filters {
			if f.Name != ecp_wellknown.HTTPConnectionManager {
				continue
			}

			hcm := ecp_v2_resource.GetHTTPConnectionManager(f)
			if hcm != nil {
				rds := hcm.GetRds()
				if rds != nil {
					return rds.RouteConfigName, true
				}
			}
		}
	}
	return "", false
}

func (d *Dispatcher) buildSnapshot(ctx context.Context) {
	d.changeCount++
	d.version = fmt.Sprintf("v%d", d.changeCount)

	endpointMap := d.buildEndpointMap()
	clusterMap, endpointWatches := d.buildClusterMap()

	clusters := []ecp_cache_types.Resource{}
	endpoints := []ecp_cache_types.Resource{}
	for name, path := range clusterMap {
		clusters = append(clusters, makeCluster(name, path))
		key := path
		if key == "" {
			key = name
		}
		la, ok := endpointMap[key]
		if ok {
			endpoints = append(endpoints, la)
		} else {
			endpoints = append(endpoints, &apiv2.ClusterLoadAssignment{
				ClusterName: key,
				Endpoints:   []*apiv2_endpoint.LocalityLbEndpoints{},
			})
		}
	}

	listeners, routes := d.buildRouteConfigurations()

	snapshot := ecp_v2_cache.NewSnapshot(
		d.version,
		endpoints,
		clusters,
		routes,
		listeners,
		nil, // runtimes
		nil, // secrets
	)
	if err := snapshot.Consistent(); err != nil {
		bs, _ := json.MarshalIndent(snapshot, "", "  ")
		dlog.Errorf(ctx, "Dispatcher Snapshot inconsistency: %v: %s", err, bs)
	} else {
		d.snapshot = &snapshot
		d.endpointWatches = endpointWatches
	}
}

func makeCluster(name, path string) *apiv2.Cluster {
	return &apiv2.Cluster{
		Name:                 name,
		ConnectTimeout:       &duration.Duration{Seconds: 10},
		ClusterDiscoveryType: &apiv2.Cluster_Type{Type: apiv2.Cluster_EDS},
		EdsClusterConfig: &apiv2.Cluster_EdsClusterConfig{
			EdsConfig:   &apiv2_core.ConfigSource{ConfigSourceSpecifier: &apiv2_core.ConfigSource_Ads{}},
			ServiceName: path,
		},
	}
}
