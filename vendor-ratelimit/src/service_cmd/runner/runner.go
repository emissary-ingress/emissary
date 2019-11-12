package runner

import (
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lyft/goruntime/loader"
	stats "github.com/lyft/gostats"
	logger "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	pb_legacy "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v1"
	pb "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2"

	"github.com/lyft/ratelimit/src/config"
	"github.com/lyft/ratelimit/src/redis"
	"github.com/lyft/ratelimit/src/server"
	ratelimit "github.com/lyft/ratelimit/src/service"
	"github.com/lyft/ratelimit/src/settings"

	mock_limiter "github.com/datawire/apro/cmd/amb-sidecar/limiter/mocks"
)

func Run() {
	// Parse settings
	s := settings.NewSettings()

	logLevel, err := logger.ParseLevel(s.LogLevel)
	if err != nil {
		logger.Fatalf("Could not parse log level. %v\n", err)
	} else {
		logger.SetLevel(logLevel)
	}

	opts := []settings.Option{
		settings.GrpcUnaryInterceptor(nil),
	}
	for _, opt := range opts {
		opt(&s)
	}

	// Initialize stats store
	statsStore := stats.NewDefaultStore()
	statsScopeRatelimit := statsStore.Scope("ratelimit")
	statsStore.AddStatGenerator(stats.NewRuntimeStats(statsScopeRatelimit.Scope("go")))

	// Create the top-level things for the 3 ports we listen on
	grpcServer := grpc.NewServer(s.GrpcUnaryInterceptor)
	debugHTTPHandler := server.NewDebugHTTPHandler()
	healthHTTPHandler := mux.NewRouter()

	// Health Service
	healthGRPCHandler := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthGRPCHandler)
	healthHTTPHandler.Path("/healthcheck").Handler(server.NewHealthChecker(healthGRPCHandler))

	// Rate Limit Service
	var perSecondPool redis.Pool
	if s.RedisPerSecond {
		perSecondPool = redis.NewPoolImpl(statsScopeRatelimit.Scope("redis_per_second_pool"), s.RedisPerSecondSocketType, s.RedisPerSecondUrl, s.RedisPerSecondPoolSize)
	}
	service := ratelimit.NewService(
		loader.New(
			s.RuntimePath,               // runtime path
			s.RuntimeSubdirectory,       // runtime subdirectory
			statsStore.Scope("runtime"), // stats scope
			&loader.SymlinkRefresher{RuntimePath: s.RuntimePath}, // refresher
		),
		redis.NewRateLimitCacheImpl(
			redis.NewPoolImpl(statsScopeRatelimit.Scope("redis_pool"), s.RedisSocketType, s.RedisUrl, s.RedisPoolSize),
			perSecondPool,
			redis.NewTimeSourceImpl(),
			rand.New(redis.NewLockedSource(time.Now().Unix())),
			s.ExpirationJitterMaxSeconds),
		config.NewRateLimitConfigLoaderImpl(),
		statsScopeRatelimit.Scope("service"),
		mock_limiter.NewMockLimiter(),
	)
	debugHTTPHandler.AddEndpoint(
		"/rlconfig",
		"print out the currently loaded configuration for debugging",
		func(writer http.ResponseWriter, request *http.Request) {
			io.WriteString(writer, service.GetCurrentConfig().Dump())
		})
	// Ratelimit is compatible with two proto definitions
	// 1. data-plane-api rls.proto: https://github.com/envoyproxy/data-plane-api/blob/master/envoy/service/ratelimit/v2/rls.proto
	pb.RegisterRateLimitServiceServer(grpcServer, service)
	// 2. ratelimit.proto defined in this repository: https://github.com/lyft/ratelimit/blob/0ded92a2af8261d43096eba4132e45b99a3b8b14/proto/ratelimit/ratelimit.proto
	pb_legacy.RegisterRateLimitServiceServer(grpcServer, service.GetLegacyService())
	// (1) is the current definition, and (2) is the legacy definition.

	// Now Run everything
	server.Run(s, grpcServer, debugHTTPHandler, healthHTTPHandler, healthGRPCHandler)
}
