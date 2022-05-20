package delta

import (
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/stream/v3"
)

// watches for all delta xDS resource types
type watches struct {
	deltaWatches map[string]watch

	// Opaque resources share a muxed channel
	deltaMuxedResponses chan cache.DeltaResponse
}

// newWatches creates and initializes watches.
func newWatches() watches {
	// deltaMuxedResponses needs a buffer to release go-routines populating it
	return watches{
		deltaWatches:        make(map[string]watch, int(types.UnknownType)),
		deltaMuxedResponses: make(chan cache.DeltaResponse, int(types.UnknownType)),
	}
}

// Cancel all watches
func (w *watches) Cancel() {
	for _, watch := range w.deltaWatches {
		if watch.cancel != nil {
			watch.cancel()
		}
	}
}

// watch contains the necessary modifiables for receiving resource responses
type watch struct {
	responses chan cache.DeltaResponse
	cancel    func()
	nonce     string

	state stream.StreamState
}

// Cancel calls terminate and cancel
func (w *watch) Cancel() {
	if w.cancel != nil {
		w.cancel()
	}
	close(w.responses)
}
