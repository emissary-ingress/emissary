package runner

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// 3rd-party libraries
	"github.com/fsnotify/fsnotify"
	"github.com/lyft/goruntime/loader"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"k8s.io/klog"

	// first-party libraries
	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/k8s"
	stats "github.com/lyft/gostats"

	// internal libraries: github.com/datawire/apro
	"github.com/datawire/apro/cmd/amb-sidecar/acmeclient"
	"github.com/datawire/apro/cmd/amb-sidecar/banner"
	devportalcontent "github.com/datawire/apro/cmd/amb-sidecar/devportal/content"
	devportalserver "github.com/datawire/apro/cmd/amb-sidecar/devportal/server"
	"github.com/datawire/apro/cmd/amb-sidecar/events"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	filterhandler "github.com/datawire/apro/cmd/amb-sidecar/filters/handler"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/health"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/limiter"
	rls "github.com/datawire/apro/cmd/amb-sidecar/ratelimits"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/cmd/amb-sidecar/watt"
	"github.com/datawire/apro/cmd/amb-sidecar/webui"
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/datawire/apro/lib/metriton"
	"github.com/datawire/apro/lib/util"

	// internal libraries: github.com/lyft/ratelimit
	lyftconfig "github.com/lyft/ratelimit/src/config"
	lyftredis "github.com/lyft/ratelimit/src/redis"
	lyftserver "github.com/lyft/ratelimit/src/server"
	lyftservice "github.com/lyft/ratelimit/src/service"

	// k8s clients
	k8sClientDynamic "k8s.io/client-go/dynamic"
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// gRPC service APIs
	rlsV1api "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v1"
	rlsV2api "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2"
	healthapi "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/datawire/apro/lib/filterapi"
)

var licenseClaims *licensekeys.LicenseClaimsLatest

var licenseWatch struct {
	Filename string
	Callback func()
}

var limit *limiter.LimiterImpl
var logrusLogger *logrus.Logger

func Main(version string) {
	argparser := &cobra.Command{
		Use:           os.Args[0],
		Version:       version,
		RunE:          runE,
		SilenceErrors: true, // we'll handle it after .Execute() returns
		SilenceUsage:  true, // our FlagErrorFunc wil handle it
	}

	cmdContext := licensekeys.InitializeCommandFlags(argparser.PersistentFlags())

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
		application := "ambassador-sidecar"
		keyCheck := func(reset bool) *licensekeys.LicenseClaimsLatest {
			claims, err := cmdContext.KeyCheck(cmd.PersistentFlags(), reset)
			if err != nil {
				logrusLogger.Errorln(err)
				claims = licensekeys.NewCommunityLicenseClaims()
				claims.CustomerID = "unregistered"
				limit.SetUnregisteredLicenseHardLimits(true)
			} else {
				limit.SetUnregisteredLicenseHardLimits(false)
			}
			limit.SetClaims(claims)
			go metriton.PhoneHome(claims, limit, application, version)
			return claims
		}

		limit = limiter.NewLimiterImpl()
		licenseClaims = keyCheck(false)
		go metriton.PhoneHomeEveryday(licenseClaims, limit, application, version)

		if cmdContext.Keyfile != "" {
			licenseWatch.Filename = cmdContext.Keyfile
			licenseWatch.Callback = func() {
				logrusLogger.Infof("Refreshing license key, %s changed", cmdContext.Keyfile)
				licenseClaims = keyCheck(true)
			}
		}
		return nil
	}

	err := argparser.Execute()
	if err != nil {
		logrusLogger.Errorln("shut down with error:", err)
		os.Exit(1)
	}
}

func triggerOnChange(ctx context.Context, watchFile string, trigger func()) {
	file := filepath.Clean(watchFile)
	dir, _ := filepath.Split(file)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrusLogger.Errorf("Failed to create watch on %s: Changes might require a restart: %v", file, err)
	}
	defer watcher.Close()

	eventsWG := sync.WaitGroup{}
	eventsWG.Add(1)
	go func() {
		defer eventsWG.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					if trigger != nil {
						trigger()
					}
				} else if filepath.Clean(event.Name) == file &&
					event.Op&fsnotify.Remove&fsnotify.Remove != 0 {
					return
				}
			case err, ok := <-watcher.Errors:
				if ok {
					logrusLogger.Errorln(err)
				}
				return
			}
		}
	}()

	logrusLogger.Infof("Creating watch on %s", dir)
	err = watcher.Add(dir)
	if err != nil {
		logrusLogger.Errorf("Failed to create watch on %s: Changes might require a restart: %v", dir, err)
	}
	eventsWG.Wait()
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
	logrusLogger.Info("Ambassador Edge Stack configuation loaded")

	if err := os.MkdirAll(filepath.Dir(cfg.RLSRuntimeDir), 0777); err != nil {
		return err
	}

	// cfg.LogLevel has already been validated in
	// ConfigFromEnv(), no need to error-check.
	level, _ := logrus.ParseLevel(cfg.LogLevel)
	logrusLogger.SetLevel(level)
	logrus.SetLevel(level) // FIXME(lukeshu): Some Lyft code still uses the global logger

	// FIXME(lukeshu): Find a way to hook klog in to our logger; client-go uses klog behind our back
	klogFlags := flag.NewFlagSet(os.Args[0], flag.PanicOnError)
	klog.InitFlags(klogFlags)
	klogFlags.Parse([]string{"-logtostderr=true", "-v=4"})

	kubeinfo := k8s.NewKubeInfo("", "", "") // Empty file/ctx/ns for defaults
	restconfig, err := kubeinfo.GetRestConfig()
	if err != nil {
		return err
	}
	coreClient, err := k8sClientCoreV1.NewForConfig(restconfig)
	if err != nil {
		return err
	}
	dynamicClient, err := k8sClientDynamic.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	eventLogger, err := events.NewEventLogger(
		cfg,
		coreClient,
		dlog.WrapLogrus(logrusLogger).WithField("MAIN", "event-broadcaster"),
	)
	if err != nil {
		return err
	}

	acmeLock, err := acmeclient.GetLeaderElectionResourceLock(cfg, kubeinfo, eventLogger)
	if err != nil {
		logrusLogger.Errorln("failed to participate in acme leader election, Ambassador Edge Stack acme client is disabled:", err)
	}

	var redisPool *pool.Pool
	var redisPoolErr error
	if cfg.RedisURL != "" {
		redisPool, redisPoolErr = pool.New(cfg.RedisSocketType, cfg.RedisURL, cfg.RedisPoolSize)
		if redisPoolErr != nil {
			return errors.Wrap(redisPoolErr, "redis pool configured but unavailable")
		}
	}
	if redisPool == nil {
		logrusLogger.Errorln("redis is not configured, Ambassador Edge Stack advanced features are disabled")
	}

	snapshotStore := watt.NewSnapshotStore(http.DefaultClient /* XXX */)

	// ... and then set the limiter's redis pool
	limit.SetRedisPool(redisPool)

	// Initialize the errgroup we'll use to orchestrate the goroutines.
	group := NewGroup(context.Background(), cfg, func(name string) dlog.Logger {
		return dlog.WrapLogrus(logrusLogger).WithField("MAIN", name)
	})

	// Launch all of the worker goroutines...
	//
	// softCtx is canceled for graceful shutdown, hardCtx is
	// canceled on not-so-graceful shutdown.  When in doubt, use
	// softCtx.

	if licenseWatch.Filename != "" {
		group.Go("license_refresh", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
			triggerOnChange(softCtx, licenseWatch.Filename, licenseWatch.Callback)
			return nil
		})
	}

	group.Go("watt_shutdown", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
		// Wait for shutdown to be initiated...
		<-softCtx.Done()
		// ... then signal snapshotStore.Subscribe()rs to shutdown.
		snapshotStore.Close()
		return nil
	})

	rls := rls.New()

	// RateLimit controller
	if limit.CanUseFeature(licensekeys.FeatureRateLimit) {
		group.Go("ratelimit_controller", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
			return rls.DoWatch(softCtx, cfg, kubeinfo, l)
		})
	}

	// Filter+FilterPolicy controller
	ct := &controller.Controller{}
	if limit.CanUseFeature(licensekeys.FeatureFilter) || limit.CanUseFeature(licensekeys.FeatureDevPortal) {
		group.Go("auth_controller", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
			ct.Config = cfg
			ct.Logger = l
			return ct.Watch(softCtx, kubeinfo, redisPool != nil)
		})
	}

	// DevPortal
	var devPortalServer *devportalserver.Server
	var devPortalContentVersion string
	if limit.CanUseFeature(licensekeys.FeatureDevPortal) {
		content, err := devportalcontent.NewContent(
			cfg.DevPortalContentURL,
			cfg.DevPortalContentBranch,
			cfg.DevPortalContentSubdir)
		if err != nil {
			return err
		}
		devPortalContentVersion = content.Config().Version
		devPortalServer = devportalserver.NewServer("/docs", content, limit)
		group.Go("devportal_fetcher", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
			fetcher := devportalserver.NewFetcher(devPortalServer, devportalserver.HTTPGet, devPortalServer.KnownServices(), cfg)
			fetcher.Run(softCtx)
			return nil
		})
	}

	// ACME client
	if acmeLock != nil {
		acmeController := acmeclient.NewController(
			redisPool,
			http.DefaultClient, // XXX
			snapshotStore.Subscribe(),
			eventLogger,
			acmeLock,
			coreClient,
			dynamicClient)
		group.Go("acme_client", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
			// FIXME(lukeshu): Perhaps EnsureFallback should observe softCtx.Done()?
			if err := acmeclient.EnsureFallback(cfg, coreClient, dynamicClient); err != nil {
				err = errors.Wrap(err, "create fallback TLSContext and TLS Secret")
				l.Errorln(err)
				// this is non fatal (mostly just to facilitate local dev); don't `return err`
			}
			return acmeController.Worker(dlog.WithLogger(softCtx, l))
		})
	}

	// HTTP server
	group.Go("http", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
		// A good chunk of this code mimics github.com/lyft/ratelimit/src/service_cmd/runner.Run()

		statsStore := stats.NewDefaultStore()
		statsStore.AddStatGenerator(stats.NewRuntimeStats(statsStore.Scope("go")))

		var redisPerSecondPool *pool.Pool
		var err error
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
		authService, err := filterhandler.NewFilterMux(cfg, l.WithField("SUB", "http-handler"), ct, coreClient, redisPool, limit)
		if err != nil {
			return errors.Wrap(err, "NewFilterMux")
		}
		filterapi.RegisterFilterService(grpcHandler, authService)
		httpHandler.AddEndpoint("/.ambassador/", "OAuth2 Filter", authService.ServeHTTP)

		// RateLimitService
		if redisPool != nil && licenseClaims.RequireFeature(licensekeys.FeatureRateLimit) == nil {
			rateLimitScope := statsStore.Scope("ratelimit")
			rateLimitService := lyftservice.NewService(
				loader.New(
					cfg.RLSRuntimeDir,               // runtime path
					cfg.RLSRuntimeSubdir,            // runtime subdirectory
					rateLimitScope.Scope("runtime"), // stats scope
					// empty line here because different versions of gofmt disagree :(
					&loader.SymlinkRefresher{RuntimePath: cfg.RLSRuntimeDir}, // refresher
				),
				lyftredis.NewRateLimitCacheImpl(
					lyftredis.NewPool(rateLimitScope.Scope("redis_pool"), redisPool),
					lyftredis.NewPool(rateLimitScope.Scope("redis_per_second_pool"), redisPerSecondPool),
					lyftredis.NewTimeSourceImpl(),
					rand.New(lyftredis.NewLockedSource(time.Now().Unix())),
					cfg.ExpirationJitterMaxSeconds),
				lyftconfig.NewRateLimitConfigLoaderImpl(),
				rateLimitScope.Scope("service"),
				limit)
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
			if devPortalContentVersion == "1" {
				httpHandler.AddEndpoint("/openapi/", "Documentation portal API", devPortalServer.Router().ServeHTTP)
			}
		}

		// web ui
		privkey, pubKey, err := secret.GetKeyPair(cfg, coreClient)
		if err != nil {
			err = errors.Wrap(err, "GetKeyPair")
			// this is non fatal (mostly just to facilitate local dev); don't `return err`
			l.Errorln("disabling webui JWT validation:", err)
		}
		webuiHandler := webui.New(
			cfg,
			dynamicClient,
			snapshotStore,
			rls,
			ct,
			privkey,
			pubKey,
			limit,
			redisPool,
		)
		httpHandler.AddEndpoint("/edge_stack_ui/", "Edge Stack admin UI", http.StripPrefix("/edge_stack_ui", webuiHandler).ServeHTTP)
		l.Debugf("DEV_WEBUI_PORT=%q", cfg.DevWebUIPort)
		if cfg.DevWebUIPort != "" {
			l.Infof("Serving webui on %q", ":"+cfg.DevWebUIPort)
			group.Go("webui_http", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
				return util.ListenAndServeHTTPWithContext(hardCtx, softCtx, &http.Server{
					Addr:     ":" + cfg.DevWebUIPort,
					ErrorLog: l.WithField("SUB", "webui-server").StdLogger(dlog.LogLevelError),
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						ctx := r.Context()
						ctx = dlog.WithLogger(ctx, l.WithField("SUB", "webui-server/handler"))
						ctx = middleware.WithRequestID(ctx, "unknown")
						r = r.WithContext(ctx)

						if r.URL.Path == "/_internal/v0/watt" {
							snapshotStore.ServeHTTP(w, r)
						} else {
							webuiHandler.ServeHTTP(w, r)
						}
					}),
				})
			})
		}

		httpHandler.AddEndpoint("/banner/", "Diag UI banner", http.StripPrefix("/banner", banner.NewBanner(limit, redisPool)).ServeHTTP)

		httpHandler.AddEndpoint("/_internal/v0/watt", "watt→post_update.py→this", snapshotStore.ServeHTTP)

		// Launch the server
		server := &http.Server{
			Addr:     ":" + cfg.HTTPPort,
			ErrorLog: l.WithField("SUB", "http-server").StdLogger(dlog.LogLevelError),
			// The net/http.Server doesn't support h2c (unencrypted
			// HTTP/2) built-in.  Since we want to have gRPC and plain
			// HTTP/1 on the same unencrypted port, we need h2c.
			// Fortunately, x/net has an h2c implementation we can use.
			Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				ctx = dlog.WithLogger(ctx, l.WithField("SUB", "http-server/handler"))
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
