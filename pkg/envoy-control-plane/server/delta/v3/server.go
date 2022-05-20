package delta

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	core "github.com/datawire/ambassador/v2/pkg/api/envoy/config/core/v3"
	discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/resource/v3"
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

var deltaErrorResponse = &cache.RawDeltaResponse{}

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

func (s *server) processDelta(str stream.DeltaStream, reqCh <-chan *discovery.DeltaDiscoveryRequest, defaultTypeURL string) error {
	streamID := atomic.AddInt64(&s.streamCount, 1)

	// streamNonce holds a unique nonce for req-resp pairs per xDS stream.
	var streamNonce int64

	// a collection of stack allocated watches per request type
	watches := newWatches()

	defer func() {
		watches.Cancel()
		if s.callbacks != nil {
			s.callbacks.OnDeltaStreamClosed(streamID)
		}
	}()

	// Sends a response, returns the new stream nonce
	send := func(resp cache.DeltaResponse) (string, error) {
		if resp == nil {
			return "", errors.New("missing response")
		}

		response, err := resp.GetDeltaDiscoveryResponse()
		if err != nil {
			return "", err
		}

		streamNonce = streamNonce + 1
		response.Nonce = strconv.FormatInt(streamNonce, 10)
		if s.callbacks != nil {
			s.callbacks.OnStreamDeltaResponse(streamID, resp.GetDeltaRequest(), response)
		}

		return response.Nonce, str.Send(response)
	}

	if s.callbacks != nil {
		if err := s.callbacks.OnDeltaStreamOpen(str.Context(), streamID, defaultTypeURL); err != nil {
			return err
		}
	}

	var node = &core.Node{}
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case resp, more := <-watches.deltaMuxedResponses:
			if !more {
				break
			}

			typ := resp.GetDeltaRequest().GetTypeUrl()
			if resp == deltaErrorResponse {
				return status.Errorf(codes.Unavailable, typ+" watch failed")
			}

			nonce, err := send(resp)
			if err != nil {
				return err
			}

			watch := watches.deltaWatches[typ]
			watch.nonce = nonce

			watch.state.SetResourceVersions(resp.GetNextVersionMap())
			watches.deltaWatches[typ] = watch
		case req, more := <-reqCh:
			// input stream ended or errored out
			if !more {
				return nil
			}
			if req == nil {
				return status.Errorf(codes.Unavailable, "empty request")
			}

			if s.callbacks != nil {
				if err := s.callbacks.OnStreamDeltaRequest(streamID, req); err != nil {
					return err
				}
			}

			// The node information might only be set on the first incoming delta discovery request, so store it here so we can
			// reset it on subsequent requests that omit it.
			if req.Node != nil {
				node = req.Node
			} else {
				req.Node = node
			}

			// type URL is required for ADS but is implicit for any other xDS stream
			if defaultTypeURL == resource.AnyType {
				if req.TypeUrl == "" {
					return status.Errorf(codes.InvalidArgument, "type URL is required for ADS")
				}
			} else if req.TypeUrl == "" {
				req.TypeUrl = defaultTypeURL
			}

			typeURL := req.GetTypeUrl()

			// cancel existing watch to (re-)request a newer version
			watch, ok := watches.deltaWatches[typeURL]
			if !ok {
				// Initialize the state of the stream.
				// Since there was no previous state, we know we're handling the first request of this type
				// so we set the initial resource versions if we have any, and also signal if this stream is in wildcard mode.
				watch.state = stream.NewStreamState(len(req.GetResourceNamesSubscribe()) == 0, req.GetInitialResourceVersions())
			} else {
				watch.Cancel()
			}

			s.subscribe(req.GetResourceNamesSubscribe(), watch.state.GetResourceVersions())
			s.unsubscribe(req.GetResourceNamesUnsubscribe(), watch.state.GetResourceVersions())

			watch.responses = make(chan cache.DeltaResponse, 1)
			watch.cancel = s.cache.CreateDeltaWatch(req, watch.state, watch.responses)
			watches.deltaWatches[typeURL] = watch

			go func() {
				resp, more := <-watch.responses
				if more {
					watches.deltaMuxedResponses <- resp
				}
			}()
		}
	}
}

func (s *server) DeltaStreamHandler(str stream.DeltaStream, typeURL string) error {
	// a channel for receiving incoming delta requests
	reqCh := make(chan *discovery.DeltaDiscoveryRequest)

	// we need to concurrently handle incoming requests since we kick off processDelta as a return
	go func() {
		for {
			select {
			case <-str.Context().Done():
				close(reqCh)
				return
			default:
				req, err := str.Recv()
				if err != nil {
					close(reqCh)
					return
				}

				reqCh <- req
			}
		}
	}()

	return s.processDelta(str, reqCh, typeURL)
}

// When we subscribe, we just want to make the cache know we are subscribing to a resource.
// Providing a name with an empty version is enough to make that happen.
func (s *server) subscribe(resources []string, sv map[string]string) {
	for _, resource := range resources {
		sv[resource] = ""
	}
}

// Unsubscriptions remove resources from the stream state to
// indicate to the cache that we don't care about the resource anymore
func (s *server) unsubscribe(resources []string, sv map[string]string) {
	for _, resource := range resources {
		delete(sv, resource)
	}
}
