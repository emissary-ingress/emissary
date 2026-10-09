package entrypoint

import (
	"context"
	"net"
	"net/http"
	"sync"
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
		diagdClient = &http.Client{Transport: DiagdTransport()}
	})

	return diagdClient
}
