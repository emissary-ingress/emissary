package main

/**********************************************
 * ambex: Ambassador Experimental ADS server
 *
 * Here's the deal.
 *
 * go-control-plane, several different classes manage this stuff:
 *
 * - The root of the world is a SnapshotCache.
 *   - import github.com/datawire/ambassador/pkg/envoy-control-plane/cache, then refer
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

	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"

	v2 "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/cache"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/server"

	bootstrap "github.com/datawire/ambassador/pkg/api/envoy/config/bootstrap/v2"
	discovery "github.com/datawire/ambassador/pkg/api/envoy/service/discovery/v2"

	"github.com/fsnotify/fsnotify"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

const (
	localhost = "127.0.0.1"
)

var (
	debug   bool
	adsPort uint
	watch   bool

	// Version is inserted at build using --ldflags -X
	Version = "-no-version-"
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Use debug logging")
	flag.UintVar(&adsPort, "ads", 18000, "ADS port")
	flag.BoolVar(&watch, "watch", false, "Watch for file changes")
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
type logger struct{}

func (logger logger) Infof(format string, args ...interface{}) {
	log.Debugf(format, args...)
}
func (logger logger) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// end logger stuff

// run stuff
// RunManagementServer starts an xDS server at the given port.
func runManagementServer(ctx context.Context, server server.Server, port uint) {
	grpcServer := grpc.NewServer()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.WithError(err).Fatal("failed to listen")
	}

	// register services
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	log.WithFields(log.Fields{"port": port}).Info("Listening")
	go func() {
		go func() {
			err := grpcServer.Serve(lis)

			if err != nil {
				log.WithFields(log.Fields{"error": err}).Error("Management server exited")
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
	any := &types.Any{}
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

	var m types.DynamicAny
	err = types.UnmarshalAny(any, &m)
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

func update(config cache.SnapshotCache, generation *int, dirs []string) {
	clusters := []cache.Resource{}  // v2.Cluster
	endpoints := []cache.Resource{} // v2.ClusterLoadAssignment
	routes := []cache.Resource{}    // v2.RouteConfiguration
	listeners := []cache.Resource{} // v2.Listener

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
		var dst *[]cache.Resource
		switch m.(type) {
		case *v2.Cluster:
			dst = &clusters
		case *v2.ClusterLoadAssignment:
			dst = &endpoints
		case *v2.RouteConfiguration:
			dst = &routes
		case *v2.Listener:
			dst = &listeners
		case *bootstrap.Bootstrap:
			bs := m.(*bootstrap.Bootstrap)
			sr := bs.StaticResources
			for _, lst := range sr.Listeners {
				listeners = append(listeners, Clone(lst).(cache.Resource))
			}
			for _, cls := range sr.Clusters {
				clusters = append(clusters, Clone(cls).(cache.Resource))
			}
			continue
		default:
			log.Warnf("Unrecognized resource %s: %v", name, e)
			continue
		}
		*dst = append(*dst, m.(cache.Resource))
	}

	version := fmt.Sprintf("v%d", *generation)
	*generation++
	snapshot := cache.NewSnapshot(version, endpoints, clusters, routes, listeners)

	err := snapshot.Consistent()

	if err != nil {
		log.Errorf("Snapshot inconsistency: %+v", snapshot)
	} else {
		err = config.SetSnapshot("test-id", snapshot)
	}

	if err != nil {
		log.Fatalf("Snapshot error %q for %+v", err, snapshot)
	} else {
		// log.Infof("Snapshot %+v", snapshot)
		log.Infof("Pushing snapshot %+v", version)
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

func main() {
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Infof("Ambex %s starting...", Version)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Fatal()
	}
	defer watcher.Close()

	dirs := flag.Args()

	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	if watch {
		for _, d := range dirs {
			watcher.Add(d)
		}
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGHUP, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := cache.NewSnapshotCache(true, Hasher{}, logger{})
	srv := server.NewServer(config, logger{})

	runManagementServer(ctx, srv, adsPort)

	pid := os.Getpid()
	file := "ambex.pid"
	if !warn(ioutil.WriteFile(file, []byte(fmt.Sprintf("%v", pid)), 0644)) {
		log.WithFields(log.Fields{"pid": pid, "file": file}).Info("Wrote PID")
	}

	generation := 0
	update(config, &generation, dirs)

OUTER:
	for {

		select {
		case sig := <-ch:
			switch sig {
			case syscall.SIGHUP:
				update(config, &generation, dirs)
			case os.Interrupt, syscall.SIGTERM:
				break OUTER
			}
		case <-watcher.Events:
			update(config, &generation, dirs)
		case err := <-watcher.Errors:
			log.WithError(err).Warn("Watcher error")
		}

	}

	log.Info("Done")
}
