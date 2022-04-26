package ambex

import (
	ecp_v2_cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
)

// FastpathSnapshot holds envoy configuration that bypasses python.
type FastpathSnapshot struct {
	Snapshot  *ecp_v2_cache.Snapshot
	Endpoints *Endpoints
}
