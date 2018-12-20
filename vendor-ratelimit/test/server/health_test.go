package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/signal"
	"syscall"
	"testing"

	"github.com/lyft/ratelimit/src/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestHealthCheck(t *testing.T) {
	defer signal.Reset(syscall.SIGTERM)

	recorder := httptest.NewRecorder()

	hc := server.NewHealthChecker(health.NewServer())

	r, _ := http.NewRequest("GET", "http://1.2.3.4/healthcheck", nil)
	hc.ServeHTTP(recorder, r)

	if 200 != recorder.Code {
		t.Errorf("expected code 200 actual %d", recorder.Code)
	}

	if "OK" != recorder.Body.String() {
		t.Errorf("expected body 'OK', got '%s'", recorder.Body.String())
	}

	hc.Fail()

	recorder = httptest.NewRecorder()

	r, _ = http.NewRequest("GET", "http://1.2.3.4/healthcheck", nil)
	hc.ServeHTTP(recorder, r)

	if 500 != recorder.Code {
		t.Errorf("expected code 500 actual %d", recorder.Code)
	}

}

func TestGrpcHealthCheck(t *testing.T) {
	defer signal.Reset(syscall.SIGTERM)

	grpcHealthServer := health.NewServer()
	hc := server.NewHealthChecker(grpcHealthServer)
	healthpb.RegisterHealthServer(grpc.NewServer(), grpcHealthServer)

	req := &healthpb.HealthCheckRequest{
		Service: "ratelimit",
	}

	res, _ := grpcHealthServer.Check(context.Background(), req)
	if healthpb.HealthCheckResponse_SERVING != res.Status {
		t.Errorf("expected status SERVING actual %v", res.Status)
	}

	hc.Fail()

	res, _ = grpcHealthServer.Check(context.Background(), req)
	if healthpb.HealthCheckResponse_NOT_SERVING != res.Status {
		t.Errorf("expected status NOT_SERVING actual %v", res.Status)
	}
}
