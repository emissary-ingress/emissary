package runner

import (
	"context"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/teleproxy/pkg/k8s"

	"github.com/datawire/apro/cmd/amb-sidecar/filters/app"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/controller"
	"github.com/datawire/apro/cmd/amb-sidecar/rls"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/util"
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
		restconfig, err := kubeinfo.GetRestConfig()
		if err != nil {
			return err
		}
		coreClient, err := k8sClientCoreV1.NewForConfig(restconfig)
		if err != nil {
			return err
		}
		httpHandler, err := app.NewFilterMux(cfg, l.WithField("SUB", "http-handler"), ct, coreClient)
		if err != nil {
			return err
		}
		server := &http.Server{
			Addr:     ":" + cfg.AuthPort,
			Handler:  httpHandler,
			ErrorLog: l.WithField("SUB", "http-server").StdLogger(types.LogLevelError),
		}
		return util.ListenAndServeHTTPWithContext(hardCtx, softCtx, server)
	})

	// And now we wait.
	return group.Wait()
}
