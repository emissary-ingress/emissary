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
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"

	v2 "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	auth "github.com/datawire/ambassador/pkg/api/envoy/api/v2/auth"
	hcm "github.com/datawire/ambassador/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/conversion"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/wellknown"
)

// Resource is the base interface for the xDS payload.
type Resource interface {
	proto.Message
}

// Resource types in xDS v2.
const (
	typePrefix   = "type.googleapis.com/envoy.api.v2."
	EndpointType = typePrefix + "ClusterLoadAssignment"
	ClusterType  = typePrefix + "Cluster"
	RouteType    = typePrefix + "RouteConfiguration"
	ListenerType = typePrefix + "Listener"
	SecretType   = typePrefix + "auth.Secret"

	// AnyType is used only by ADS
	AnyType = ""
)

var (
	// ResponseTypes are supported response types.
	ResponseTypes = []string{
		EndpointType,
		ClusterType,
		RouteType,
		ListenerType,
		SecretType,
	}
)

// GetResourceName returns the resource name for a valid xDS response type.
func GetResourceName(res Resource) string {
	switch v := res.(type) {
	case *v2.ClusterLoadAssignment:
		return v.GetClusterName()
	case *v2.Cluster:
		return v.GetName()
	case *v2.RouteConfiguration:
		return v.GetName()
	case *v2.Listener:
		return v.GetName()
	case *auth.Secret:
		return v.GetName()
	default:
		return ""
	}
}

// GetResourceReferences returns the names for dependent resources (EDS cluster
// names for CDS, RDS routes names for LDS).
func GetResourceReferences(resources map[string]Resource) map[string]bool {
	out := make(map[string]bool)
	for _, res := range resources {
		if res == nil {
			continue
		}
		switch v := res.(type) {
		case *v2.ClusterLoadAssignment:
			// no dependencies
		case *v2.Cluster:
			// for EDS type, use cluster name or ServiceName override
			switch typ := v.ClusterDiscoveryType.(type) {
			case *v2.Cluster_Type:
				if typ.Type == v2.Cluster_EDS {
					if v.EdsClusterConfig != nil && v.EdsClusterConfig.ServiceName != "" {
						out[v.EdsClusterConfig.ServiceName] = true
					} else {
						out[v.Name] = true
					}
				}
			}
		case *v2.RouteConfiguration:
			// References to clusters in both routes (and listeners) are not included
			// in the result, because the clusters are retrieved in bulk currently,
			// and not by name.
		case *v2.Listener:
			// extract route configuration names from HTTP connection manager
			for _, chain := range v.FilterChains {
				for _, filter := range chain.Filters {
					if filter.Name != wellknown.HTTPConnectionManager {
						continue
					}

					config := &hcm.HttpConnectionManager{}

					// use typed config if available
					if typedConfig := filter.GetTypedConfig(); typedConfig != nil {
						ptypes.UnmarshalAny(typedConfig, config)
					} else {
						conversion.StructToMessage(filter.GetConfig(), config)
					}

					if config == nil {
						continue
					}

					if rds, ok := config.RouteSpecifier.(*hcm.HttpConnectionManager_Rds); ok && rds != nil && rds.Rds != nil {
						out[rds.Rds.RouteConfigName] = true
					}
				}
			}
		}
	}
	return out
}
