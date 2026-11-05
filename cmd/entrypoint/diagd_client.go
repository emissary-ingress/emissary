package entrypoint

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// DiagdURLOrigin is the origin to use when building URLs for diagd. diagd
// listens on a Unix-domain socket rather than a TCP port, so the hostname here
// is a placeholder that never gets resolved -- DiagdTransport ignores it and
// dials the socket instead. It still has to be a syntactically valid URL,
// though, and it shows up in diagd's logs as the request's Host header.
const DiagdURLOrigin = "http://diagd"

var (
	diagdClientOnce sync.Once
	diagdClient     *http.Client
)

// DiagdTransport returns an http.RoundTripper that speaks HTTP over diagd's
// Unix-domain socket, no matter what host appears in the request URL. The
// socket path is looked up per-dial rather than cached, so that tests can
// relocate it.
func DiagdTransport() *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", GetDiagdSocketPath())
		},
	}
}

// DiagdClient returns a shared http.Client that talks to diagd over its
// Unix-domain socket.
func DiagdClient() *http.Client {
	diagdClientOnce.Do(func() {
		transport := DiagdTransport()
		// gunicorn closes an idle keep-alive connection after 2 seconds. A
		// snapshot POST that lands on a connection gunicorn is closing fails
		// with EOF, Go does not replay a POST that was already sent, and
		// notifyWebhookUrl treats that EOF as fatal. Dropping idle connections
		// sooner than gunicorn does keeps every POST off a closing connection.
		transport.IdleConnTimeout = time.Second
		diagdClient = &http.Client{Transport: transport}
	})

	return diagdClient
}
