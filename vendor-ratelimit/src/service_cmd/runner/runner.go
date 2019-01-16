package runner

import (
	"io"
	"math/rand"
	"net/http"
	"time"

	pb "github.com/lyft/ratelimit/proto/envoy/service/ratelimit/v2"
	pb_legacy "github.com/lyft/ratelimit/proto/ratelimit"

	"github.com/lyft/ratelimit/src/config"
	"github.com/lyft/ratelimit/src/redis"
	"github.com/lyft/ratelimit/src/server"
	"github.com/lyft/ratelimit/src/service"
	"github.com/lyft/ratelimit/src/settings"
	logger "github.com/sirupsen/logrus"
)

func Run() {
	s := settings.NewSettings()

	logLevel, err := logger.ParseLevel(s.LogLevel)
	if err != nil {
		logger.Fatalf("Could not parse log level. %v\n", err)
	} else {
		logger.SetLevel(logLevel)
	}

	srv := server.NewServer("ratelimit", settings.GrpcUnaryInterceptor(nil))

	var perSecondPool redis.Pool
	if s.RedisPerSecond {
		perSecondPool = redis.NewPoolImpl(srv.Scope().Scope("redis_per_second_pool"), s.RedisPerSecondSocketType, s.RedisPerSecondUrl, s.RedisPerSecondPoolSize)

	}

	service := ratelimit.NewService(
		srv.Runtime(),
		redis.NewRateLimitCacheImpl(
			redis.NewPoolImpl(srv.Scope().Scope("redis_pool"), s.RedisSocketType, s.RedisUrl, s.RedisPoolSize),
			perSecondPool,
			redis.NewTimeSourceImpl(),
			rand.New(redis.NewLockedSource(time.Now().Unix())),
			s.ExpirationJitterMaxSeconds),
		config.NewRateLimitConfigLoaderImpl(),
		srv.Scope().Scope("service"))

	srv.AddDebugHttpEndpoint(
		"/rlconfig",
		"print out the currently loaded configuration for debugging",
		func(writer http.ResponseWriter, request *http.Request) {
			io.WriteString(writer, service.GetCurrentConfig().Dump())
		})

	// Ratelimit is compatible with two proto definitions
	// 1. data-plane-api rls.proto: https://github.com/envoyproxy/data-plane-api/blob/master/envoy/service/ratelimit/v2/rls.proto
	pb.RegisterRateLimitServiceServer(srv.GrpcServer(), service)
	// 2. ratelimit.proto defined in this repository: https://github.com/lyft/ratelimit/blob/0ded92a2af8261d43096eba4132e45b99a3b8b14/proto/ratelimit/ratelimit.proto
	pb_legacy.RegisterRateLimitServiceServer(srv.GrpcServer(), service.GetLegacyService())
	// (1) is the current definition, and (2) is the legacy definition.

	srv.Start()
}
