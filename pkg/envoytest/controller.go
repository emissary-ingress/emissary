package envoytest

import (
	// standard library
	"context"
	"fmt"
	"sync"

	// third-party libraries
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"

	// envoy api v2
	apiv2 "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2"
	apiv2_core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/api/v2/core"
	apiv2_discovery "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/service/discovery/v2"

	// envoy control plane
	ecp_cache_types "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/types"
	ecp_v2_cache "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v2"
	ecp_log "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/log"
	ecp_v2_server "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v2"

	// first-party-libraries
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
)

// EnvoyController runs a go control plane for envoy that tracks ACKS/NACKS for configuration
// updates. This allows code to know when envoy has successfully reconfigured as well as have access
// to the error details when envoy is fed invalid configuration.
type EnvoyController struct {
	address string

	configCache ecp_v2_cache.SnapshotCache

	cond        *sync.Cond            // Protects the 'results' and 'outstanding'
	results     map[string]*errorInfo // Maps config version to error info related to that config
	outstanding map[string]ackInfo    // Maps response nonce to config version and typeURL

	// logCtx gets set when .Run() starts.
	logCtx context.Context
}

// ackInfo is used to correlate the nonce supplied in discovery responses to the error detail
// supplied in discovery requests.
type ackInfo struct {
	version string
	typeURL string
}

// Holds the error info associated with a configuration version. The details map is keyed by typeURL
// and has
type errorInfo struct {
	version string
	details map[string]*status.Status // keyed by typeURL
}

func (e *errorInfo) String() string {
	return fmt.Sprintf("%s %v", e.version, e.details)
}

// NewEnvoyControler creates a new envoy controller that binds to the supplied address when Run.
func NewEnvoyController(address string) *EnvoyController {
	ret := &EnvoyController{
		address:     address,
		cond:        sync.NewCond(&sync.Mutex{}),
		results:     map[string]*errorInfo{},
		outstanding: map[string]ackInfo{},
	}
	ret.configCache = ecp_v2_cache.NewSnapshotCache(
		true,              // ads
		ecNodeHash{},      // hash
		ecLogger{ec: ret}, // logger
	)
	return ret
}

// Configure will update the envoy configuration and block until the reconfiguration either succeeds
// or signals an error.
func (e *EnvoyController) Configure(ctx context.Context, node, version string, snapshot ecp_v2_cache.Snapshot) (*status.Status, error) {
	err := e.configCache.SetSnapshot(node, snapshot)
	if err != nil {
		return nil, err
	}

	// Versioning happens on a per type basis, so we need to figure out how many versions will be
	// requested in order to figure out how to properly check that the entire snapshot was
	// acked/nacked.
	var typeURLs []string
	if len(snapshot.Resources[ecp_cache_types.Endpoint].Items) > 0 {
		typeURLs = append(typeURLs, "type.googleapis.com/envoy.api.v2.ClusterLoadAssignment")
	}
	if len(snapshot.Resources[ecp_cache_types.Cluster].Items) > 0 {
		typeURLs = append(typeURLs, "type.googleapis.com/envoy.api.v2.Cluster")
	}
	if len(snapshot.Resources[ecp_cache_types.Route].Items) > 0 {
		typeURLs = append(typeURLs, "type.googleapis.com/envoy.api.v2.RouteConfiguration")
	}
	if len(snapshot.Resources[ecp_cache_types.Listener].Items) > 0 {
		typeURLs = append(typeURLs, "type.googleapis.com/envoy.api.v2.Listener")
	}

	for _, t := range typeURLs {
		status, err := e.waitFor(ctx, version, t)
		if err != nil {
			return nil, err
		}
		if status != nil {
			return status, nil
		}
	}

	return nil, nil
}

// waitFor blocks until the supplied version and typeURL are acknowledged by envoy. It returns the
// status if there is an error and nil if the configuration is successfully accepted by envoy.
func (e *EnvoyController) waitFor(ctx context.Context, version string, typeURL string) (*status.Status, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
	}()
	go func() {
		<-ctx.Done()
		e.cond.L.Lock()
		defer e.cond.L.Unlock()
		e.cond.Broadcast()
	}()

	var (
		retStatus *status.Status
		retErr    error
	)

	condition := func() bool {
		// If the Context was canceled, then go ahead and bail early.
		if err := ctx.Err(); err != nil {
			retErr = err
			return true
		}
		// See if our 'version' has a result yet.
		result, ok := e.results[version]
		if !ok {
			return false
		}
		// Does our typeURL within that result have a status?
		if status, ok := result.details[typeURL]; ok {
			retStatus = status
			return true
		}
		// OK, our 'version' has a result, but our typeURL doesn't have a status within it.
		// Do any other typeURLs within the result have an error status that we can return?
		for _, status := range result.details {
			if status != nil {
				retStatus = status
				return true
			}
		}
		// Darn, we didn't find anything worth returning.
		return false
	}

	e.cond.L.Lock()
	defer e.cond.L.Unlock()
	for !condition() {
		e.cond.Wait()
	}
	return retStatus, retErr
}

// Run the ADS server.
func (e *EnvoyController) Run(ctx context.Context) error {
	// The callbacks don't have access to a context, so we'll capture this one for them to use.
	e.logCtx = ctx

	srv := ecp_v2_server.NewServer(ctx,
		e.configCache,      // config
		ecCallbacks{ec: e}, // calbacks
	)

	grpcMux := grpc.NewServer()
	apiv2_discovery.RegisterAggregatedDiscoveryServiceServer(grpcMux, srv)
	apiv2.RegisterEndpointDiscoveryServiceServer(grpcMux, srv)
	apiv2.RegisterClusterDiscoveryServiceServer(grpcMux, srv)
	apiv2.RegisterRouteDiscoveryServiceServer(grpcMux, srv)
	apiv2.RegisterListenerDiscoveryServiceServer(grpcMux, srv)

	sc := &dhttp.ServerConfig{
		Handler: grpcMux,
	}
	return sc.ListenAndServe(ctx, e.address)
}

////////////////////////////////////////////////////////////////////////////////

type ecNodeHash struct{}

var _ ecp_v2_cache.NodeHash = ecNodeHash{}

// ID implements ecp_v2_cache.NodeHash.
func (ecNodeHash) ID(node *apiv2_core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

////////////////////////////////////////////////////////////////////////////////

type ecCallbacks struct {
	ec *EnvoyController
}

var _ ecp_v2_server.Callbacks = ecCallbacks{}

// OnStreamOpen implements ecp_v2_server.Callbacks.
func (ecc ecCallbacks) OnStreamOpen(_ context.Context, sid int64, stype string) error {
	//e.Infof("Stream open[%v]: %v", sid, stype)
	return nil
}

// OnStreamClosed implements ecp_v2_server.Callbacks.
func (ecc ecCallbacks) OnStreamClosed(sid int64) {
	//e.Infof("Stream closed[%v]", sid)
}

// OnStreamRequest implements ecp_v2_server.Callbacks.
func (ecc ecCallbacks) OnStreamRequest(sid int64, req *apiv2.DiscoveryRequest) error {
	//e.Infof("Stream request[%v]: %v", sid, req.TypeURL)

	ecc.ec.cond.L.Lock()
	defer ecc.ec.cond.L.Unlock()
	defer ecc.ec.cond.Broadcast()

	if ackInfo, ok := ecc.ec.outstanding[req.ResponseNonce]; ok {
		results, ok := ecc.ec.results[ackInfo.version]
		if !ok {
			results = &errorInfo{version: ackInfo.version, details: map[string]*status.Status{}}
			ecc.ec.results[ackInfo.version] = results
		}
		results.details[ackInfo.typeURL] = req.ErrorDetail
		delete(ecc.ec.outstanding, req.ResponseNonce)
	}

	return nil
}

// OnStreamResponse implements ecp_v2_server.Callbacks.
func (ecc ecCallbacks) OnStreamResponse(sid int64, req *apiv2.DiscoveryRequest, res *apiv2.DiscoveryResponse) {
	//e.Infof("Stream response[%v]: %v -> %v", sid, req.TypeURL, res.Nonce)

	ecc.ec.cond.L.Lock()
	defer ecc.ec.cond.L.Unlock()
	defer ecc.ec.cond.Broadcast()

	ecc.ec.outstanding[res.Nonce] = ackInfo{res.VersionInfo, res.TypeUrl}

}

// OnFetchRequest implements ecp_v2_server.Callbacks.
func (ecc ecCallbacks) OnFetchRequest(_ context.Context, r *apiv2.DiscoveryRequest) error {
	//e.Infof("Fetch request: %v", r)
	return nil
}

// OnFetchResponse implements ecp_v2_server.Callbacks.
func (ecc ecCallbacks) OnFetchResponse(req *apiv2.DiscoveryRequest, res *apiv2.DiscoveryResponse) {
	//e.Infof("Fetch response: %v -> %v", req, res)
}

////////////////////////////////////////////////////////////////////////////////

type ecLogger struct {
	ec *EnvoyController
}

var _ ecp_log.Logger = ecLogger{}

// Debugf implements ecp_log.Logger.
func (ecl ecLogger) Debugf(format string, args ...interface{}) {
	dlog.Debugf(ecl.ec.logCtx, format, args...)
}

// Infof implements ecp_log.Logger.
func (ecl ecLogger) Infof(format string, args ...interface{}) {
	dlog.Infof(ecl.ec.logCtx, format, args...)
}

// Warnf implements ecp_log.Logger.
func (ecl ecLogger) Warnf(format string, args ...interface{}) {
	dlog.Warnf(ecl.ec.logCtx, format, args...)
}

// Errorf implements ecp_log.Logger.
func (ecl ecLogger) Errorf(format string, args ...interface{}) {
	dlog.Errorf(ecl.ec.logCtx, format, args...)
}
