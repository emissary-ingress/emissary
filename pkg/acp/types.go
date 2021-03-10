package acp

import (
	"context"
	"time"
)

// EnvoyFetcherResponse is a simple response struct for an envoyFetcher.
//
// XXX I'm a little torn about this -- should we just return a net/http Response?
// That looks like it's not necessarily easy to synthesize, though, so let's
// just keep it simple.
type EnvoyFetcherResponse struct {
	StatusCode int
	Text       []byte
}

// timeFetcher is a function that returns the current time. We use time.Now
// unless overridden for testing.
type timeFetcher func() time.Time

// envoyFetcher is a function that returns Envoy's stats. We supply a default
// envoyFetcher, but it can be overridden (usually for testing).
type envoyFetcher func(ctx context.Context) (*EnvoyFetcherResponse, error)
