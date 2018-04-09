package main

/**********************************************
 * ambex: Ambassador Experimental ADS server
 *
 * This really is about as awful as it looks. There are multiple different
 * sorts of things going on here; it should really be four or five different
 * packages. Probably.
 *
 * Here's the deal.
 *
 * go-control-plane, several different classes manage this stuff:
 *
 * - The root of the world is a SnapshotCache.
 *   - import github.com/envoyproxy/go-control-plane/pkg/cache, then refer
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
 *   - import github.com/envoyproxy/go-control-plane/pkg/server, then refer
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
 * - We manage the SnapshotCache using our configurator object, which knows
 *   how to listen to stdin and for HTTP requests to change objects and post
 *   new Snapshots.
 *   - Obviously this piece has to be rewritten for the real world.
 */

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/envoyproxy/go-control-plane/pkg/util"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
)

const (
	serverCert = "cert/server.crt"
	serverKey  = "cert/server.key"
	localhost  = "127.0.0.1"
)

var (
	debug    bool
	adsPort  uint
	cfigPort uint
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Use debug logging")
	flag.UintVar(&adsPort, "ads", 18000, "ADS port")
	flag.UintVar(&cfigPort, "cfig", 9000, "Configurator port")
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

// Callbacks stuff
type callbacks struct {
	signal   chan struct{}
	fetches  int
	requests int
	mu       sync.Mutex
}

func (cb *callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	log.WithFields(log.Fields{"fetches": cb.fetches, "requests": cb.requests}).Info("server callbacks")
}
func (cb *callbacks) OnStreamOpen(id int64, typ string) {
	log.Debugf("stream %d open for %s", id, typ)
}
func (cb *callbacks) OnStreamClosed(id int64) {
	log.Debugf("stream %d closed", id)
}
func (cb *callbacks) OnStreamRequest(int64, *v2.DiscoveryRequest) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.requests++
	log.Debugf("stream request %d", cb.requests)
	if cb.signal != nil {
		log.Debugf("stream request %d signalling", cb.requests)
		close(cb.signal)
		cb.signal = nil
	}
}

func (cb *callbacks) OnStreamResponse(int64, *v2.DiscoveryRequest, *v2.DiscoveryResponse) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	log.Debugf("stream response %d", cb.requests)
}

func (cb *callbacks) OnFetchRequest(req *v2.DiscoveryRequest) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.fetches++
	log.Debugf("fetch request %d", cb.fetches)
	if cb.signal != nil {
		log.Debugf("fetch request %d signalling", cb.fetches)
		close(cb.signal)
		cb.signal = nil
	}
}
func (cb *callbacks) OnFetchResponse(*v2.DiscoveryRequest, *v2.DiscoveryResponse) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	log.Debugf("fetch response %d", cb.fetches)
}

// end callbacks stuff

// run stuff
// RunManagementServer starts an xDS server at the given port.
func runManagementServer(ctx context.Context, server server.Server, port uint) {
	// // Create the TLS credentials
	// creds, err := credentials.NewServerTLSFromFile(serverCert, serverKey)
	// if err != nil {
	// 	log.WithError(err).Fatal("could not load TLS keys")
	// }

	// grpcServer := grpc.NewServer(grpc.Creds(creds))
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

	log.WithFields(log.Fields{"port": port}).Info("management server listening")
	go func() {
		err := grpcServer.Serve(lis)

		log.WithFields(log.Fields{"error": err}).Error("management server exited?")

		// if err != nil {
		// 	log.Error(err)
		// }
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}

// This is our actual configuration engine. Such as it is.
type mappingInfo struct {
	prefix      string
	clusterName string
}

type configurator struct {
	// Current snapshot cache
	config cache.SnapshotCache
	// Configuration generation
	generation int
	// Most recent snapshot (is this even useful?)
	currentSnapshot *cache.Snapshot
	// Map of clusters, indexed by cluster name
	clusters map[string]*v2.Cluster
	// Map of endpoints, indexed by cluster name
	endpoints map[string]*v2.ClusterLoadAssignment
	// Map of mappings, indexed by mapping name
	mappings map[string]*mappingInfo
	// Map of listener ports, indexed by listener name
	listeners map[string]uint32
	// Mutex. Yay.
	mu sync.Mutex
}

func newConfigurator(config cache.SnapshotCache) configurator {
	return configurator{
		config:     config,
		clusters:   make(map[string]*v2.Cluster),
		endpoints:  make(map[string]*v2.ClusterLoadAssignment),
		mappings:   make(map[string]*mappingInfo),
		listeners:  make(map[string]uint32),
		generation: 0,
	}
}

func (cfig *configurator) handleListener(listenerName string, port uint32) {
	cfig.mu.Lock()
	defer cfig.mu.Unlock()

	cfig.listeners[listenerName] = port
}

func (cfig *configurator) handleListenerReq(rw http.ResponseWriter, request *http.Request) {
	args := request.URL.Query()
	log.Infof("handleListenerReq: %+v", args)

	listenerName := args["name"][0]
	port, err := strconv.Atoi(args["port"][0])

	if err != nil {
		fmt.Fprintf(rw, "port must be an integer, not '%s'\n", args["port"][0])
	} else {
		cfig.handleListener(listenerName, uint32(port))
		fmt.Fprintf(rw, "OK")
	}
}

func (cfig *configurator) handleMapping(mappingName, prefix, clusterName string) {
	cfig.mu.Lock()
	defer cfig.mu.Unlock()

	cfig.mappings[mappingName] = &mappingInfo{
		prefix:      prefix,
		clusterName: clusterName,
	}
}

func (cfig *configurator) handleMappingReq(rw http.ResponseWriter, request *http.Request) {
	args := request.URL.Query()
	log.Infof("handleMappingReq: %+v", args)

	cfig.handleMapping(args["name"][0], args["prefix"][0], args["cluster"][0])

	if _, present := args["update"]; present {
		cfig.update()
	}

	fmt.Fprintf(rw, "OK")
}

func (cfig *configurator) makeRouteConfiguration(routeName string) *v2.RouteConfiguration {
	routes := make([]route.Route, 0)

	for _, info := range cfig.mappings {
		routes = append(routes, route.Route{
			Match: route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: info.prefix,
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: info.clusterName,
					},
					PrefixRewrite: "/",
				},
			},
		})
	}

	return &v2.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []route.VirtualHost{{
			Name:    routeName,
			Domains: []string{"*"},
			Routes:  routes,
		}},
	}
}

func (cfig *configurator) makeListener(listenerName, routeConfigName string) *v2.Listener {
	// data source configuration
	rdsSource := core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_Ads{
			Ads: &core.AggregatedConfigSource{},
		},
	}

	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    rdsSource,
				RouteConfigName: routeConfigName,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: util.Router,
		}},
	}
	pbst, err := util.MessageToStruct(manager)
	if err != nil {
		panic(err)
	}

	return &v2.Listener{
		Name: listenerName,
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: cfig.listeners[listenerName],
					},
				},
			},
		},
		FilterChains: []listener.FilterChain{{
			Filters: []listener.Filter{{
				Name:   util.HTTPConnectionManager,
				Config: pbst,
			}},
		}},
	}
}

func (cfig *configurator) handleCluster(clusterName string, serviceHost string, servicePort uint32) {
	cluster := &v2.Cluster{
		Name:           clusterName,
		ConnectTimeout: 5 * time.Second,
		Type:           v2.Cluster_EDS,
		EdsClusterConfig: &v2.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
		},
	}

	endpoint := &v2.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []endpoint.LocalityLbEndpoints{{
			LbEndpoints: []endpoint.LbEndpoint{{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.TCP,
								Address:  serviceHost,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: servicePort,
								},
							},
						},
					},
				},
			}},
		}},
	}

	cfig.mu.Lock()
	defer cfig.mu.Unlock()

	cfig.clusters[clusterName] = cluster
	cfig.endpoints[clusterName] = endpoint
}

func (cfig *configurator) handleClusterReq(rw http.ResponseWriter, request *http.Request) {
	args := request.URL.Query()
	log.Infof("handleClusterReq: %+v", args)

	port, err := strconv.Atoi(args["port"][0])

	if err != nil {
		fmt.Fprintf(rw, "port must be an integer, not '%s'\n", args["port"][0])
	} else {
		cfig.handleCluster(args["name"][0], args["service"][0], uint32(port))
		fmt.Fprintf(rw, "OK")
	}

	fmt.Fprintf(rw, "OK")
}

func (cfig *configurator) update() {
	cfig.mu.Lock()
	defer cfig.mu.Unlock()

	clusters := []cache.Resource{}
	endpoints := []cache.Resource{}
	routes := []cache.Resource{}
	listeners := []cache.Resource{}

	for _, value := range cfig.clusters {
		clusters = append(clusters, value)
	}

	for _, value := range cfig.endpoints {
		endpoints = append(endpoints, value)
	}

	routes = append(routes, cfig.makeRouteConfiguration("test-routes"))

	for listenerName := range cfig.listeners {
		listeners = append(listeners, cfig.makeListener(listenerName, "test-routes"))
	}

	cfig.generation++
	version := fmt.Sprintf("v%d", cfig.generation)

	snapshot := cache.NewSnapshot(version, endpoints, clusters, routes, listeners)

	err := snapshot.Consistent()

	if err != nil {
		log.Errorf("snapshot inconsistency: %+v", snapshot)
	} else {
		err = cfig.config.SetSnapshot("test-id", snapshot)
	}

	if err != nil {
		log.Fatalf("snapshot error %q for %+v", err, snapshot)
	} else {
		log.Infof("current snapshot now %+v", snapshot)
		cfig.currentSnapshot = &snapshot
	}
}

func (cfig *configurator) handleUpdateReq(rw http.ResponseWriter, request *http.Request) {
	cfig.update()

	fmt.Fprintf(rw, "OK")
}

func (cfig *configurator) run(ctx context.Context, port uint) {
	reader := bufio.NewReader(os.Stdin)

	log.Info("configurator running")

	http.HandleFunc("/listener", func(rw http.ResponseWriter, request *http.Request) {
		cfig.handleListenerReq(rw, request)
	})
	http.HandleFunc("/mapping", func(rw http.ResponseWriter, request *http.Request) {
		cfig.handleMappingReq(rw, request)
	})
	http.HandleFunc("/cluster", func(rw http.ResponseWriter, request *http.Request) {
		cfig.handleClusterReq(rw, request)
	})
	http.HandleFunc("/update", func(rw http.ResponseWriter, request *http.Request) {
		cfig.handleUpdateReq(rw, request)
	})
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	log.WithFields(log.Fields{"port": port}).Info("update server listening")

	for {
		fmt.Print("-> ")
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)

		args := strings.Fields(text)
		cmd := args[0]
		args = args[1:]

		fmt.Printf("got cmd %s fields %v\n", cmd, args)

		if cmd == "quit" {
			os.Exit(0)
			return
		} else if (cmd == "c") || (cmd == "cluster") {
			if len(args) != 3 {
				fmt.Printf("usage: cluster name serviceHost servicePort\n")
			} else {
				clusterName := args[0]
				serviceHost := args[1]
				servicePort := args[2]

				srvPort, err := strconv.Atoi(servicePort)

				if err != nil {
					fmt.Printf("servicePort must be an integer, not %s\n", servicePort)
				} else {
					fmt.Printf("cluster %s => %s:%v\n", clusterName, serviceHost, srvPort)
					cfig.handleCluster(clusterName, serviceHost, uint32(srvPort))
				}
			}
		} else if (cmd == "u") || (cmd == "update") {
			fmt.Printf("update\n")
			cfig.update()
		} else if (cmd == "d") || (cmd == "dump") {
			fmt.Printf("listeners:\n")

			for key, value := range cfig.listeners {
				fmt.Printf("%s:\n", key)
				fmt.Printf("    %v\n", value)
			}

			fmt.Printf("mappings:\n")

			for key, value := range cfig.mappings {
				fmt.Printf("%s:\n", key)
				fmt.Printf("    %v\n", value)
			}

			fmt.Printf("clusters:\n")

			for key, value := range cfig.clusters {
				fmt.Printf("%s:\n", key)
				fmt.Printf("    %v\n", value)
				fmt.Printf("    %v\n", cfig.endpoints[key])
			}
		} else {
			fmt.Printf("command %s unknown\n", cmd)
		}
	}
}

func main() {
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	ctx := context.Background()

	signal := make(chan struct{})
	cb := &callbacks{signal: signal}
	config := cache.NewSnapshotCache(true, Hasher{}, logger{})
	srv := server.NewServer(config, cb)

	cfig := newConfigurator(config)

	cfig.handleCluster("qotm-1", "192.168.65.2", 5000)
	cfig.handleCluster("qotm-2", "192.168.65.2", 6000)
	cfig.handleMapping("qotm-1-mapping", "/qotm-1/", "qotm-1")
	cfig.handleMapping("qotm-2-mapping", "/qotm-2/", "qotm-2")
	cfig.handleMapping("qotm-mapping", "/qotm/", "qotm-1")
	cfig.handleListener("test-listener", 8000)
	cfig.update()

	go runManagementServer(ctx, srv, adsPort)
	go cfig.run(ctx, cfigPort)

	log.Infof("waiting for the first request...")
	<-signal

	// Load up some stuff.

	select {
	// case <-time.After(10 * time.Second):
	case <-ctx.Done():
		log.Info("done")
	}

	log.Infof("worked! debug %s", debug)
}
