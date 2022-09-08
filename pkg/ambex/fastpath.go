package ambex

import (
	v1 "github.com/emissary-ingress/emissary/v3/internal/ir/caching/v1"
	ecp_v3_cache "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
)

// FastpathSnapshot holds envoy configuration that bypasses python.
type FastpathSnapshot struct {
	Snapshot      *ecp_v3_cache.Snapshot
	Endpoints     *Endpoints
	CachePolicies []v1.CachePolicyContext
	CacheMap      v1.CacheMap
}
