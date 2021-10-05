package entrypoint

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dlog"
)

// Check if we return false when we get a connection refused.
func TestNotifyWebhookUrlConnectionRefused(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)

	finished, err := notifyWebhookUrl(ctx, "test", "http://localhost:5555")
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

	_, err := notifyWebhookUrl(ctx, "test", srv.URL)
	assert.Error(t, err)
}
