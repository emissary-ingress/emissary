package runner

import (
	"context"
	"net/http"
	"os"
	"strings"

	// 3rd-party libraries
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	// first-party libraries
	"github.com/datawire/teleproxy/pkg/k8s"

	// internal libraries: github.com/datawire/apro
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/health"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/rls"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/util"

	// internal libraries: github.com/lyft/ratelimit
	lyftserver "github.com/lyft/ratelimit/src/server"

	// k8s clients
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// gRPC service APIs
	"github.com/datawire/apro/lib/filterapi"
)

func init() {
	argparser.AddCommand(&cobra.Command{
		Use:   "main",
		Short: "Run the main Ambassador Pro process",
		RunE:  cmdMain,
	})
}

func cmdMain(cmd *cobra.Command, args []string) error {
	// Initialize the root logger.  We'll use this for top-level
	// things that don't involve any specific worker process.
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	// Load the configuration
	cfg, warn, fatal := types.ConfigFromEnv()
	for _, err := range warn {
		l.Warnln("config error:", err)
	}
	for _, err := range fatal {
		l.Errorln("config error:", err)
	}
	if len(fatal) > 0 {
		return fatal[len(fatal)-1]
	}

	// cfg.LogLevel has already been validated in
	// ConfigFromEnv(), no need to error-check.
	level, _ := logrus.ParseLevel(cfg.LogLevel)
	l.SetLevel(level)

	kubeinfo, err := k8s.NewKubeInfo("", "", "") // Empty file/ctx/ns for defaults
	if err != nil {
		return err
	}

	// Initialize the errgroup we'll use to orchestrate the goroutines.
	group := NewGroup(context.Background(), cfg, func(name string) types.Logger {
		return types.WrapLogrus(l).WithField("MAIN", name)
	})

	// Launch all of the worker goroutines...

	// RateLimit controller
	if os.Getenv("REDIS_URL") != "" {
		group.Go("ratelimit_controller", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
			return rls.DoWatch(softCtx, cfg, l)
		})
	}

	// Filter+FilterPolicy controller
	ct := &controller.Controller{}
	group.Go("auth_controller", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
		ct.Config = cfg
		ct.Logger = l
		return ct.Watch(softCtx, kubeinfo)
	})

	// Auth HTTP server
	group.Go("auth_http", func(hardCtx, softCtx context.Context, cfg types.Config, l types.Logger) error {
		redisPool, err := pool.New(cfg.RedisSocketType, cfg.RedisURL, cfg.RedisPoolSize)
		if err != nil {
			return errors.Wrap(err, "redis pool")
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

		// AuthService
		restconfig, err := kubeinfo.GetRestConfig()
		if err != nil {
			return err
		}
		coreClient, err := k8sClientCoreV1.NewForConfig(restconfig)
		if err != nil {
			return err
		}
		authService, err := app.NewFilterMux(cfg, l.WithField("SUB", "http-handler"), ct, coreClient, redisPool)
		if err != nil {
			return err
		}
		filterapi.RegisterFilterService(grpcHandler, authService)

		// Launch the server
		server := &http.Server{
			Addr:     ":" + cfg.AuthPort,
			ErrorLog: l.WithField("SUB", "http-server").StdLogger(types.LogLevelError),
			// The net/http.Server doesn't support h2c (unencrypted
			// HTTP/2) built-in.  Since we want to have gRPC and plain
			// HTTP/1 on the same unencrypted port, we need h2c.
			// Fortunately, x/net has an h2c implementation we can use.
			Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
