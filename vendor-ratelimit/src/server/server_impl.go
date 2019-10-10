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

	reuseport "github.com/kavu/go_reuseport"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"

	"github.com/lyft/ratelimit/src/settings"
)

func Run(s settings.Settings, grpcServer *grpc.Server, debugHTTPHandler http.Handler, healthHTTPHandler http.Handler, healthGRPCHandler *health.Server) {
	var debugListener net.Listener
	go func() {
		addr := fmt.Sprintf(":%d", s.DebugPort)
		logger.Warnf("Listening for debug on '%s'", addr)
		var err error
		debugListener, err = reuseport.Listen("tcp", addr)

		if err != nil {
			logger.Errorf("Failed to open debug HTTP listener: '%+v'", err)
			return
		}
		err = http.Serve(debugListener, debugHTTPHandler)
		logger.Infof("Failed to start debug server '%+v'", err)
	}()

	go func() {
		addr := fmt.Sprintf(":%d", s.GrpcPort)
		logger.Warnf("Listening for gRPC on '%s'", addr)
		lis, err := reuseport.Listen("tcp", addr)
		if err != nil {
			logger.Fatalf("Failed to listen for gRPC: %v", err)
		}
		grpcServer.Serve(lis)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-sigs

		logger.Infof("Ratelimit server received %v, shutting down gracefully", sig)
		healthGRPCHandler.Shutdown()
		grpcServer.GracefulStop()
		if debugListener != nil {
			debugListener.Close()
		}
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%d", s.Port)
	logger.Warnf("Listening for HTTP on '%s'", addr)
	list, err := reuseport.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to open HTTP listener: '%+v'", err)
	}
	logger.Fatal(http.Serve(list, healthHTTPHandler))
}

type serverDebugHandler struct {
	endpoints map[string]string
	debugMux  *http.ServeMux
}

func NewDebugHTTPHandler() DebugHTTPHandler {
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
			if request.URL.Path != "/" {
				http.NotFound(writer, request)
				return
			}
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
