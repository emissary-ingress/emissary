package entrypoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Check if we return false when we get a connection refused.
func TestNotifyWebhookUrlConnectionRefused(t *testing.T) {
	ctx := context.Background()

	assert.False(t, notifyWebhookUrl(ctx, "test", "http://localhost:5555"))
}

// Check that we panic if we do not get a properly formed http response of some kind such as an EOF.
func TestNotifyWebhookUrlEOF(t *testing.T) {
	ctx := context.Background()

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We want to generate an EOF for the connected client. This seems to do that.
		srv.CloseClientConnections()
	}))

	func() {
		defer func() {
			e := recover()
			assert.Error(t, e.(error))
		}()
		notifyWebhookUrl(ctx, "test", srv.URL)
		assert.Fail(t, "did not panic on EOF")
	}()
}
