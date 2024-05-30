package ambex

import (
	ecp_v3_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// FastpathSnapshot holds envoy configuration that bypasses python.
type FastpathSnapshot struct {
	Snapshot  *ecp_v3_cache.Snapshot
	Endpoints *Endpoints
}
