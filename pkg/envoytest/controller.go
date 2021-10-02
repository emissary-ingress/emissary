package envoytest

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	v2 "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v2"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/v2"
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
)

// EnvoyController runs a go control plane for envoy that tracks ACKS/NACKS for configuration
// updates. This allows code to know when envoy has successfully reconfigured as well as have access
// to the error details when envoy is fed invalid configuration.
type EnvoyController struct {
	address string

	configCache cache.SnapshotCache

	// Protects the errors and outstanding fields.
	cond        *sync.Cond
	errors      map[string]*errorInfo // Maps config version to error info related to that config
	outstanding map[string]ackInfo    // Maps response nonce to config version and typeUrl

	// Captured context for logging callbacks.
	logCtx context.Context
}

// ackInfo is used to correlate the nonce supplied in discovery responses to the error detail
// supplied in discovery requests.
type ackInfo struct {
	version string
	typeUrl string
}

// Holds the error info associated with a configuration version. The details map is keyed by typeUrl and has
type errorInfo struct {
	version string
	details map[string]*status.Status // keyed by typeUrl
}

func (e *errorInfo) String() string {
	return fmt.Sprintf("%s %v", e.version, e.details)
}

// NewEnvoyControler creates a new envoy controller that binds to the supplied address when Run.
func NewEnvoyController(address string) *EnvoyController {
	result := &EnvoyController{
		address:     address,
		cond:        sync.NewCond(&sync.Mutex{}),
		errors:      map[string]*errorInfo{},
		outstanding: map[string]ackInfo{},
	}
	result.configCache = cache.NewSnapshotCache(true, result, result)
	return result
}

// Configure will update the envoy configuration and block until the reconfiguration either succeeds
// or signals an error.
func (e *EnvoyController) Configure(node, version string, snapshot cache.Snapshot) *status.Status {
	err := e.configCache.SetSnapshot(node, snapshot)
	if err != nil {
		panic(err)
	}

	// Versioning happens on a per type basis, so we need to figure out how many versions will be
	// requested in order to figure out how to properly check that the entire snapshot was
	// acked/nacked.
	typeUrls := []string{}
	if len(snapshot.Resources[types.Endpoint].Items) > 0 {
		typeUrls = append(typeUrls, "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment")
	}
	if len(snapshot.Resources[types.Cluster].Items) > 0 {
		typeUrls = append(typeUrls, "type.googleapis.com/envoy.api.v2.Cluster")
	}
	if len(snapshot.Resources[types.Route].Items) > 0 {
		typeUrls = append(typeUrls, "type.googleapis.com/envoy.api.v2.RouteConfiguration")
	}
	if len(snapshot.Resources[types.Listener].Items) > 0 {
		typeUrls = append(typeUrls, "type.googleapis.com/envoy.api.v2.Listener")
	}

	for _, t := range typeUrls {
		status := e.waitFor(version, t)
		if status != nil {
			return status
		}
	}

	return nil
}

// waitFor blocks until the supplied version and typeUrl are acknowledged by envoy. It returns the
// status if there is an error and nil if the configuration is successfully accepted by envoy.
func (e *EnvoyController) waitFor(version string, typeUrl string) *status.Status {
	e.cond.L.Lock()
	defer e.cond.L.Unlock()
	for {
		error, ok := e.errors[version]
		if ok {
			for k, v := range error.details {
				if v != nil {
					return v
				}
				if k == typeUrl {
					return v
				}
			}
		}
		e.cond.Wait()
	}
}

// Run the ADS server.
func (e *EnvoyController) Run(ctx context.Context) error {
	// The callbacks don't have access to a context, so we'll capture this one for them to use.
	e.logCtx = ctx

	grpcServer := grpc.NewServer()
	srv := server.NewServer(ctx, e.configCache, e)

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)
	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, srv)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, srv)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, srv)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, srv)

	lis, err := net.Listen("tcp", e.address)
	if err != nil {
		return err
	}

	sc := &dhttp.ServerConfig{
		Handler: grpcServer,
	}
	if err := sc.Serve(ctx, lis); err != nil {
		if err != nil && err != context.Canceled {
			return err
		}
	}
	return nil
}

// SetupEnvoyController will create and run an EnvoyController with the supplied address as well as
// registering a Cleanup function to shutdown the EnvoyController.
func SetupEnvoyController(t *testing.T, address string) *EnvoyController {
	e := NewEnvoyController(address)
	ctx, cancel := context.WithCancel(dlog.NewTestContext(t, false))
	done := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		<-done
	})
	go func() {
		err := e.Run(ctx)
		if err != nil {
			t.Errorf("envoy controller exited with error: %+v", err)
		}
		close(done)
	}()
	return e
}

// ID is a callback function that the go control plane uses. I don't know what it does.
func (e EnvoyController) ID(node *core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

// OnStreamOpen is called once an xDS stream is open with a stream ID and the type URL (or "" for ADS).
func (e *EnvoyController) OnStreamOpen(_ context.Context, sid int64, stype string) error {
	//e.Infof("Stream open[%v]: %v", sid, stype)
	return nil
}

// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
func (e *EnvoyController) OnStreamClosed(sid int64) {
	//e.Infof("Stream closed[%v]", sid)
}

// OnStreamRequest is called once a request is received on a stream.
func (e *EnvoyController) OnStreamRequest(sid int64, req *v2.DiscoveryRequest) error {
	//e.Infof("Stream request[%v]: %v", sid, req.TypeUrl)

	func() {
		e.cond.L.Lock()
		defer e.cond.L.Unlock()
		ackInfo, ok := e.outstanding[req.ResponseNonce]
		if ok {
			errors, ok := e.errors[ackInfo.version]
			if !ok {
				errors = &errorInfo{version: ackInfo.version, details: map[string]*status.Status{}}
				e.errors[ackInfo.version] = errors
			}
			errors.details[ackInfo.typeUrl] = req.ErrorDetail
			delete(e.outstanding, req.ResponseNonce)
		}
		e.cond.Broadcast()
	}()

	return nil
}

// OnStreamResponse is called immediately prior to sending a response on a stream.
func (e *EnvoyController) OnStreamResponse(sid int64, req *v2.DiscoveryRequest, res *v2.DiscoveryResponse) {
	//e.Infof("Stream response[%v]: %v -> %v", sid, req.TypeUrl, res.Nonce)
	func() {
		e.cond.L.Lock()
		defer e.cond.L.Unlock()
		e.outstanding[res.Nonce] = ackInfo{res.VersionInfo, res.TypeUrl}
	}()
}

// OnFetchRequest is called for each Fetch request
func (e *EnvoyController) OnFetchRequest(_ context.Context, r *v2.DiscoveryRequest) error {
	//e.Infof("Fetch request: %v", r)
	return nil
}

// OnFetchResponse is called immediately prior to sending a response.
func (e *EnvoyController) OnFetchResponse(req *v2.DiscoveryRequest, res *v2.DiscoveryResponse) {
	//e.Infof("Fetch response: %v -> %v", req, res)
}

// The go control plane requires a logger to be injected. These methods implement the logger
// interface.
func (e *EnvoyController) Debugf(format string, args ...interface{}) {
	dlog.Debugf(e.logCtx, format, args...)
}
func (e *EnvoyController) Infof(format string, args ...interface{}) {
	dlog.Infof(e.logCtx, format, args...)
}
func (e *EnvoyController) Warnf(format string, args ...interface{}) {
	dlog.Warnf(e.logCtx, format, args...)
}
func (e *EnvoyController) Errorf(format string, args ...interface{}) {
	dlog.Errorf(e.logCtx, format, args...)
}
