// Package test contains test utilities
package test

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	serverv2 "github.com/datawire/ambassador/pkg/envoy-control-plane/server/v2"
	serverv3 "github.com/datawire/ambassador/pkg/envoy-control-plane/server/v3"
	testv2 "github.com/datawire/ambassador/pkg/envoy-control-plane/test/v2"
	testv3 "github.com/datawire/ambassador/pkg/envoy-control-plane/test/v3"
	"google.golang.org/grpc"

	gcplogger "github.com/datawire/ambassador/pkg/envoy-control-plane/log"
)

const (
	grpcMaxConcurrentStreams = 1000000
)

// HTTPGateway is a custom implementation of [gRPC gateway](https://github.com/grpc-ecosystem/grpc-gateway)
// specialized to Envoy xDS API.
type HTTPGateway struct {
	// Log is an optional log for errors in response write
	Log gcplogger.Logger

	GatewayV2 serverv2.HTTPGateway

	GatewayV3 serverv3.HTTPGateway
}

// RunAccessLogServer starts an accesslog server.
func RunAccessLogServer(ctx context.Context, alsv2 *testv2.AccessLogService, alsv3 *testv3.AccessLogService, alsPort uint) {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", alsPort))
	if err != nil {
		log.Fatal(err)
	}

	testv2.RegisterAccessLogServer(grpcServer, alsv2)
	testv3.RegisterAccessLogServer(grpcServer, alsv3)
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
func RunManagementServer(ctx context.Context, srv2 serverv2.Server, srv3 serverv3.Server, port uint) {
	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems.
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	testv2.RegisterServer(grpcServer, srv2)
	testv3.RegisterServer(grpcServer, srv3)

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
func RunManagementGateway(ctx context.Context, srv2 serverv2.Server, srv3 serverv3.Server, port uint, lg gcplogger.Logger) {
	log.Printf("gateway listening HTTP/1.1 on %d\n", port)
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: &HTTPGateway{
			GatewayV2: serverv2.HTTPGateway{Log: lg, Server: srv2},
			GatewayV3: serverv3.HTTPGateway{Log: lg, Server: srv3},
			Log:       lg,
		},
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	<-ctx.Done()

	// Cleanup our gateway if we receive a shutdown
	server.Shutdown(ctx)
}

func (h *HTTPGateway) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	bytes, code, err := h.GatewayV2.ServeHTTP(req)
	if code == http.StatusNotFound {
		bytes, code, err = h.GatewayV3.ServeHTTP(req)
	}

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
