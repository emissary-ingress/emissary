package ambex

/**********************************************
 * ambex: Ambassador Experimental ADS server
 *
 * Here's the deal.
 *
 * go-control-plane, several different classes manage this stuff:
 *
 * - The root of the world is a SnapshotCache.
 *   - import github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2, then refer
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
 *   - import github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server, then refer
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
	// standard library
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	// third-party libraries
	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	// envoy control plane
	ecp_cache_types "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/types"
	ecp_v2_cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
	ecp_v3_cache "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v3"
	ecp_log "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/log"
	ecp_v2_server "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/v2"
	ecp_v3_server "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/v3"

	// Envoy API v2
	// Be sure to import the package of any types that're referenced with "@type" in our
	// generated Envoy config, even if that package is otherwise not used by ambex.
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
	// Be sure to import the package of any types that're referenced with "@type" in our
	// generated Envoy config, even if that package is otherwise not used by ambex.
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/config/accesslog/v3"
	v3bootstrap "github.com/datawire/ambassador/v2/pkg/api/envoy/config/bootstrap/v3"
	v3clusterconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/config/cluster/v3"
	v3core "github.com/datawire/ambassador/v2/pkg/api/envoy/config/core/v3"
	v3endpointconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/config/endpoint/v3"
	v3listenerconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/config/listener/v3"
	v3routeconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/config/route/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/access_loggers/file/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/access_loggers/grpc/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/compression/gzip/compressor/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/buffer/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/compressor/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/ext_authz/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/grpc_stats/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/gzip/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/lua/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/ratelimit/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/rbac/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/response_map/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/http/router/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/filters/network/tcp_proxy/v3"
	v3cluster "github.com/datawire/ambassador/v2/pkg/api/envoy/service/cluster/v3"
	v3discovery "github.com/datawire/ambassador/v2/pkg/api/envoy/service/discovery/v3"
	v3endpoint "github.com/datawire/ambassador/v2/pkg/api/envoy/service/endpoint/v3"
	v3listener "github.com/datawire/ambassador/v2/pkg/api/envoy/service/listener/v3"
	v3route "github.com/datawire/ambassador/v2/pkg/api/envoy/service/route/v3"
	v3runtime "github.com/datawire/ambassador/v2/pkg/api/envoy/service/runtime/v3"

	// first-party libraries
	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
)

type Args struct {
	watch bool

	adsNetwork string
	adsAddress string

	dirs []string

	snapdirPath string
	numsnaps    int
}

func parseArgs(ctx context.Context, rawArgs ...string) (*Args, error) {
	var args Args
	flagset := flag.NewFlagSet("ambex", flag.ContinueOnError)

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

	// ambex logs its own snapshots, separately from the ones provided by the Python
	// side of the world, in $rootdir/snapshots/ambex-#.json, where rootdir is taken
	// from $AMBASSADOR_CONFIG_BASE_DIR if set, else $ambassador_root if set, else
	// whatever, set rootdir to /ambassador.
	snapdirPath := os.Getenv("AMBASSADOR_CONFIG_BASE_DIR")
	if snapdirPath == "" {
		snapdirPath = os.Getenv("ambassador_root")
	}
	if snapdirPath == "" {
		snapdirPath = "/ambassador"
	}
	args.snapdirPath = path.Join(snapdirPath, "snapshots")

	// We'll keep $AMBASSADOR_AMBEX_SNAPSHOT_COUNT snapshots. If unset, or set to
	// something we can't treat as an int, use 30 (which Flynn just made up, so don't
	// be afraid to change it if need be).
	numsnapStr := os.Getenv("AMBASSADOR_AMBEX_SNAPSHOT_COUNT")
	if numsnapStr == "" {
		numsnapStr = "30"
	}
	var err error
	args.numsnaps, err = strconv.Atoi(numsnapStr)
	if (err != nil) || (args.numsnaps < 0) {
		args.numsnaps = 30
		dlog.Errorf(ctx, "Invalid AMBASSADOR_AMBEX_SNAPSHOT_COUNT: %s, using %d", numsnapStr, args.numsnaps)
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

// run stuff
// RunManagementServer starts an xDS server at the given port.
func runManagementServer(ctx context.Context, server ecp_v2_server.Server, serverv3 ecp_v3_server.Server, adsNetwork, adsAddress string) error {
	grpcServer := grpc.NewServer()

	lis, err := net.Listen(adsNetwork, adsAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
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

	dlog.Infof(ctx, "Listening on %s:%s", adsNetwork, adsAddress)
	go func() {
		sc := &dhttp.ServerConfig{
			Handler: grpcServer,
		}
		if err := sc.Serve(ctx, lis); err != nil {
			dlog.Errorf(ctx, "Management server exited: %v", err)
		}
	}()
	return nil
}

// Decoders for unmarshalling our config
var decoders = map[string](func([]byte, proto.Message) error){
	".json": protojson.Unmarshal,
	".pb":   prototext.Unmarshal,
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

func Decode(ctx context.Context, name string) (proto.Message, error) {
	any := &anypb.Any{}
	contents, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(name)
	decoder := decoders[ext]
	err = decoder(contents, any)
	if err != nil {
		return nil, err
	}

	m, err := any.UnmarshalNew()
	if err != nil {
		return nil, err
	}

	v := m.(Validatable)

	if err := v.Validate(); err != nil {
		return nil, err
	}
	dlog.Infof(ctx, "Loaded file %s", name)
	return v, nil
}

// Observability:
//
// These "expanded" snapshots make the snapshots we log easier to read: basically,
// instead of just indexing by Golang types, make the JSON marshal with real names.
type v2ExpandedSnapshot struct {
	Endpoints ecp_v2_cache.Resources `json:"endpoints"`
	Clusters  ecp_v2_cache.Resources `json:"clusters"`
	Routes    ecp_v2_cache.Resources `json:"routes"`
	Listeners ecp_v2_cache.Resources `json:"listeners"`
	Runtimes  ecp_v2_cache.Resources `json:"runtimes"`
}

func NewV2ExpandedSnapshot(v2snap *ecp_v2_cache.Snapshot) v2ExpandedSnapshot {
	return v2ExpandedSnapshot{
		Endpoints: v2snap.Resources[ecp_cache_types.Endpoint],
		Clusters:  v2snap.Resources[ecp_cache_types.Cluster],
		Routes:    v2snap.Resources[ecp_cache_types.Route],
		Listeners: v2snap.Resources[ecp_cache_types.Listener],
		Runtimes:  v2snap.Resources[ecp_cache_types.Runtime],
	}
}

type v3ExpandedSnapshot struct {
	Endpoints ecp_v3_cache.Resources `json:"endpoints"`
	Clusters  ecp_v3_cache.Resources `json:"clusters"`
	Routes    ecp_v3_cache.Resources `json:"routes"`
	Listeners ecp_v3_cache.Resources `json:"listeners"`
	Runtimes  ecp_v3_cache.Resources `json:"runtimes"`
}

func NewV3ExpandedSnapshot(v3snap *ecp_v3_cache.Snapshot) v3ExpandedSnapshot {
	return v3ExpandedSnapshot{
		Endpoints: v3snap.Resources[ecp_cache_types.Endpoint],
		Clusters:  v3snap.Resources[ecp_cache_types.Cluster],
		Routes:    v3snap.Resources[ecp_cache_types.Route],
		Listeners: v3snap.Resources[ecp_cache_types.Listener],
		Runtimes:  v3snap.Resources[ecp_cache_types.Runtime],
	}
}

// A combinedSnapshot has both a V2 and V3 snapshot, for logging.
type combinedSnapshot struct {
	Version string             `json:"version"`
	V2      v2ExpandedSnapshot `json:"v2"`
	V3      v3ExpandedSnapshot `json:"v3"`
}

// csDump creates a combinedSnapshot from a V2 snapshot and a V3 snapshot, then
// dumps the combinedSnapshot to disk. Only numsnaps snapshots are kept: ambex-1.json
// is the newest, then ambex-2.json, etc., so ambex-$numsnaps.json is the oldest.
// Every time we write a new one, we rename all the older ones, ditching the oldest
// after we've written numsnaps snapshots.
func csDump(ctx context.Context, snapdirPath string, numsnaps int, generation int, v2snap *ecp_v2_cache.Snapshot, v3snap *ecp_v3_cache.Snapshot) {
	if numsnaps <= 0 {
		// Don't do snapshotting at all.
		return
	}

	// OK, they want snapshots. Make a proper version string...
	version := fmt.Sprintf("v%d", generation)

	// ...and a combinedSnapshot.
	cs := combinedSnapshot{
		Version: version,
		V2:      NewV2ExpandedSnapshot(v2snap),
		V3:      NewV3ExpandedSnapshot(v3snap),
	}

	// Next up, marshal as JSON and write to ambex-0.json. Note that we
	// didn't say anything about a -0 file; that's because it's about to
	// be renamed.

	bs, err := json.MarshalIndent(cs, "", "  ")

	if err != nil {
		dlog.Errorf(ctx, "CSNAP: marshal failure: %s", err)
		return
	}

	csPath := path.Join(snapdirPath, "ambex-0.json")

	err = ioutil.WriteFile(csPath, bs, 0644)

	if err != nil {
		dlog.Errorf(ctx, "CSNAP: write failure: %s", err)
	} else {
		dlog.Infof(ctx, "Saved snapshot %s", version)
	}

	// Rotate everything one file down. This includes renaming the just-written
	// ambex-0 to ambex-1.
	for i := numsnaps; i > 0; i-- {
		previous := i - 1

		fromPath := path.Join(snapdirPath, fmt.Sprintf("ambex-%d.json", previous))
		toPath := path.Join(snapdirPath, fmt.Sprintf("ambex-%d.json", i))

		err := os.Rename(fromPath, toPath)

		if (err != nil) && !os.IsNotExist(err) {
			dlog.Infof(ctx, "CSNAP: could not rename %s -> %s: %#v", fromPath, toPath, err)
		}
	}
}

// Get an updated snapshot going.
func update(
	ctx context.Context,
	snapdirPath string,
	numsnaps int,
	config ecp_v2_cache.SnapshotCache,
	configv3 ecp_v3_cache.SnapshotCache,
	generation *int,
	dirs []string,
	edsEndpoints map[string]*v2.ClusterLoadAssignment,
	edsEndpointsV3 map[string]*v3endpointconfig.ClusterLoadAssignment,
	fastpathSnapshot *FastpathSnapshot,
	updates chan<- Update,
) error {
	clusters := []ecp_cache_types.Resource{}  // v2.Cluster
	routes := []ecp_cache_types.Resource{}    // v2.RouteConfiguration
	listeners := []ecp_cache_types.Resource{} // v2.Listener
	runtimes := []ecp_cache_types.Resource{}  // discovery.Runtime

	clustersv3 := []ecp_cache_types.Resource{}  // v3.Cluster
	routesv3 := []ecp_cache_types.Resource{}    // v3.RouteConfiguration
	listenersv3 := []ecp_cache_types.Resource{} // v3.Listener
	runtimesv3 := []ecp_cache_types.Resource{}  // v3.Runtime

	var filenames []string

	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			dlog.Warnf(ctx, "Error listing %q: %v", dir, err)
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
		m, e := Decode(ctx, name)
		if e != nil {
			dlog.Warnf(ctx, "%s: %v", name, e)
			continue
		}
		var dst *[]ecp_cache_types.Resource
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
					dlog.Errorf(ctx, "Error converting listener to RDS: %+v", err)
					listeners = append(listeners, proto.Clone(lst).(ecp_cache_types.Resource))
					continue
				}
				listeners = append(listeners, rdsListener)
				for _, rc := range routeConfigs {
					// These routes will get included in the configuration snapshot created below.
					routes = append(routes, rc)
				}
			}
			for _, cls := range sr.Clusters {
				clusters = append(clusters, proto.Clone(cls).(ecp_cache_types.Resource))
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
					dlog.Errorf(ctx, "Error converting listener to RDS: %+v", err)
					listenersv3 = append(listenersv3, proto.Clone(lst).(ecp_cache_types.Resource))
					continue
				}
				listenersv3 = append(listenersv3, rdsListener)
				for _, rc := range routeConfigs {
					// These routes will get included in the configuration snapshot created below.
					routesv3 = append(routesv3, rc)
				}
			}
			for _, cls := range sr.Clusters {
				clustersv3 = append(clustersv3, proto.Clone(cls).(ecp_cache_types.Resource))
			}
			continue
		default:
			dlog.Warnf(ctx, "Unrecognized resource %s: %v", name, e)
			continue
		}
		*dst = append(*dst, m.(ecp_cache_types.Resource))
	}

	if fastpathSnapshot != nil && fastpathSnapshot.Snapshot != nil {
		for _, lst := range fastpathSnapshot.Snapshot.Resources[ecp_cache_types.Listener].Items {
			listeners = append(listeners, lst)
		}
		for _, route := range fastpathSnapshot.Snapshot.Resources[ecp_cache_types.Route].Items {
			routes = append(routes, route)
		}
		for _, clu := range fastpathSnapshot.Snapshot.Resources[ecp_cache_types.Cluster].Items {
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
	curgen := *generation
	*generation++

	version := fmt.Sprintf("v%d", curgen)
	snapshot := ecp_v2_cache.NewSnapshot(
		version,
		endpoints,
		clusters,
		routes,
		listeners,
		runtimes,
		nil, // secrets
	)

	if err := snapshot.Consistent(); err != nil {
		bs, _ := json.Marshal(snapshot)
		dlog.Errorf(ctx, "V2 Snapshot inconsistency: %v: %s", err, bs)
		return nil // TODO: should we return the error, rather than just logging it?
	}

	snapshotv3 := ecp_v3_cache.NewSnapshot(
		version,
		endpointsv3,
		clustersv3,
		routesv3,
		listenersv3,
		runtimesv3,
		nil, // secrets
	)

	if err := snapshotv3.Consistent(); err != nil {
		bs, _ := json.Marshal(snapshotv3)
		dlog.Errorf(ctx, "V3 Snapshot inconsistency: %v: %s", err, bs)
		return nil // TODO: should we return the error, rather than just logging it?
	}

	// This used to just directly update envoy. Since we want ratelimiting, we now send an
	// Update object down the channel with a function that knows how to do the update if/when
	// the ratelimiting logic decides.

	dlog.Debugf(ctx, "Created snapshot %s", version)
	csDump(ctx, snapdirPath, numsnaps, curgen, &snapshot, &snapshotv3)

	update := Update{version, func() error {
		dlog.Debugf(ctx, "Accepting snapshot %s", version)

		err := config.SetSnapshot("test-id", snapshot)
		if err != nil {
			return fmt.Errorf("V2 Snapshot error %q for %+v", err, snapshot)
		}

		err = configv3.SetSnapshot("test-id", snapshotv3)
		if err != nil {
			return fmt.Errorf("V3 Snapshot error %q for %+v", err, snapshotv3)
		}

		return nil
	}}

	// We also need to pay attention to contexts here so we can shutdown properly. If we didn't
	// have the context portion, the ratelimit goroutine could shutdown first and we could end
	// up blocking here and never shutting down.
	select {
	case updates <- update:
	case <-ctx.Done():
	}
	return nil
}

type logAdapterBase struct {
	prefix string
}

type logAdapterV2 struct {
	logAdapterBase
}

var _ ecp_v2_server.Callbacks = logAdapterV2{}
var _ ecp_log.Logger = logAdapterV2{}

type logAdapterV3 struct {
	logAdapterBase
}

var _ ecp_v3_server.Callbacks = logAdapterV3{}
var _ ecp_log.Logger = logAdapterV3{}

// Debugf implements ecp_log.Logger.
func (l logAdapterBase) Debugf(format string, args ...interface{}) {
	dlog.Debugf(context.TODO(), format, args...)
}

// Infof implements ecp_log.Logger.
func (l logAdapterBase) Infof(format string, args ...interface{}) {
	dlog.Infof(context.TODO(), format, args...)
}

// Warnf implements ecp_log.Logger.
func (l logAdapterBase) Warnf(format string, args ...interface{}) {
	dlog.Warnf(context.TODO(), format, args...)
}

// Errorf implements ecp_log.Logger.
func (l logAdapterBase) Errorf(format string, args ...interface{}) {
	dlog.Errorf(context.TODO(), format, args...)
}

// OnStreamOpen implements ecp_v2_server.Callbacks and ecp_v3_server.Callbacks.
func (l logAdapterBase) OnStreamOpen(ctx context.Context, sid int64, stype string) error {
	dlog.Debugf(ctx, "%v Stream open[%v]: %v", l.prefix, sid, stype)
	return nil
}

// OnStreamClosed implements ecp_v2_server.Callbacks and ecp_v3_server.Callbacks.
func (l logAdapterBase) OnStreamClosed(sid int64) {
	dlog.Debugf(context.TODO(), "%v Stream closed[%v]", l.prefix, sid)
}

// OnStreamRequest implements ecp_v2_server.Callbacks.
func (l logAdapterV2) OnStreamRequest(sid int64, req *v2.DiscoveryRequest) error {
	dlog.Debugf(context.TODO(), "V2 Stream request[%v] for type %s: requesting %d resources", sid, req.TypeUrl, len(req.ResourceNames))
	dlog.Debugf(context.TODO(), "V2 Stream request[%v] dump: %v", sid, req)
	return nil
}

// OnStreamRequest implements ecp_v3_server.Callbacks.
func (l logAdapterV3) OnStreamRequest(sid int64, req *v3discovery.DiscoveryRequest) error {
	dlog.Debugf(context.TODO(), "V3 Stream request[%v] for type %s: requesting %d resources", sid, req.TypeUrl, len(req.ResourceNames))
	dlog.Debugf(context.TODO(), "V3 Stream request[%v] dump: %v", sid, req)
	return nil
}

// OnStreamResponse implements ecp_v2_server.Callbacks.
func (l logAdapterV2) OnStreamResponse(sid int64, req *v2.DiscoveryRequest, res *v2.DiscoveryResponse) {
	dlog.Debugf(context.TODO(), "V2 Stream response[%v] for type %s: returning %d resources", sid, res.TypeUrl, len(res.Resources))
	dlog.Debugf(context.TODO(), "V2 Stream dump response[%v]: %v -> %v", sid, req, res)
}

// OnStreamResponse implements ecp_v3_server.Callbacks.
func (l logAdapterV3) OnStreamResponse(sid int64, req *v3discovery.DiscoveryRequest, res *v3discovery.DiscoveryResponse) {
	dlog.Debugf(context.TODO(), "V3 Stream response[%v] for type %s: returning %d resources", sid, res.TypeUrl, len(res.Resources))
	dlog.Debugf(context.TODO(), "V3 Stream dump response[%v]: %v -> %v", sid, req, res)
}

// OnFetchRequest implements ecp_v2_server.Callbacks.
func (l logAdapterV2) OnFetchRequest(ctx context.Context, r *v2.DiscoveryRequest) error {
	dlog.Debugf(ctx, "V2 Fetch request: %v", r)
	return nil
}

// OnFetchRequest implements ecp_v3_server.Callbacks.
func (l logAdapterV3) OnFetchRequest(ctx context.Context, r *v3discovery.DiscoveryRequest) error {
	dlog.Debugf(ctx, "V3 Fetch request: %v", r)
	return nil
}

// OnFetchResponse implements ecp_v2_server.Callbacks.
func (l logAdapterV2) OnFetchResponse(req *v2.DiscoveryRequest, res *v2.DiscoveryResponse) {
	dlog.Debugf(context.TODO(), "V2 Fetch response: %v -> %v", req, res)
}

// OnFetchResponse implements ecp_v3_server.Callbacks.
func (l logAdapterV3) OnFetchResponse(req *v3discovery.DiscoveryRequest, res *v3discovery.DiscoveryResponse) {
	dlog.Debugf(context.TODO(), "V3 Fetch response: %v -> %v", req, res)
}

func Main(
	ctx context.Context,
	Version string,
	getUsage MemoryGetter,
	fastpathCh <-chan *FastpathSnapshot,
	rawArgs ...string,
) error {
	args, err := parseArgs(ctx, rawArgs...)
	if err != nil {
		return err
	}

	dlog.Infof(ctx, "Ambex %s starting, snapdirPath %s", Version, args.snapdirPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if args.watch {
		for _, d := range args.dirs {
			if err := watcher.Add(d); err != nil {
				return err
			}
		}
	}

	// The golang signal package does not block when it writes to the channel. We therefore need a
	// nonzero buffer for the channel to minimize the possiblity that we miss out on a signal that
	// comes in while we are doing work and not reading from the channel. To minimize the chance
	// of that happening we will choose a buffer size of 100. That may well be overkill, but
	// better to not have to consider the possibility that we lose a signal.
	sigCh := make(chan os.Signal, 100)
	signal.Notify(sigCh, syscall.SIGHUP)
	defer func() { signal.Stop(sigCh) }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	config := ecp_v2_cache.NewSnapshotCache(true, HasherV2{}, logAdapterV2{logAdapterBase{"V2"}})
	configv3 := ecp_v3_cache.NewSnapshotCache(true, HasherV3{}, logAdapterV3{logAdapterBase{"V3"}})
	server := ecp_v2_server.NewServer(ctx, config, logAdapterV2{logAdapterBase{"V2"}})
	serverv3 := ecp_v3_server.NewServer(ctx, configv3, logAdapterV3{logAdapterBase{"V3"}})

	if err := runManagementServer(ctx, server, serverv3, args.adsNetwork, args.adsAddress); err != nil {
		return err
	}

	pid := os.Getpid()
	file := "ambex.pid"
	if err := ioutil.WriteFile(file, []byte(fmt.Sprintf("%v", pid)), 0644); err != nil {
		dlog.Warn(ctx, err)
	} else {
		ctx := dlog.WithField(ctx, "pid", pid)
		ctx = dlog.WithField(ctx, "file", file)
		dlog.Info(ctx, "Wrote PID")
	}

	updates := make(chan Update)
	envoyUpdaterDone := make(chan struct{})
	go func() {
		defer close(envoyUpdaterDone)
		if err := Updater(ctx, updates, getUsage); err != nil {
			panic(err) // TODO: Find a better way of reporting errors from goroutines.
		}
	}()

	generation := 0
	var fastpathSnapshot *FastpathSnapshot
	edsEndpoints := map[string]*v2.ClusterLoadAssignment{}
	edsEndpointsV3 := map[string]*v3endpointconfig.ClusterLoadAssignment{}

	// We always start by updating with a totally empty snapshot.
	//
	// XXX This seems questionable: why do we do this? Envoy isn't currently started until
	// we have a real configuration...
	err = update(
		ctx,
		args.snapdirPath,
		args.numsnaps,
		config,
		configv3,
		&generation,
		args.dirs,
		edsEndpoints,
		edsEndpointsV3,
		fastpathSnapshot,
		updates,
	)
	if err != nil {
		return err
	}

	// This is the main loop where the magic happens. The fact that it uses a label
	// depresses me, though.
OUTER:
	for {

		select {
		case _ = <-sigCh:
			err := update(
				ctx,
				args.snapdirPath,
				args.numsnaps,
				config,
				configv3,
				&generation,
				args.dirs,
				edsEndpoints,
				edsEndpointsV3,
				fastpathSnapshot,
				updates,
			)
			if err != nil {
				return err
			}
		case fpSnap := <-fastpathCh:
			// Fastpath update. Grab new endpoints and update.
			if fpSnap.Endpoints != nil {
				edsEndpoints = fpSnap.Endpoints.ToMap_v2()
				edsEndpointsV3 = fpSnap.Endpoints.ToMap_v3()
			}
			fastpathSnapshot = fpSnap
			err := update(
				ctx,
				args.snapdirPath,
				args.numsnaps,
				config,
				configv3,
				&generation,
				args.dirs,
				edsEndpoints,
				edsEndpointsV3,
				fastpathSnapshot,
				updates,
			)
			if err != nil {
				return err
			}
		case <-watcher.Events:
			// Non-fastpath update. Just update.
			err := update(
				ctx,
				args.snapdirPath,
				args.numsnaps,
				config,
				configv3,
				&generation,
				args.dirs,
				edsEndpoints,
				edsEndpointsV3,
				fastpathSnapshot,
				updates,
			)
			if err != nil {
				return err
			}
		case err := <-watcher.Errors:
			// Something went wrong, so scream about that.
			dlog.Warnf(ctx, "Watcher error: %v", err)
		case <-ctx.Done():
			break OUTER
		}

	}

	<-envoyUpdaterDone
	dlog.Info(ctx, "Done")
	return nil
}
