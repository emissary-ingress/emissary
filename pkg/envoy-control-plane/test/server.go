// Package test contains test utilities
package test

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	server "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/test/v3"

	gcplogger "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/log"
)

const (
	grpcKeepaliveTime        = 30 * time.Second
	grpcKeepaliveTimeout     = 5 * time.Second
	grpcKeepaliveMinTime     = 30 * time.Second
	grpcMaxConcurrentStreams = 1000000
)

// HTTPGateway is a custom implementation of [gRPC gateway](https://github.com/grpc-ecosystem/grpc-gateway)
// specialized to Envoy xDS API.
type HTTPGateway struct {
	// Log is an optional log for errors in response write
	Log gcplogger.Logger

	Gateway server.HTTPGateway
}

// RunAccessLogServer starts an accesslog server.
func RunAccessLogServer(ctx context.Context, als *test.AccessLogService, alsPort uint) {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", alsPort))
	if err != nil {
		log.Fatal(err)
	}

	test.RegisterAccessLogServer(grpcServer, als)
	log.Printf("access log server listening on %d\n", alsPort)

	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Println(err)
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}

// RunManagementServer starts an xDS server at the given port.
func RunManagementServer(ctx context.Context, srv server.Server, port uint) {
	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems. Keepalive timeouts based on connection_keepalive parameter https://www.envoyproxy.io/docs/envoy/latest/configuration/overview/examples#dynamic
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions,
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    grpcKeepaliveTime,
			Timeout: grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	)
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	test.RegisterServer(grpcServer, srv)

	log.Printf("management server listening on %d\n", port)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Println(err)
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}

// RunManagementGateway starts an HTTP gateway to an xDS server.
func RunManagementGateway(ctx context.Context, srv server.Server, port uint) {
	log.Printf("gateway listening HTTP/1.1 on %d\n", port)
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: &HTTPGateway{
			Gateway: server.HTTPGateway{Server: srv},
		},
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("failed to start listening: %s", err)
		}
	}()
	<-ctx.Done()

	// Cleanup our gateway if we receive a shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("failed to shut down: %s", err)
	}
}

func (h *HTTPGateway) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	bytes, code, err := h.Gateway.ServeHTTP(req)

	if err != nil {
		http.Error(resp, err.Error(), code)
		return
	}

	if bytes == nil {
		resp.WriteHeader(http.StatusNotModified)
		return
	}

	if _, err = resp.Write(bytes); err != nil && h.Log != nil {
		h.Log.Errorf("gateway error: %v", err)
	}
}
