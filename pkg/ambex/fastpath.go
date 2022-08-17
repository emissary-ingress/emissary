package ambex

import (
	ecp_v3_cache "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
)

// FastpathSnapshot holds envoy configuration that bypasses python.
//
// Note that "Endpoints" and "Secrets" are are the moral equivalent of IR types --
// and, in fact, should probably become proper IR types. They are _not_ yet tied
// specific Envoy versions here.
type FastpathSnapshot struct {
	Snapshot         *ecp_v3_cache.Snapshot
	Endpoints        *Endpoints
	Secrets          *Secrets
	ValidationGroups [][]string
}
