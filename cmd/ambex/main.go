package ambex

/**********************************************
 * ambex: Ambassador Experimental ADS server
 *
 * Here's the deal.
 *
 * go-control-plane, several different classes manage this stuff:
 *
 * - The root of the world is a SnapshotCache.
 *   - import github.com/datawire/ambassador/pkg/envoy-control-plane/cache/v2, then refer
 *     to cache.SnapshotCache.
 *   - A collection of internally consistent configuration objects is a
 *     Snapshot (cache.Snapshot).
 *   - Snapshots are collected in the SnapshotCache.
 *   - A given SnapshotCache can hold configurations for multiple Envoys,
 *     identified by the Envoy 'node ID', which must be configured for the
 *     Envoy.
 * - The SnapshotCache can only hold go-control-plane configuration objects,
 *   so you have to build these up to hand to the SnapshotCache.
 * - The gRPC stuff is handled by a Server.
 *   - import github.com/datawire/ambassador/pkg/envoy-control-plane/server, then refer
 *     to server.Server.
 *   - Our runManagementServer (largely ripped off from the go-control-plane
 *     tests) gets this running. It takes a SnapshotCache (cleverly called a
 *     "config" for no reason I understand) and a gRPCServer as arguments.
 *   - _ALL_ the gRPC madness is handled by the Server, with the assistance
 *     of the methods in a callback object.
 * - Once the Server is running, Envoy can open a gRPC stream to it.
 *   - On connection, Envoy will get handed the most recent Snapshot that
 *     the Server's SnapshotCache knows about.
 *   - Whenever a newer Snapshot is added to the SnapshotCache, that Snapshot
 *     will get sent to the Envoy.
 * - We manage the SnapshotCache by loading envoy configuration from
 *   json and/or protobuf files on disk.
 *   - By default when we get a SIGHUP, we reload configuration.
 *   - When passed the -watch argument we reload whenever any file in
 *     the directory changes.
 */

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	// protobuf library
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	// envoy control plane
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	ctypes "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	v2cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
	v3cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v3"
	v2server "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/v2"
	v3server "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/v3"
	"github.com/datawire/ambassador/v2/pkg/memory"

	// envoy protobuf v2 -- Be sure to import the package of any types that the Python emits a
	// "@type" of in the generated config, even if that package is otherwise not used by ambex.
	v2 "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/auth"
	v2core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/accesslog/v2"
	v2bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/buffer/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/ext_authz/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/gzip/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/lua/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/rate_limit/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/rbac/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/http/router/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/filter/network/tcp_proxy/v2"
	v2discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v2"

	// Envoy API v3
	//
	// XXX TODO Is this actually true and necessary?
	// yes it is true and necessary. need to import all types so they can be decoded by `decode`
	// Be sure to import the package of any types that the Python
	// emits a "@type" of in the generated config, even if that package is otherwise
	// not used by ambex.
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/accesslog/v3"
	v3bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	v3clusterconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/config/cluster/v3"
	v3core "github.com/datawire/ambassador/v2/pkg/api/envoy/config/core/v3"
	v3endpointconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/config/endpoint/v3"
	v3listenerconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/config/listener/v3"
	v3routeconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/config/route/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/access_loggers/file/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/access_loggers/grpc/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/buffer/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/ext_authz/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/grpc_stats/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/gzip/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/lua/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/ratelimit/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/rbac/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/router/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/tcp_proxy/v3"
	v3cluster "github.com/datawire/ambassador/v2/pkg/api/envoy/service/cluster/v3"
	v3discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v3"
	v3endpoint "github.com/datawire/ambassador/v2/pkg/api/envoy/service/endpoint/v3"
	v3listener "github.com/datawire/ambassador/v2/pkg/api/envoy/service/listener/v3"
	v3route "github.com/datawire/ambassador/v2/pkg/api/envoy/service/route/v3"
	v3runtime "github.com/datawire/ambassador/v2/pkg/api/envoy/service/runtime/v3"

	// envoy protobuf v3 -- likewise
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/response_map/v3"

	// first-party libraries
	"github.com/datawire/dlib/dhttp"
)

type Args struct {
	debug bool
	watch bool

	adsNetwork string
	adsAddress string

	dirs []string
}

func parseArgs(rawArgs ...string) (*Args, error) {
	var args Args
	flagset := flag.NewFlagSet("ambex", flag.ContinueOnError)

	flagset.BoolVar(&args.debug, "debug", false, "Use debug logging")
	flagset.BoolVar(&args.watch, "watch", false, "Watch for file changes")

	// TODO(lukeshu): Consider changing the default here so we don't need to put it in entrypoint.sh
	flagset.StringVar(&args.adsNetwork, "ads-listen-network", "tcp", "network for ADS to listen on")
	flagset.StringVar(&args.adsAddress, "ads-listen-address", ":18000", "address (on --ads-listen-network) for ADS to listen on")

	var legacyAdsPort uint
	flagset.UintVar(&legacyAdsPort, "ads", 0, "port number for ADS to listen on--deprecated, use --ads-listen-address=:1234 instead")

	if err := flagset.Parse(rawArgs); err != nil {
		return nil, err
	}

	if legacyAdsPort != 0 {
		args.adsAddress = fmt.Sprintf(":%v", legacyAdsPort)
	}

	args.dirs = flagset.Args()
	if len(args.dirs) == 0 {
		args.dirs = []string{"."}
	}

	return &args, nil
}

// Hasher returns node ID as an ID
type HasherV2 struct {
}

// ID function
func (h HasherV2) ID(node *v2core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

// Hasher returns node ID as an ID
type HasherV3 struct {
}

// ID function
func (h HasherV3) ID(node *v3core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

// end Hasher stuff

// This feels kinda dumb.
type loggerv2 struct {
	*logrus.Logger
}

var log = &loggerv2{
	Logger: logrus.StandardLogger(),
}

type loggerv3 struct {
	*logrus.Logger
}

var logv3 = &loggerv3{
	Logger: logrus.StandardLogger(),
}

// run stuff
// RunManagementServer starts an xDS server at the given port.
func runManagementServer(ctx context.Context, server v2server.Server, serverv3 v3server.Server, adsNetwork, adsAddress string) {
	grpcServer := grpc.NewServer()

	lis, err := net.Listen(adsNetwork, adsAddress)
	if err != nil {
		log.WithError(err).Panic("failed to listen")
	}

	// register services
	v2discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	v3discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, serverv3)
	v3endpoint.RegisterEndpointDiscoveryServiceServer(grpcServer, serverv3)
	v3cluster.RegisterClusterDiscoveryServiceServer(grpcServer, serverv3)
	v3route.RegisterRouteDiscoveryServiceServer(grpcServer, serverv3)
	v3listener.RegisterListenerDiscoveryServiceServer(grpcServer, serverv3)

	log.WithFields(logrus.Fields{"addr": adsNetwork + ":" + adsAddress}).Info("Listening")
	go func() {
		sc := &dhttp.ServerConfig{
			Handler: grpcServer,
		}
		if err := sc.Serve(ctx, lis); err != nil {
			log.WithFields(logrus.Fields{"error": err}).Error("Management server exited")
		}
	}()
}

// Decoders for unmarshalling our config
var decoders = map[string](func(string, proto.Message) error){
	".json": jsonpb.UnmarshalString,
	".pb":   proto.UnmarshalText,
}

func isDecodable(name string) bool {
	if strings.HasPrefix(name, ".") {
		return false
	}

	ext := filepath.Ext(name)
	_, ok := decoders[ext]
	return ok
}

// Not sure if there is a better way to do this, but we cast to this
// so we can call the generated Validate method.
type Validatable interface {
	proto.Message
	Validate() error
}

func Decode(name string) (proto.Message, error) {
	any := &any.Any{}
	contents, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(name)
	decoder := decoders[ext]
	err = decoder(string(contents), any)
	if err != nil {
		return nil, err
	}

	var m ptypes.DynamicAny
	err = ptypes.UnmarshalAny(any, &m)
	if err != nil {
		return nil, err
	}

	var v = m.Message.(Validatable)

	err = v.Validate()
	if err != nil {
		return nil, err
	}
	log.Infof("Loaded file %s", name)
	return v, nil
}

func Merge(to, from proto.Message) {
	str, err := (&jsonpb.Marshaler{}).MarshalToString(from)
	if err != nil {
		panic(err)
	}
	err = jsonpb.UnmarshalString(str, to)
	if err != nil {
		panic(err)
	}
}

func Clone(src proto.Message) proto.Message {
	in := reflect.ValueOf(src)
	if in.IsNil() {
		return src
	}
	out := reflect.New(in.Type().Elem())
	dst := out.Interface().(proto.Message)
	Merge(dst, src)
	return dst
}

func update(ctx context.Context, config v2cache.SnapshotCache, configv3 v3cache.SnapshotCache, generation *int, dirs []string, edsEndpoints map[string]*v2.ClusterLoadAssignment, edsEndpointsV3 map[string]*v3endpointconfig.ClusterLoadAssignment, fastpathSnapshot *FastpathSnapshot, updates chan<- Update) {
	clusters := []ctypes.Resource{}  // v2.Cluster
	routes := []ctypes.Resource{}    // v2.RouteConfiguration
	listeners := []ctypes.Resource{} // v2.Listener
	runtimes := []ctypes.Resource{}  // discovery.Runtime

	clustersv3 := []ctypes.Resource{}  // v3.Cluster
	routesv3 := []ctypes.Resource{}    // v3.RouteConfiguration
	listenersv3 := []ctypes.Resource{} // v3.Listener
	runtimesv3 := []ctypes.Resource{}  // v3.Runtime

	var filenames []string

	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			log.WithError(err).Warnf("Error listing %v", dir)
			continue
		}
		for _, file := range files {
			name := file.Name()
			if isDecodable(name) {
				filenames = append(filenames, filepath.Join(dir, name))
			}
		}
	}

	for _, name := range filenames {
		m, e := Decode(name)
		if e != nil {
			log.Warnf("%s: %v", name, e)
			continue
		}
		var dst *[]ctypes.Resource
		switch m.(type) {
		case *v2.Cluster:
			dst = &clusters
		case *v2.RouteConfiguration:
			dst = &routes
		case *v2.Listener:
			dst = &listeners
		case *v2discovery.Runtime:
			dst = &runtimes
		case *v2bootstrap.Bootstrap:
			bs := m.(*v2bootstrap.Bootstrap)
			sr := bs.StaticResources
			for _, lst := range sr.Listeners {
				// When the RouteConfiguration is embedded in the listener, it will cause envoy to
				// go through a complete drain cycle whenever there is a routing change and that
				// will potentially disrupt in-flight requests. By converting all listeners to use
				// RDS rather than inlining their routing configuration, we significantly reduce the
				// set of circumstances where the listener definition itself changes, and this in
				// turn reduces the set of circumstances where envoy has to go through that drain
				// process and disrupt in-flight requests.
				rdsListener, routeConfigs, err := ListenerToRdsListener(lst)
				if err != nil {
					log.Errorf("Error converting listener to RDS: %+v", err)
					listeners = append(listeners, Clone(lst).(ctypes.Resource))
					continue
				}
				listeners = append(listeners, rdsListener)
				for _, rc := range routeConfigs {
					// These routes will get included in the configuration snapshot created below.
					routes = append(routes, rc)
				}
			}
			for _, cls := range sr.Clusters {
				clusters = append(clusters, Clone(cls).(ctypes.Resource))
			}
			continue
		case *v3clusterconfig.Cluster:
			dst = &clustersv3
		case *v3routeconfig.RouteConfiguration:
			dst = &routesv3
		case *v3listenerconfig.Listener:
			dst = &listenersv3
		case *v3runtime.Runtime:
			dst = &runtimesv3
		case *v3bootstrap.Bootstrap:
			bs := m.(*v3bootstrap.Bootstrap)
			sr := bs.StaticResources
			for _, lst := range sr.Listeners {
				// When the RouteConfiguration is embedded in the listener, it will cause envoy to
				// go through a complete drain cycle whenever there is a routing change and that
				// will potentially disrupt in-flight requests. By converting all listeners to use
				// RDS rather than inlining their routing configuration, we significantly reduce the
				// set of circumstances where the listener definition itself changes, and this in
				// turn reduces the set of circumstances where envoy has to go through that drain
				// process and disrupt in-flight requests.
				rdsListener, routeConfigs, err := V3ListenerToRdsListener(lst)
				if err != nil {
					log.Errorf("Error converting listener to RDS: %+v", err)
					listenersv3 = append(listenersv3, Clone(lst).(ctypes.Resource))
					continue
				}
				listenersv3 = append(listenersv3, rdsListener)
				for _, rc := range routeConfigs {
					// These routes will get included in the configuration snapshot created below.
					routesv3 = append(routesv3, rc)
				}
			}
			for _, cls := range sr.Clusters {
				clustersv3 = append(clustersv3, Clone(cls).(ctypes.Resource))
			}
			continue
		default:
			log.Warnf("Unrecognized resource %s: %v", name, e)
			continue
		}
		*dst = append(*dst, m.(ctypes.Resource))
	}

	if fastpathSnapshot != nil && fastpathSnapshot.Snapshot != nil {
		for _, lst := range fastpathSnapshot.Snapshot.Resources[types.Listener].Items {
			listeners = append(listeners, lst)
		}
		for _, route := range fastpathSnapshot.Snapshot.Resources[types.Route].Items {
			routes = append(routes, route)
		}
		for _, clu := range fastpathSnapshot.Snapshot.Resources[types.Cluster].Items {
			clusters = append(clusters, clu)
		}
		// We intentionally omit endpoints since those are carried separately.
	}

	// The configuration data that reaches us here arrives via two parallel paths that race each
	// other. The endpoint data comes in realtime directly from the golang watcher in the entrypoint
	// package. The cluster configuration comes from the python code. Either one can win which means
	// we might at times see endpoint data with no corresponding cluster and we might also see
	// clusters with no corresponding endpoint data. Both of these circumstances should be
	// transient.
	//
	// To produce a consistent configuration we do an outer join operation on the endpoint and
	// cluster configuration that we have at this moment. If there is no endpoint information for a
	// given cluster, we will synthesize an empty ClusterLoadAssignment.
	//
	// Note that a cluster not existing is very different to envoy than a cluster existing but
	// having an empty ClusterLoadAssignment. When envoy first discovers clusters it goes through a
	// warmup process to be sure the cluster is properly bootstrapped before routing traffic to
	// it. See here for more details:
	//
	// https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/cluster_manager.html?highlight=cluster%20warming
	//
	// For this reason if there is no endpoint data for the cluster we will synthesize an empty
	// ClusterLoadAssignment rather than filtering out the cluster. This avoids triggering the
	// warmup sequence in scenarios where the endpoint data for a cluster is really flapping into
	// and out of existence. In that circumstance we want to faithfully relay to envoy that the
	// cluster exists but currently has no endpoints.
	endpoints := JoinEdsClusters(ctx, clusters, edsEndpoints)
	endpointsv3 := JoinEdsClustersV3(ctx, clustersv3, edsEndpointsV3)

	// Create a new configuration snapshot from everything we have just loaded from disk.
	version := fmt.Sprintf("v%d", *generation)
	*generation++
	snapshot := v2cache.NewSnapshot(
		version,
		endpoints,
		clusters,
		routes,
		listeners,
		runtimes)

	if err := snapshot.Consistent(); err != nil {
		bs, _ := json.Marshal(snapshot)
		log.Errorf("V2 Snapshot inconsistency: %v: %s", err, bs)
		return
	}

	snapshotv3 := v3cache.NewSnapshot(
		version,
		endpointsv3,
		clustersv3,
		routesv3,
		listenersv3,
		runtimesv3)

	if err := snapshotv3.Consistent(); err != nil {
		bs, _ := json.Marshal(snapshotv3)
		log.Errorf("V3 Snapshot inconsistency: %v: %s", err, bs)
		return
	}

	// This used to just directly update envoy. Since we want ratelimiting, we now send an
	// Update object down the channel with a function that knows how to do the update if/when
	// the ratelimiting logic decides.
	//
	// We also need to pay attention to contexts here so we can shutdown properly. If we didn't
	// have the context portion, the ratelimit goroutine could shutdown first and we could end
	// up blocking here and never shutting down.
	select {
	case updates <- Update{version, func() error {
		err := config.SetSnapshot("test-id", snapshot)
		if err != nil {
			return fmt.Errorf("V2 Snapshot error %q for %+v", err, snapshot)
		}

		err = configv3.SetSnapshot("test-id", snapshotv3)
		if err != nil {
			return fmt.Errorf("V3 Snapshot error %q for %+v", err, snapshotv3)
		}

		return nil
	}}:
	case <-ctx.Done():
	}
}

func warn(err error) bool {
	if err != nil {
		log.Warn(err)
		return true
	} else {
		return false
	}
}

// OnStreamOpen is called once an xDS stream is open with a stream ID and the type URL (or "" for ADS).
func (l loggerv2) OnStreamOpen(_ context.Context, sid int64, stype string) error {
	l.Infof("V2 Stream open[%v]: %v", sid, stype)
	return nil
}

func (l loggerv3) OnStreamOpen(_ context.Context, sid int64, stype string) error {
	l.Infof("V3 Stream open[%v]: %v", sid, stype)
	return nil
}

// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
func (l loggerv2) OnStreamClosed(sid int64) {
	l.Infof("V2 Stream closed[%v]", sid)
}

func (l loggerv3) OnStreamClosed(sid int64) {
	l.Infof("V3 Stream closed[%v]", sid)
}

// OnStreamRequest is called once a request is received on a stream.
func (l loggerv2) OnStreamRequest(sid int64, req *v2.DiscoveryRequest) error {
	l.Infof("V2 Stream request[%v] for type %s: requesting %d resources", sid, req.TypeUrl, len(req.ResourceNames))
	l.Debugf("V2 Stream request[%v] dump: %v", sid, req)
	return nil
}

func (l loggerv3) OnStreamRequest(sid int64, req *v3discovery.DiscoveryRequest) error {
	l.Infof("V3 Stream request[%v] for type %s: requesting %d resources", sid, req.TypeUrl, len(req.ResourceNames))
	l.Debugf("V3 Stream request[%v] dump: %v", sid, req)
	return nil
}

// OnStreamResponse is called immediately prior to sending a response on a stream.
func (l loggerv2) OnStreamResponse(sid int64, req *v2.DiscoveryRequest, res *v2.DiscoveryResponse) {
	l.Infof("V2 Stream response[%v] for type %s: returning %d resources", sid, res.TypeUrl, len(res.Resources))
	l.Debugf("V2 Stream dump response[%v]: %v -> %v", sid, req, res)
}

func (l loggerv3) OnStreamResponse(sid int64, req *v3discovery.DiscoveryRequest, res *v3discovery.DiscoveryResponse) {
	l.Infof("V3 Stream response[%v] for type %s: returning %d resources", sid, res.TypeUrl, len(res.Resources))
	l.Debugf("V3 Stream dump response[%v]: %v -> %v", sid, req, res)
}

// OnFetchRequest is called for each Fetch request
func (l loggerv2) OnFetchRequest(_ context.Context, r *v2.DiscoveryRequest) error {
	l.Infof("V2 Fetch request: %v", r)
	return nil
}

func (l loggerv3) OnFetchRequest(_ context.Context, r *v3discovery.DiscoveryRequest) error {
	l.Infof("V3 Fetch request: %v", r)
	return nil
}

// OnFetchResponse is called immediately prior to sending a response.
func (l loggerv2) OnFetchResponse(req *v2.DiscoveryRequest, res *v2.DiscoveryResponse) {
	l.Infof("V2 Fetch response: %v -> %v", req, res)
}

func (l loggerv3) OnFetchResponse(req *v3discovery.DiscoveryRequest, res *v3discovery.DiscoveryResponse) {
	l.Infof("V3 Fetch response: %v -> %v", req, res)
}

func Main(ctx context.Context, Version string, rawArgs ...string) error {
	usage := memory.GetMemoryUsage()
	go usage.Watch(ctx)
	return Main2(ctx, Version, usage.PercentUsed, make(chan *FastpathSnapshot), rawArgs...)
}

func Main2(ctx context.Context, Version string, getUsage MemoryGetter, fastpathCh <-chan *FastpathSnapshot,
	rawArgs ...string) error {
	args, err := parseArgs(rawArgs...)
	if err != nil {
		return err
	}

	if args.debug {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.WarnLevel)
	}

	log.Infof("Ambex %s starting...", Version)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Panic()
	}
	defer watcher.Close()

	if args.watch {
		for _, d := range args.dirs {
			watcher.Add(d)
		}
	}

	// The golang signal package does not block when it writes to the channel. We therefore need a
	// nonzero buffer for the channel to minimize the possiblity that we miss out on a signal that
	// comes in while we are doing work and not reading from the channel. Since we are subscribing
	// to multiple signals there is also the possibility that even with buffering, too many of one
	// kind of signal can fill up the buffer and cause us to drop an occurance of the other types of
	// signal. To minimize the chance of that happening we will choose a buffer size of 100. That
	// may well be overkill, but better to not have to consider the possibility that we lose a
	// signal.
	ch := make(chan os.Signal, 100)
	signal.Notify(ch, syscall.SIGHUP, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	config := v2cache.NewSnapshotCache(true, HasherV2{}, log)
	configv3 := v3cache.NewSnapshotCache(true, HasherV3{}, logv3)
	server := v2server.NewServer(ctx, config, log)
	serverv3 := v3server.NewServer(ctx, configv3, logv3)

	runManagementServer(ctx, server, serverv3, args.adsNetwork, args.adsAddress)

	pid := os.Getpid()
	file := "ambex.pid"
	if !warn(ioutil.WriteFile(file, []byte(fmt.Sprintf("%v", pid)), 0644)) {
		log.WithFields(logrus.Fields{"pid": pid, "file": file}).Info("Wrote PID")
	}

	updates := make(chan Update)
	envoyUpdaterDone := make(chan struct{})
	go func() {
		defer close(envoyUpdaterDone)
		err := Updater(ctx, updates, getUsage)
		if err != nil {
			// Panic will get reported more usefully by entrypoint.go's exit code than logging the
			// error.
			panic(err)
		}
	}()

	generation := 0
	var fastpathSnapshot *FastpathSnapshot
	edsEndpoints := map[string]*v2.ClusterLoadAssignment{}
	edsEndpointsV3 := map[string]*v3endpointconfig.ClusterLoadAssignment{}
	update(ctx, config, configv3, &generation, args.dirs, edsEndpoints, edsEndpointsV3, fastpathSnapshot, updates)

OUTER:
	for {

		select {
		case sig := <-ch:
			switch sig {
			case syscall.SIGHUP:
				update(ctx, config, configv3, &generation, args.dirs, edsEndpoints, edsEndpointsV3, fastpathSnapshot, updates)
			case os.Interrupt, syscall.SIGTERM:
				break OUTER
			}
		case fpSnap := <-fastpathCh:
			if fpSnap.Endpoints != nil {
				edsEndpoints = fpSnap.Endpoints.ToMap_v2()
				edsEndpointsV3 = fpSnap.Endpoints.ToMap_v3()
			}
			fastpathSnapshot = fpSnap
			update(ctx, config, configv3, &generation, args.dirs, edsEndpoints, edsEndpointsV3, fastpathSnapshot, updates)
		case <-watcher.Events:
			update(ctx, config, configv3, &generation, args.dirs, edsEndpoints, edsEndpointsV3, fastpathSnapshot, updates)
		case err := <-watcher.Errors:
			log.WithError(err).Warn("Watcher error")
		case <-ctx.Done():
			break OUTER
		}

	}

	<-envoyUpdaterDone
	log.Info("Done")
	return nil
}
