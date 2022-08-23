package resource

import (
	"github.com/golang/protobuf/ptypes"

	core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	listener "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/listener"
	hcm "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/conversion"
)

// Resource types in xDS v2.
const (
	apiTypePrefix       = "type.googleapis.com/envoy.api.v2."
	discoveryTypePrefix = "type.googleapis.com/envoy.service.discovery.v2."
	EndpointType        = apiTypePrefix + "ClusterLoadAssignment"
	ClusterType         = apiTypePrefix + "Cluster"
	RouteType           = apiTypePrefix + "RouteConfiguration"
	ListenerType        = apiTypePrefix + "Listener"
	SecretType          = apiTypePrefix + "auth.Secret"
	ExtensionConfigType = apiTypePrefix + "ExtensionConfig" // This isn't actually supported for V2
	RuntimeType         = discoveryTypePrefix + "Runtime"

	// AnyType is used only by ADS
	AnyType = ""
)

// Fetch urls in xDS v2.
const (
	FetchEndpoints        = "/v2/discovery:endpoints"
	FetchClusters         = "/v2/discovery:clusters"
	FetchListeners        = "/v2/discovery:listeners"
	FetchRoutes           = "/v2/discovery:routes"
	FetchSecrets          = "/v2/discovery:secrets"
	FetchRuntimes         = "/v2/discovery:runtime"
	FetchExtensionConfigs = "/v3/discovery:extension_configs"
)

// DefaultAPIVersion is the api version
const DefaultAPIVersion = core.ApiVersion_V2

// GetHTTPConnectionManager creates a HttpConnectionManager from filter
func GetHTTPConnectionManager(filter *listener.Filter) *hcm.HttpConnectionManager {
	config := &hcm.HttpConnectionManager{}

	// use typed config if available
	if typedConfig := filter.GetTypedConfig(); typedConfig != nil {
		ptypes.UnmarshalAny(typedConfig, config)
	} else {
		conversion.StructToMessage(filter.GetConfig(), config)
	}
	return config
}
