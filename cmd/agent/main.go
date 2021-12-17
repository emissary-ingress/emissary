package agent

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	"github.com/datawire/ambassador/v2/pkg/agent"
	"github.com/datawire/ambassador/v2/pkg/busy"
	"github.com/datawire/ambassador/v2/pkg/logutil"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
	"github.com/datawire/envconfig"
)

type Config struct {
	LogLevel    string   `env:"AES_LOG_LEVEL    ,parser=logrus.ParseLevel ,default=info"`
	SnapshotURL *url.URL `env:"AES_SNAPSHOT_URL ,parser=absolute-URL      ,default=http://ambassador-admin:8005/snapshot-external"`
}

func ConfigFromEnv() (cfg Config, warn []error, fatal []error) {
	parser, err := envconfig.GenerateParser(reflect.TypeOf(Config{}), nil)
	if err != nil {
		// panic, because it means that the definition of
		// 'Config' is invalid, which is a bug, not a
		// runtime error.
		panic(err)
	}
	warn, fatal = parser.ParseFromEnv(&cfg)
	return
}

func run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cfg, warn, fatal := ConfigFromEnv()
	for _, err := range warn {
		dlog.Warnln(ctx, "config error:", err)
	}
	for _, err := range fatal {
		dlog.Errorln(ctx, "config error:", err)
	}
	if len(fatal) > 0 {
		return fatal[len(fatal)-1]
	}

	ambAgent := agent.NewAgent(nil, agent.NewArgoRolloutsGetter)

	logrusLevel, _ := logutil.ParseLogLevel(cfg.LogLevel)
	busy.SetLogLevel(logrusLevel)
	klogLevel = logutil.LogrusToKLogLevel(logrusLevel)

	klogFlags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	klog.InitFlags(klogFlags)
	if err := klogFlags.Parse([]string{fmt.Sprintf("-stderrthreshold=%d", klogLevel), "-v=2", "-logtostderr=false"}); err != nil {
		return err
	}

	metricsListener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}
	dlog.Info(ctx, "metrics service listening on :8080")

	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{})

	grp.Go("metrics-server", func(ctx context.Context) error {
		metricsServer := agent.NewMetricsServer(ambAgent.MetricsRelayHandler)
		return metricsServer.Serve(ctx, metricsListener)
	})

	grp.Go("watch", func(ctx context.Context) error {
		return ambAgent.Watch(ctx, snapshotURL)
	})

	return grp.Wait()
}

func Main(ctx context.Context, version string, args ...string) error {
	argparser := &cobra.Command{
		Use:           os.Args[0],
		Version:       version,
		RunE:          run,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	argparser.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err == nil {
			return nil
		}
		dlog.Errorf(ctx, "%s\nSee '%s --help'.\n", err, cmd.CommandPath())
		return nil
	})
	argparser.SetArgs(args)
	return argparser.ExecuteContext(ctx)
}
