package runner

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	// 3rd-party libraries
	"github.com/lyft/goruntime/loader"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	grpchealth "google.golang.org/grpc/health"

	// first-party libraries
	"github.com/datawire/ambassador/pkg/k8s"
	stats "github.com/lyft/gostats"

	// internal libraries: github.com/datawire/apro
	"github.com/datawire/apro/cmd/amb-sidecar/acmeclient"
	"github.com/datawire/apro/cmd/amb-sidecar/banner"
	devportalcontent "github.com/datawire/apro/cmd/amb-sidecar/devportal/content"
	devportalserver "github.com/datawire/apro/cmd/amb-sidecar/devportal/server"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	filterhandler "github.com/datawire/apro/cmd/amb-sidecar/filters/handler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/health"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/firstboot"
	rlscontroller "github.com/datawire/apro/cmd/amb-sidecar/ratelimits"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/datawire/apro/lib/util"

	// internal libraries: github.com/lyft/ratelimit
	lyftconfig "github.com/lyft/ratelimit/src/config"
	lyftredis "github.com/lyft/ratelimit/src/redis"
	lyftserver "github.com/lyft/ratelimit/src/server"
	lyftservice "github.com/lyft/ratelimit/src/service"

	// k8s clients
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// gRPC service APIs
	rlsV1api "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v1"
	rlsV2api "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2"
	healthapi "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/datawire/apro/lib/filterapi"
)

var licenseClaims *licensekeys.LicenseClaimsLatest
var logrusLogger *logrus.Logger

func Main(version string) {
	argparser := &cobra.Command{
		Use:           os.Args[0],
		Version:       version,
		RunE:          runE,
		SilenceErrors: true, // we'll handle it after .Execute() returns
		SilenceUsage:  true, // our FlagErrorFunc wil handle it
	}

	keycheck := licensekeys.InitializeCommandFlags(argparser.PersistentFlags(), "ambassador-sidecar", version)

	argparser.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err == nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "%s\nSee '%s --help'.\n", err, cmd.CommandPath())
		os.Exit(2)
		return nil
	})

	// Initialize the root logger.  We'll use this for top-level
	// things that don't involve any specific worker process.
	logrusLogger = logrus.New()
	logrusFormatter := &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	}
	logrusLogger.SetFormatter(logrusFormatter)
	logrus.SetFormatter(logrusFormatter) // FIXME(lukeshu): Some Lyft code still uses the global logger

	argparser.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// License key validation
		var err error
		licenseClaims, err = keycheck(cmd.PersistentFlags())
		return err
	}

	err := argparser.Execute()
	if err != nil {
		logrusLogger.Errorln(err)
		os.Exit(1)
	}
}

func runE(cmd *cobra.Command, args []string) error {
	// Load the configuration
	cfg, warn, fatal := types.ConfigFromEnv()
	for _, err := range warn {
		logrusLogger.Warnln("config error:", err)
	}
	for _, err := range fatal {
		logrusLogger.Errorln("config error:", err)
	}
	if len(fatal) > 0 {
		return fatal[len(fatal)-1]
	}
	logrusLogger.Info("Ambassador Pro configuation loaded")

	if err := os.MkdirAll(filepath.Dir(cfg.RLSRuntimeDir), 0777); err != nil {
		return err
	}

	// cfg.LogLevel has already been validated in
	// ConfigFromEnv(), no need to error-check.
	level, _ := logrus.ParseLevel(cfg.LogLevel)
	logrusLogger.SetLevel(level)
	logrus.SetLevel(level) // FIXME(lukeshu): Some Lyft code still uses the global logger

	kubeinfo := k8s.NewKubeInfo("", "", "") // Empty file/ctx/ns for defaults

	// Initialize the errgroup we'll use to orchestrate the goroutines.
	group := NewGroup(context.Background(), cfg, func(name string) types.Logger {
		return types.WrapLogrus(logrusLogger).WithField("MAIN", name)
	})

	// Launch all of the worker goroutines...

	// RateLimit controller
	if licenseClaims.RequireFeature(licensekeys.FeatureRateLimit) == nil {
		group.Go("ratelimit_controller", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
			return rlscontroller.DoWatch(softCtx, cfg, l)
		})
	}

	// Filter+FilterPolicy controller
	ct := &controller.Controller{}
	if licenseClaims.RequireFeature(licensekeys.FeatureFilter) == nil || licenseClaims.RequireFeature(licensekeys.FeatureDevPortal) == nil {
		group.Go("auth_controller", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
			ct.Config = cfg
			ct.Logger = l
			return ct.Watch(softCtx, kubeinfo, licenseClaims)
		})
	}

	// DevPortal
	var devPortalServer *devportalserver.Server
	if licenseClaims.RequireFeature(licensekeys.FeatureDevPortal) == nil {
		content, err := devportalcontent.NewContent(cfg.DevPortalContentURL)
		if err != nil {
			return err
		}
		devPortalServer = devportalserver.NewServer("/docs", content,
			licenseClaims.GetLimitValue(licensekeys.LimitDevPortalServices))
		group.Go("devportal_fetcher", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
			fetcher := devportalserver.NewFetcher(devPortalServer, devportalserver.HTTPGet, devPortalServer.KnownServices(), cfg)
			fetcher.Run(softCtx)
			return nil
		})
	}

	// ACME client
	group.Go("acme_client", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
		return acmeclient.EnsureFallback(cfg, kubeinfo)
	})

	// HTTP server
	group.Go("http", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
		// A good chunk of this code mimics github.com/lyft/ratelimit/src/service_cmd/runner.Run()

		statsStore := stats.NewDefaultStore()
		statsStore.AddStatGenerator(stats.NewRuntimeStats(statsStore.Scope("go")))

		redisPool, err := pool.New(cfg.RedisSocketType, cfg.RedisURL, cfg.RedisPoolSize)
		if err != nil {
			return errors.Wrap(err, "redis pool")
		}

		var redisPerSecondPool *pool.Pool
		if cfg.RedisPerSecond {
			redisPerSecondPool, err = pool.New(cfg.RedisPerSecondSocketType, cfg.RedisPerSecondURL, cfg.RedisPerSecondPoolSize)
			if err != nil {
				return errors.Wrap(err, "redis per-second pool")
			}
		}

		// Now attach services to these 2 handlers
		grpcHandler := grpc.NewServer(grpc.UnaryInterceptor(nil))
		httpHandler := lyftserver.NewDebugHTTPHandler()

		// Liveness and Readiness probes
		healthprobe := health.MultiProbe{
			Logger: l,
		}
		// This is a probe that always returns true... it is
		// admittedly "dumb", but if the HTTP server stops
		// serving this will fail and it forms the basis of
		// the Probe API which we can use for subsequent more
		// involved probes.
		healthprobe.RegisterProbe("basic", &health.StaticProbe{Value: true})
		healthprobeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			healthy := healthprobe.Check()
			if healthy {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		httpHandler.AddEndpoint("/_/sys/readyz", "readiness probe endpoint", healthprobeHandler)
		httpHandler.AddEndpoint("/_/sys/healthz", "liveness probe endpoint", healthprobeHandler)

		// HealthService
		healthService := grpchealth.NewServer()
		healthapi.RegisterHealthServer(grpcHandler, healthService)
		go func() {
			<-softCtx.Done()
			healthService.Shutdown()
		}()
		httpHandler.AddEndpoint(
			"/healthcheck",
			"check the health of Ambassador Pro",
			lyftserver.NewHealthChecker(healthService).ServeHTTP)

		// AuthService
		if licenseClaims.RequireFeature(licensekeys.FeatureFilter) == nil || licenseClaims.RequireFeature(licensekeys.FeatureDevPortal) == nil {
			restconfig, err := kubeinfo.GetRestConfig()
			if err != nil {
				return err
			}
			coreClient, err := k8sClientCoreV1.NewForConfig(restconfig)
			if err != nil {
				return err
			}
			authService, err := filterhandler.NewFilterMux(cfg, l.WithField("SUB", "http-handler"), ct, coreClient, redisPool)
			if err != nil {
				return err
			}
			filterapi.RegisterFilterService(grpcHandler, authService)
			httpHandler.AddEndpoint("/.ambassador/", "OAuth2 Filter", authService.ServeHTTP)
		}

		// RateLimitService
		if licenseClaims.RequireFeature(licensekeys.FeatureRateLimit) == nil {
			rateLimitScope := statsStore.Scope("ratelimit")
			rateLimitService := lyftservice.NewService(
				loader.New(
					cfg.RLSRuntimeDir,               // runtime path
					cfg.RLSRuntimeSubdir,            // runtime subdirectory
					rateLimitScope.Scope("runtime"), // stats scope
					&loader.SymlinkRefresher{RuntimePath: cfg.RLSRuntimeDir}, // refresher
				),
				lyftredis.NewRateLimitCacheImpl(
					lyftredis.NewPool(rateLimitScope.Scope("redis_pool"), redisPool),
					lyftredis.NewPool(rateLimitScope.Scope("redis_per_second_pool"), redisPerSecondPool),
					lyftredis.NewTimeSourceImpl(),
					rand.New(lyftredis.NewLockedSource(time.Now().Unix())),
					cfg.ExpirationJitterMaxSeconds),
				lyftconfig.NewRateLimitConfigLoaderImpl(),
				rateLimitScope.Scope("service"))
			rlsV1api.RegisterRateLimitServiceServer(grpcHandler, rateLimitService.GetLegacyService())
			rlsV2api.RegisterRateLimitServiceServer(grpcHandler, rateLimitService)
			httpHandler.AddEndpoint(
				"/rlconfig",
				"print out the currently loaded configuration for debugging",
				func(writer http.ResponseWriter, request *http.Request) {
					io.WriteString(writer, rateLimitService.GetCurrentConfig().Dump())
				})
		}

		// DevPortal
		if licenseClaims.RequireFeature(licensekeys.FeatureDevPortal) == nil {
			httpHandler.AddEndpoint("/docs/", "Documentation portal", devPortalServer.Router().ServeHTTP)
			httpHandler.AddEndpoint("/openapi/", "Documentation portal API", devPortalServer.Router().ServeHTTP)
		}

		httpHandler.AddEndpoint("/firstboot/", "First boot wizard", http.StripPrefix("/firstboot", firstboot.NewFirstBootWizard()).ServeHTTP)
		httpHandler.AddEndpoint("/banner/", "Diag UI banner", http.StripPrefix("/banner", banner.NewBanner()).ServeHTTP)

		// Launch the server
		server := &http.Server{
			Addr:     ":" + cfg.HTTPPort,
			ErrorLog: l.WithField("SUB", "http-server").StdLogger(types.LogLevelError),
			// The net/http.Server doesn't support h2c (unencrypted
			// HTTP/2) built-in.  Since we want to have gRPC and plain
			// HTTP/1 on the same unencrypted port, we need h2c.
			// Fortunately, x/net has an h2c implementation we can use.
			Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				ctx = middleware.WithLogger(ctx, l.WithField("SUB", "http-server/handler"))
				ctx = middleware.WithRequestID(ctx, "unknown")
				r = r.WithContext(ctx)

				if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
					grpcHandler.ServeHTTP(w, r)
				} else {
					httpHandler.ServeHTTP(w, r)
				}
			}), &http2.Server{}),
		}
		return util.ListenAndServeHTTPWithContext(hardCtx, softCtx, server)
	})

	// And now we wait.
	return group.Wait()
}
