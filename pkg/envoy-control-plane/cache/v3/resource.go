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
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"google.golang.org/protobuf/proto"

	cluster "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/cluster/v3"
	core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	endpoint "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/endpoint/v3"
	listener "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/listener/v3"
	route "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/route/v3"
	hcm "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"
	auth "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/extensions/transport_sockets/tls/v3"
	runtime "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/runtime/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/resource/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/wellknown"
)

// GetResponseType returns the enumeration for a valid xDS type URL.
func GetResponseType(typeURL resource.Type) types.ResponseType {
	switch typeURL {
	case resource.EndpointType:
		return types.Endpoint
	case resource.ClusterType:
		return types.Cluster
	case resource.RouteType:
		return types.Route
	case resource.ScopedRouteType:
		return types.ScopedRoute
	case resource.ListenerType:
		return types.Listener
	case resource.SecretType:
		return types.Secret
	case resource.RuntimeType:
		return types.Runtime
	case resource.ExtensionConfigType:
		return types.ExtensionConfig
	}
	return types.UnknownType
}

// GetResponseTypeURL returns the type url for a valid enum.
func GetResponseTypeURL(responseType types.ResponseType) (string, error) {
	switch responseType {
	case types.Endpoint:
		return resource.EndpointType, nil
	case types.Cluster:
		return resource.ClusterType, nil
	case types.Route:
		return resource.RouteType, nil
	case types.ScopedRoute:
		return resource.ScopedRouteType, nil
	case types.Listener:
		return resource.ListenerType, nil
	case types.Secret:
		return resource.SecretType, nil
	case types.Runtime:
		return resource.RuntimeType, nil
	case types.ExtensionConfig:
		return resource.ExtensionConfigType, nil
	}

	return "", fmt.Errorf("couldn't map response type to known resource type")
}

// GetResourceName returns the resource name for a valid xDS response type.
func GetResourceName(res types.Resource) string {
	switch v := res.(type) {
	case *endpoint.ClusterLoadAssignment:
		return v.GetClusterName()
	case *cluster.Cluster:
		return v.GetName()
	case *route.RouteConfiguration:
		return v.GetName()
	case *route.ScopedRouteConfiguration:
		return v.GetName()
	case *listener.Listener:
		return v.GetName()
	case *auth.Secret:
		return v.GetName()
	case *runtime.Runtime:
		return v.GetName()
	case *core.TypedExtensionConfig:
		return v.GetName()
	default:
		return ""
	}
}

// MarshalResource converts the Resource to MarshaledResource.
func MarshalResource(resource types.Resource) (types.MarshaledResource, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(resource)
}

// GetResourceReferences returns a map of dependent resources keyed by resource type, given a map of resources.
// (EDS cluster names for CDS, RDS/SRDS routes names for LDS, RDS route names for SRDS).
func GetResourceReferences(resources map[string]types.ResourceWithTTL) map[resource.Type]map[string]bool {
	out := make(map[resource.Type]map[string]bool)
	getResourceReferences(resources, out)

	return out
}

// GetAllResourceReferences returns a map of dependent resources keyed by resources type, given all resources.
func GetAllResourceReferences(resourceGroups [types.UnknownType]Resources) map[resource.Type]map[string]bool {
	ret := map[resource.Type]map[string]bool{}

	// We only check resources that we expect to have references to other resources.
	responseTypesWithReferences := map[types.ResponseType]struct{}{
		types.Cluster:     {},
		types.Listener:    {},
		types.ScopedRoute: {},
	}

	for responseType, resourceGroup := range resourceGroups {
		if _, ok := responseTypesWithReferences[types.ResponseType(responseType)]; ok {
			items := resourceGroup.Items
			getResourceReferences(items, ret)
		}
	}

	return ret
}

func getResourceReferences(resources map[string]types.ResourceWithTTL, out map[resource.Type]map[string]bool) {
	for _, res := range resources {
		if res.Resource == nil {
			continue
		}

		switch v := res.Resource.(type) {
		case *endpoint.ClusterLoadAssignment:
			// No dependencies.
		case *cluster.Cluster:
			getClusterReferences(v, out)
		case *route.RouteConfiguration:
			// References to clusters in both routes (and listeners) are not included
			// in the result, because the clusters are retrieved in bulk currently,
			// and not by name.
		case *route.ScopedRouteConfiguration:
			getScopedRouteReferences(v, out)
		case *listener.Listener:
			getListenerReferences(v, out)
		case *runtime.Runtime:
			// no dependencies
		}
	}
}

func mapMerge(dst map[string]bool, src map[string]bool) {
	for k, v := range src {
		dst[k] = v
	}
}

// Clusters will reference either the endpoint's cluster name or ServiceName override.
func getClusterReferences(src *cluster.Cluster, out map[resource.Type]map[string]bool) {
	endpoints := map[string]bool{}

	switch typ := src.ClusterDiscoveryType.(type) {
	case *cluster.Cluster_Type:
		if typ.Type == cluster.Cluster_EDS {
			if src.EdsClusterConfig != nil && src.EdsClusterConfig.ServiceName != "" {
				endpoints[src.EdsClusterConfig.ServiceName] = true
			} else {
				endpoints[src.Name] = true
			}
		}
	}

	if len(endpoints) > 0 {
		if _, ok := out[resource.EndpointType]; !ok {
			out[resource.EndpointType] = map[string]bool{}
		}

		mapMerge(out[resource.EndpointType], endpoints)
	}
}

// HTTP listeners will either reference ScopedRoutes or Routes.
func getListenerReferences(src *listener.Listener, out map[resource.Type]map[string]bool) {
	scopedRoutes := map[string]bool{}
	routes := map[string]bool{}

	// extract route configuration names from HTTP connection manager
	for _, chain := range src.FilterChains {
		for _, filter := range chain.Filters {
			if filter.Name != wellknown.HTTPConnectionManager {
				continue
			}

			config := resource.GetHTTPConnectionManager(filter)
			if config == nil {
				continue
			}

			routeSpecifier := config.RouteSpecifier
			switch r := routeSpecifier.(type) {
			case *hcm.HttpConnectionManager_Rds:
				if r != nil && r.Rds != nil {
					routes[r.Rds.RouteConfigName] = true
				}

			case *hcm.HttpConnectionManager_ScopedRoutes:
				if r != nil && r.ScopedRoutes != nil {
					scopedRoutes[r.ScopedRoutes.Name] = true
				}
			}
		}
	}

	if len(scopedRoutes) > 0 {
		if _, ok := out[resource.ScopedRouteType]; !ok {
			out[resource.ScopedRouteType] = map[string]bool{}
		}

		mapMerge(out[resource.ScopedRouteType], scopedRoutes)
	}
	if len(routes) > 0 {
		if _, ok := out[resource.RouteType]; !ok {
			out[resource.RouteType] = map[string]bool{}
		}

		mapMerge(out[resource.RouteType], routes)
	}
}

func getScopedRouteReferences(src *route.ScopedRouteConfiguration, out map[resource.Type]map[string]bool) {
	routes := map[string]bool{}

	// For a scoped route configuration, the dependent resource is the RouteConfigurationName.
	routes[src.RouteConfigurationName] = true

	if len(routes) > 0 {
		if _, ok := out[resource.RouteType]; !ok {
			out[resource.RouteType] = map[string]bool{}
		}

		mapMerge(out[resource.RouteType], routes)
	}
}

// HashResource will take a resource and create a SHA256 hash sum out of the marshaled bytes
func HashResource(resource []byte) string {
	hasher := sha256.New()
	hasher.Write(resource)

	return hex.EncodeToString(hasher.Sum(nil))
}
