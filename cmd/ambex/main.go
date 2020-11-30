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
	ctypes "github.com/datawire/ambassador/pkg/envoy-control-plane/cache/types"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/cache/v2"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/server/v2"
	"github.com/datawire/ambassador/pkg/memory"

	// envoy protobuf -- Be sure to import the package of any types that the Python
	// emits a "@type" of in the generated config, even if that package is otherwise
	// not used by ambex.
	v2 "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/api/v2/auth"
	core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/accesslog/v2"
	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/filter/http/buffer/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/filter/http/ext_authz/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/filter/http/gzip/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/filter/http/lua/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/filter/http/rate_limit/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/filter/http/rbac/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/filter/http/router/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/filter/network/http_connection_manager/v2"
	_ "github.com/datawire/ambassador/pkg/api/envoy/config/filter/network/tcp_proxy/v2"
	discovery "github.com/datawire/ambassador/pkg/api/envoy/service/discovery/v2"

	// envoy protobuf v3
	_ "github.com/datawire/ambassador/pkg/api/envoy/extensions/filters/http/response_map/v3"
)

const (
	localhost = "127.0.0.1"
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
type Hasher struct {
}

// ID function
func (h Hasher) ID(node *core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

// end Hasher stuff

// This feels kinda dumb.
type logger struct {
	*logrus.Logger
}

var log = &logger{
	Logger: logrus.StandardLogger(),
}

// run stuff
// RunManagementServer starts an xDS server at the given port.
func runManagementServer(ctx context.Context, server server.Server, adsNetwork, adsAddress string) {
	grpcServer := grpc.NewServer()

	lis, err := net.Listen(adsNetwork, adsAddress)
	if err != nil {
		log.WithError(err).Panic("failed to listen")
	}

	// register services
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	log.WithFields(logrus.Fields{"addr": adsNetwork + ":" + adsAddress}).Info("Listening")
	go func() {
		go func() {
			err := grpcServer.Serve(lis)

			if err != nil {
				log.WithFields(logrus.Fields{"error": err}).Error("Management server exited")
			}
		}()

		<-ctx.Done()
		grpcServer.GracefulStop()
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

func decode(name string) (proto.Message, error) {
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

func update(ctx context.Context, config cache.SnapshotCache, generation *int, dirs []string, updates chan<- Update) {
	clusters := []ctypes.Resource{}  // v2.Cluster
	endpoints := []ctypes.Resource{} // v2.ClusterLoadAssignment
	routes := []ctypes.Resource{}    // v2.RouteConfiguration
	listeners := []ctypes.Resource{} // v2.Listener
	runtimes := []ctypes.Resource{}  // discovery.Runtime

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
		m, e := decode(name)
		if e != nil {
			log.Warnf("%s: %v", name, e)
			continue
		}
		var dst *[]ctypes.Resource
		switch m.(type) {
		case *v2.Cluster:
			dst = &clusters
		case *v2.ClusterLoadAssignment:
			dst = &endpoints
		case *v2.RouteConfiguration:
			dst = &routes
		case *v2.Listener:
			dst = &listeners
		case *discovery.Runtime:
			dst = &runtimes
		case *bootstrap.Bootstrap:
			bs := m.(*bootstrap.Bootstrap)
			sr := bs.StaticResources
			for _, lst := range sr.Listeners {
				listeners = append(listeners, Clone(lst).(ctypes.Resource))
			}
			for _, cls := range sr.Clusters {
				clusters = append(clusters, Clone(cls).(ctypes.Resource))
			}
			continue
		default:
			log.Warnf("Unrecognized resource %s: %v", name, e)
			continue
		}
		*dst = append(*dst, m.(ctypes.Resource))
	}

	version := fmt.Sprintf("v%d", *generation)
	*generation++
	snapshot := cache.NewSnapshot(
		version,
		endpoints,
		clusters,
		routes,
		listeners,
		runtimes)

	err := snapshot.Consistent()

	if err != nil {
		log.Errorf("Snapshot inconsistency: %+v", snapshot)
	} else {
		// This used to just directly update envoy. Since we want ratelimiting, we now send an
		// Update object down the channel with a fuction that knows how to do the update if/when the
		// ratelimiting logic decides.
		//
		// We also need to pay attention to contexts here so we can shutdown properly. If we didn't
		// have the context portion, the ratelimit goroutine could shutdown first and we could end
		// up blocking here and never shutting down.
		select {
		case updates <- Update{version, func() error {
			err := config.SetSnapshot("test-id", snapshot)
			if err != nil {
				return fmt.Errorf("Snapshot error %q for %+v", err, snapshot)
			} else {
				return nil
			}
		}}:
		case <-ctx.Done():
		}
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
func (l logger) OnStreamOpen(_ context.Context, sid int64, stype string) error {
	l.Infof("Stream open[%v]: %v", sid, stype)
	return nil
}

// OnStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
func (l logger) OnStreamClosed(sid int64) {
	l.Infof("Stream closed[%v]", sid)
}

// OnStreamRequest is called once a request is received on a stream.
func (l logger) OnStreamRequest(sid int64, req *v2.DiscoveryRequest) error {
	l.Infof("Stream request[%v]: %v", sid, req)
	return nil
}

// OnStreamResponse is called immediately prior to sending a response on a stream.
func (l logger) OnStreamResponse(sid int64, req *v2.DiscoveryRequest, res *v2.DiscoveryResponse) {
	l.Infof("Stream response[%v]: %v -> %v", sid, req, res)
}

// OnFetchRequest is called for each Fetch request
func (l logger) OnFetchRequest(_ context.Context, r *v2.DiscoveryRequest) error {
	l.Infof("Fetch request: %v", r)
	return nil
}

// OnFetchResponse is called immediately prior to sending a response.
func (l logger) OnFetchResponse(req *v2.DiscoveryRequest, res *v2.DiscoveryResponse) {
	l.Infof("Fetch response: %v -> %v", req, res)
}

func Main(ctx context.Context, Version string, rawArgs ...string) error {
	usage := memory.GetMemoryUsage()
	go usage.Watch(ctx)
	return Main2(ctx, Version, usage.PercentUsed, rawArgs...)
}

func Main2(ctx context.Context, Version string, getUsage MemoryGetter, rawArgs ...string) error {
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

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGHUP, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	config := cache.NewSnapshotCache(true, Hasher{}, log)
	srv := server.NewServer(ctx, config, log)

	runManagementServer(ctx, srv, args.adsNetwork, args.adsAddress)

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
	update(ctx, config, &generation, args.dirs, updates)

OUTER:
	for {

		select {
		case sig := <-ch:
			switch sig {
			case syscall.SIGHUP:
				update(ctx, config, &generation, args.dirs, updates)
			case os.Interrupt, syscall.SIGTERM:
				break OUTER
			}
		case <-watcher.Events:
			update(ctx, config, &generation, args.dirs, updates)
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
