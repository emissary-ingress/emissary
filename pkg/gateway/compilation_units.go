package gateway

import (
	v3cluster "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/cluster/v3"
	v3endpoint "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/endpoint/v3"
	v3listener "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/listener/v3"
	v3route "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/route/v3"
	gw "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

// The types in this file primarily decorate envoy configuration with pointers back to Sources
// and/or error messages. In the case of CompiledListener there are some additional fields that
// allow the Dispatcher to automatically assemble RouteConfigurations for a given v2.Listener.

// CompiledItem has fields common to all compilation units.
type CompiledItem struct {
	Source    Source // Tracks the source of truth for whatever produced this compiled item.
	Namespace string // The namespace of whatever produced this item.
	Error     string // Holds any error associated with this compiled item.
}

func NewCompiledItem(source Source) CompiledItem {
	return CompiledItem{Source: source}
}

func NewCompiledItemError(source Source, error string) CompiledItem {
	return CompiledItem{Source: source, Error: error}
}

// CompiledConfig can hold any amount of any kind of envoy configuration fragments. All compile
// functions produce this type.
type CompiledConfig struct {
	CompiledItem
	Listeners       []*CompiledListener
	Routes          []*CompiledRoute
	Clusters        []*CompiledCluster
	LoadAssignments []*CompiledLoadAssignment
}

// CompiledListener is an envoy Listener plus a Predicate that the dispatcher uses to determine
// which routes to supply to the listener.
type CompiledListener struct {
	CompiledItem
	Listener *v3listener.Listener

	// The predicate determines which routes belong to which listeners. If the listener specifies
	// and Rds configuration, this Predicate and the Domains below will be used to construct a
	// RouteConfiguration from all the available CompiledRoutes.
	Predicate func(route *CompiledRoute) bool
	Domains   []string
}

// CompiledRoute is
type CompiledRoute struct {
	CompiledItem

	// This field will likely get replaced with something more astract, e.g. just info about the
	// source such as labels kind, namespace, name, etc.
	HTTPRoute *gw.HTTPRoute

	Routes      []*v3route.Route
	ClusterRefs []*ClusterRef
}

// ClusterRef represents a reference to an envoy v2.Cluster.
type ClusterRef struct {
	CompiledItem
	Name string

	// These are temporary fields to deal with how endpoints are currently plumbed from the watcher
	// through to ambex.
	EndpointPath string
}

// CompiledCluster decorates an envoy v2.Cluster.
type CompiledCluster struct {
	CompiledItem
	Cluster *v3cluster.Cluster
}

// CompiledLoadAssignment decorates an envoy v2.ClusterLoadAssignment.
type CompiledLoadAssignment struct {
	CompiledItem
	LoadAssignment *v3endpoint.ClusterLoadAssignment
}
