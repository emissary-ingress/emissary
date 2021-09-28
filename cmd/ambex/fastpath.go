package ambex

import (
	v2cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
)

// FastpathSnapshot holds envoy configuration that bypasses python.
type FastpathSnapshot struct {
	Snapshot  *v2cache.Snapshot
	Endpoints *Endpoints
}
