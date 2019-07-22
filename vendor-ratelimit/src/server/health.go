package server

import (
	"net/http"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type HealthChecker interface {
	http.Handler

	// Fail is only used for testing.
	Fail()
}

type healthChecker struct {
	grpc *health.Server
}

func NewHealthChecker(grpcHealthServer *health.Server) HealthChecker {
	ret := &healthChecker{}

	ret.grpc = grpcHealthServer
	ret.grpc.SetServingStatus("ratelimit", healthpb.HealthCheckResponse_SERVING)

	return ret
}

func (hc *healthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response, err := hc.grpc.Check(r.Context(), &healthpb.HealthCheckRequest{Service: "ratelimit"})
	if err == nil && response != nil && response.Status == healthpb.HealthCheckResponse_SERVING {
		w.Write([]byte("OK"))
	} else {
		w.WriteHeader(500)
	}
}

func (hc *healthChecker) Fail() {
	hc.grpc.SetServingStatus("ratelimit", healthpb.HealthCheckResponse_NOT_SERVING)
}
