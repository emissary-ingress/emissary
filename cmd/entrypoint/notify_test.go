package entrypoint

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/dlib/dlog"
)

// Check if we return false when we get a connection refused.
func TestNotifyWebhookUrlConnectionRefused(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)

	finished, err := notifyWebhookUrl(ctx, http.DefaultClient, "test", "http://localhost:5555")
	assert.NoError(t, err)
	assert.False(t, finished)
}

// diagd's Unix-domain socket doesn't exist until diagd has finished starting up, and
// dialing a socket that isn't there gives ENOENT rather than ECONNREFUSED. That's still
// just "not up yet", so we need to retry rather than treating it as fatal.
func TestNotifyWebhookUrlSocketMissing(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)

	// Not t.TempDir(): sockaddr_un.sun_path is only ~104 bytes, and the temp dir
	// paths that Go hands out are easily longer than that.
	dir, err := os.MkdirTemp("/tmp", "diagd")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	t.Setenv("AMBASSADOR_DIAGD_SOCKET", filepath.Join(dir, "nonexistent.sock"))

	finished, err := notifyWebhookUrl(ctx, DiagdClient(), "diagd", GetEventUrl())
	assert.NoError(t, err)
	assert.False(t, finished)
}

// Check that we panic if we do not get a properly formed http response of some kind such as an EOF.
func TestNotifyWebhookUrlEOF(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We want to generate an EOF for the connected client. This seems to do that.
		srv.CloseClientConnections()
	}))

	_, err := notifyWebhookUrl(ctx, http.DefaultClient, "test", srv.URL)
	assert.Error(t, err)
}
