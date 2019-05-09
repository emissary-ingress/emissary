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

type serverDebugHandler struct {
	endpoints map[string]string
	debugMux  *http.ServeMux
}

type server struct {
	port          int
	grpcPort      int
	debugPort     int
	router        *mux.Router
	grpcServer    *grpc.Server
	store         stats.Store
	scope         stats.Scope
	runtime       loader.IFace
	debugHandler  *serverDebugHandler
	debugListener net.Listener
	health        *healthChecker
}

func (debugHandler *serverDebugHandler) AddEndpoint(path string, help string, handler http.HandlerFunc) {
	debugHandler.debugMux.HandleFunc(path, handler)
	debugHandler.endpoints[path] = help
}

func (debugHandler *serverDebugHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	debugHandler.debugMux.ServeHTTP(w, r)
}

func (server *server) DebugHTTPHandler() DebugHTTPHandler {
	return server.debugHandler
}

func (server *server) GrpcServer() *grpc.Server {
	return server.grpcServer
}

func (server *server) Start() {
	go func() {
		addr := fmt.Sprintf(":%d", server.debugPort)
		logger.Warnf("Listening for debug on '%s'", addr)
		var err error
		server.debugListener, err = reuseport.Listen("tcp", addr)

		if err != nil {
			logger.Errorf("Failed to open debug HTTP listener: '%+v'", err)
			return
		}
		err = http.Serve(server.debugListener, server.debugHandler)
		logger.Infof("Failed to start debug server '%+v'", err)
	}()

	go server.startGrpc()

	server.handleGracefulShutdown()

	addr := fmt.Sprintf(":%d", server.port)
	logger.Warnf("Listening for HTTP on '%s'", addr)
	list, err := reuseport.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to open HTTP listener: '%+v'", err)
	}
	logger.Fatal(http.Serve(list, server.router))
}

func (server *server) startGrpc() {
	addr := fmt.Sprintf(":%d", server.grpcPort)
	logger.Warnf("Listening for gRPC on '%s'", addr)
	lis, err := reuseport.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to listen for gRPC: %v", err)
	}
	server.grpcServer.Serve(lis)
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
	ret.health = NewHealthChecker(health.NewServer())
	ret.router.Path("/healthcheck").Handler(ret.health)
	healthpb.RegisterHealthServer(ret.grpcServer, ret.health.grpc)

	// setup default debug listener
	ret.debugHandler = newDebugHTTPHandler()

	return ret
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

func (server *server) handleGracefulShutdown() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		sig := <-sigs

		logger.Infof("Ratelimit server received %v, shutting down gracefully", sig)
		server.grpcServer.GracefulStop()
		if server.debugListener != nil {
			server.debugListener.Close()
		}
		os.Exit(0)
	}()
}
