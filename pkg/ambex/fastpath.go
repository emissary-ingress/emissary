package ambex

import (
	v3tlsconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/transport_sockets/tls/v3"
	ecp_v3_cache "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
)

// FastpathSnapshot holds envoy configuration that bypasses python.
type FastpathSnapshot struct {
	Snapshot  *ecp_v3_cache.Snapshot
	Endpoints *Endpoints
	Secrets   []*v3tlsconfig.Secret
}
