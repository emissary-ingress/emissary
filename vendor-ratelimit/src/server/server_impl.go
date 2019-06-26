package server

import (
	"expvar"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/gorilla/mux"
	reuseport "github.com/kavu/go_reuseport"
	"github.com/lyft/goruntime/loader"
	stats "github.com/lyft/gostats"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/lyft/ratelimit/src/settings"
)

type server struct {
	grpcServer *grpc.Server
	// ports
	port      int
	grpcPort  int
	debugPort int
	// stats
	store stats.Store
	scope stats.Scope
	// runtime
	runtime loader.IFace
	// heathcheck
	healthGRPC *health.Server
	healthHTTP HealthChecker
	router     *mux.Router
	// debug
	debugHandler *serverDebugHandler
}

func (server *server) DebugHTTPHandler() DebugHTTPHandler {
	return server.debugHandler
}

func (server *server) GrpcServer() *grpc.Server {
	return server.grpcServer
}

// - http.Serve(sock, server.debugHandler)
// - server.grpcServer.Serve(sock)
// - http.Serve(sock, server.router) // healthcheck
func (server *server) Start() {
	var debugListener net.Listener
	go func() {
		addr := fmt.Sprintf(":%d", server.debugPort)
		logger.Warnf("Listening for debug on '%s'", addr)
		var err error
		debugListener, err = reuseport.Listen("tcp", addr)

		if err != nil {
			logger.Errorf("Failed to open debug HTTP listener: '%+v'", err)
			return
		}
		err = http.Serve(debugListener, server.debugHandler)
		logger.Infof("Failed to start debug server '%+v'", err)
	}()

	go func() {
		addr := fmt.Sprintf(":%d", server.grpcPort)
		logger.Warnf("Listening for gRPC on '%s'", addr)
		lis, err := reuseport.Listen("tcp", addr)
		if err != nil {
			logger.Fatalf("Failed to listen for gRPC: %v", err)
		}
		server.grpcServer.Serve(lis)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-sigs

		logger.Infof("Ratelimit server received %v, shutting down gracefully", sig)
		server.healthGRPC.Shutdown()
		server.grpcServer.GracefulStop()
		if debugListener != nil {
			debugListener.Close()
		}
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%d", server.port)
	logger.Warnf("Listening for HTTP on '%s'", addr)
	list, err := reuseport.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to open HTTP listener: '%+v'", err)
	}
	logger.Fatal(http.Serve(list, server.router))
}

func (server *server) Scope() stats.Scope {
	return server.scope
}

func (server *server) Runtime() loader.IFace {
	return server.runtime
}

func NewServer(name string, opts ...settings.Option) Server {
	return newServer(name, opts...)
}

func newServer(name string, opts ...settings.Option) *server {
	s := settings.NewSettings()

	for _, opt := range opts {
		opt(&s)
	}

	ret := new(server)
	ret.grpcServer = grpc.NewServer(s.GrpcUnaryInterceptor)

	// setup ports
	ret.port = s.Port
	ret.grpcPort = s.GrpcPort
	ret.debugPort = s.DebugPort

	// setup stats
	ret.store = stats.NewDefaultStore()
	ret.scope = ret.store.Scope(name)
	ret.store.AddStatGenerator(stats.NewRuntimeStats(ret.scope.Scope("go")))

	// setup runtime
	ret.runtime = loader.New(
		s.RuntimePath,              // runtime path
		s.RuntimeSubdirectory,      // runtime subdirectory
		ret.store.Scope("runtime"), // stats scope
		&loader.SymlinkRefresher{RuntimePath: s.RuntimePath}, // refresher
	)

	// setup http router
	ret.router = mux.NewRouter()

	// setup healthcheck path
	ret.healthGRPC = health.NewServer()
	healthpb.RegisterHealthServer(ret.grpcServer, ret.healthGRPC)

	ret.healthHTTP = NewHealthChecker(ret.healthGRPC)
	ret.router.Path("/healthcheck").Handler(ret.healthHTTP)

	// setup default debug listener
	ret.debugHandler = newDebugHTTPHandler()

	return ret
}

type serverDebugHandler struct {
	endpoints map[string]string
	debugMux  *http.ServeMux
}

func newDebugHTTPHandler() *serverDebugHandler {
	ret := &serverDebugHandler{}

	ret.debugMux = http.NewServeMux()
	ret.endpoints = map[string]string{}
	ret.AddEndpoint(
		"/debug/pprof/",
		"root of various pprof endpoints. hit for help.",
		func(writer http.ResponseWriter, request *http.Request) {
			pprof.Index(writer, request)
		})

	// setup stats endpoint
	ret.AddEndpoint(
		"/stats",
		"print out stats",
		func(writer http.ResponseWriter, request *http.Request) {
			expvar.Do(func(kv expvar.KeyValue) {
				io.WriteString(writer, fmt.Sprintf("%s: %s\n", kv.Key, kv.Value))
			})
		})

	// setup debug root
	ret.debugMux.HandleFunc(
		"/",
		func(writer http.ResponseWriter, request *http.Request) {
			sortedKeys := []string{}
			for key := range ret.endpoints {
				sortedKeys = append(sortedKeys, key)
			}

			sort.Strings(sortedKeys)
			for _, key := range sortedKeys {
				io.WriteString(
					writer, fmt.Sprintf("%s: %s\n", key, ret.endpoints[key]))
			}
		})

	return ret
}

func (debugHandler *serverDebugHandler) AddEndpoint(path string, help string, handler http.HandlerFunc) {
	debugHandler.debugMux.HandleFunc(path, handler)
	debugHandler.endpoints[path] = help
}

func (debugHandler *serverDebugHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	debugHandler.debugMux.ServeHTTP(w, r)
}
