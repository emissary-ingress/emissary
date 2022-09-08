package v1

import (
	"github.com/emissary-ingress/emissary/v3/internal/ir/types"
	simple_http_cachev3 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/extensions/cache/simple_http_cache/v3"
	cachev3 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/extensions/filters/http/cache/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

// CacheProviderType is an alias type for defining supported CacheProviders
type CacheProviderType = string

const (
	//InMemoryCacheProvider provides basic http caching support by storing the responses in-memory within envoy
	InMemoryCacheProvider CacheProviderType = "in-memory"
)

// CacheContext provides the basic information needed for xDS to configure the cache filter
// based on the CachePolicies that are applied to a http filter chain
type CacheContext struct {
	NamespacedName types.NamespacedName `json:"namespacedName"`
	ProviderType   CacheProviderType    `json:"providerType"`
}

// ToCacheConfig
func (cc CacheContext) ToCacheConfig() *cachev3.CacheConfig {
	simpleCache, _ := anypb.New(&simple_http_cachev3.SimpleHttpCacheConfig{})

	cacheConfig := &cachev3.CacheConfig{
		TypedConfig: simpleCache,
	}
	return cacheConfig
}

// CacheRuleContext provides the information for matching polices to determe if caching
// should be enabled for an envoy filter chain
type CacheRuleContext struct {
	// Host is the glob pattern used for matching
	Host string `json:"host"`

	// Path is the glob pattern used to for matching
	Path string `json:"path"`

	CacheRef types.NamespacedName `json:"cacheRef"`
}

// CacheMap is an alias over a map that can be used for fast lookup of CacheContext Resources
type CacheMap map[string]CacheContext

// CachePolicyContext provides an internal representation used by xDS server for determining
// whether to apply caching policy.
type CachePolicyContext struct {
	NamespacedName types.NamespacedName `json:"namespacedName"`
	Rules          []CacheRuleContext   `json:"rules"`
}

// CacheRuleMatch is the rule for a CachePolicy that is matched
type CacheRuleMatch struct {
	PolicyNamespacedName types.NamespacedName
	CacheRuleContext
}
