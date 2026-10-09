package entrypoint

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/acp"
)

// The health check port is reachable from off-pod, and `kubectl port-forward`
// makes off-pod traffic look like it came from 127.0.0.1, so diagd's internal
// API -- above all POST /_internal/v0/watt, which submits a new snapshot --
// must not be reachable here no matter what the client claims about itself.
func TestHealthCheckMuxRejectsInternalAPI(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)
	sm := healthCheckMux(ctx, acp.NewAmbassadorWatcher(acp.NewEnvoyWatcher(), acp.NewDiagdWatcher()))

	testcases := []struct {
		name   string
		method string
		path   string
	}{
		{"internal-root", http.MethodGet, "/_internal"},
		{"watt", http.MethodPost, "/_internal/v0/watt"},
		{"encoded-slash", http.MethodPost, "/_internal%2fv0%2fwatt"},
		{"fs", http.MethodPost, "/_internal/v0/fs?path=/tmp/evil"},
		{"ping", http.MethodGet, "/_internal/v0/ping"},
		{"features", http.MethodGet, "/_internal/v0/features"},
		// ServeMux cleans the path before matching, so this must not sneak past.
		{"traversal", http.MethodPost, "/ambassador/v0/../../_internal/v0/watt"},
		{"double-slash", http.MethodGet, "//_internal/v0/ping"},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			for _, spoof := range []bool{false, true} {
				path := tc.path

				// ServeMux answers dirty paths with a redirect to the cleaned path,
				// so follow those before checking the result -- a redirect on its own
				// proves nothing.
				for redirects := 0; ; redirects++ {
					require.Less(t, redirects, 5, "redirect loop")

					req := httptest.NewRequest(tc.method, path, nil)
					req.RemoteAddr = "10.42.1.7:54321"

					if spoof {
						req.Header.Set("X-Ambassador-Diag-IP", "127.0.0.1")
					}

					w := httptest.NewRecorder()
					sm.ServeHTTP(w, req)

					if w.Code/100 == 3 {
						path = w.Header().Get("Location")
						require.NotEmpty(t, path, "redirect with no Location")
						continue
					}

					assert.Equal(t, http.StatusNotFound, w.Code,
						"spoofed=%v: %s reached diagd", spoof, path)
					break
				}
			}
		})
	}
}

// Liveness and readiness have to keep working on the health check port.
func TestHealthCheckMuxServesHealthChecks(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)
	sm := healthCheckMux(ctx, acp.NewAmbassadorWatcher(acp.NewEnvoyWatcher(), acp.NewDiagdWatcher()))

	for _, path := range []string{"/ambassador/v0/check_alive", "/ambassador/v0/check_ready"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = "10.42.1.7:54321"

		w := httptest.NewRecorder()
		sm.ServeHTTP(w, req)

		// Nothing is actually running, so we expect 503 rather than 200 -- but we
		// must not get a 404, which would mean we'd stopped answering health
		// checks at all.
		require.NotEqual(t, http.StatusNotFound, w.Code, "%s returned 404", path)
	}
}
