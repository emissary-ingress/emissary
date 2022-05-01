package delta

import (
	"context"

	discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/stream/v3"
)

// Server is a wrapper interface which is meant to hold the proper stream handler for each xDS protocol.
type Server interface {
	DeltaStreamHandler(stream stream.DeltaStream, typeURL string) error
}

type Callbacks interface {
	// OnDeltaStreamOpen is called once an incremental xDS stream is open with a stream ID and the type URL (or "" for ADS).
	// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
	OnDeltaStreamOpen(context.Context, int64, string) error
	// OnDeltaStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
	OnDeltaStreamClosed(int64)
	// OnStreamDeltaRequest is called once a request is received on a stream.
	// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
	OnStreamDeltaRequest(int64, *discovery.DeltaDiscoveryRequest) error
	// OnStreamDelatResponse is called immediately prior to sending a response on a stream.
	OnStreamDeltaResponse(int64, *discovery.DeltaDiscoveryRequest, *discovery.DeltaDiscoveryResponse)
}

type server struct {
	cache     cache.ConfigWatcher
	callbacks Callbacks

	// total stream count for counting bi-di streams
	streamCount int64
	ctx         context.Context
}

// NewServer creates a delta xDS specific server which utilizes a ConfigWatcher and delta Callbacks.
func NewServer(ctx context.Context, config cache.ConfigWatcher, callbacks Callbacks) Server {
	return &server{
		cache:     config,
		callbacks: callbacks,
		ctx:       ctx,
	}
}

func (s *server) DeltaStreamHandler(str stream.DeltaStream, typeURL string) error {
	return nil
}
