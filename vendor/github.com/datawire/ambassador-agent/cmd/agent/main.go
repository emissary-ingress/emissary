package agent

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/datawire/ambassador-agent/pkg/agent"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/busy"
	"github.com/emissary-ingress/emissary/v3/pkg/logutil"
)

// internal k8s service
const (
	AdminDiagnosticsPort     = 8877
	DefaultSnapshotURLFmt    = "http://ambassador-admin:%d/snapshot-external"
	DefaultDiagnosticsURLFmt = "http://ambassador-admin:%d/ambassador/v0/diag/?json=true"

	ExternalSnapshotPort = 8005
)

// HACK to allow main to be imported by emissary-ingress
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

func run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	ambAgent := agent.NewAgent(
		nil, agent.NewArgoRolloutsGetter, agent.NewSecretsGetter,
	)

	// all log things need to happen here because we still allow the agent to run in amb-sidecar
	// and amb-sidecar should control all the logging if it's kicking off the agent.
	// this codepath is only hit when the agent is running on its own
	logLevel := os.Getenv("AES_LOG_LEVEL")
	// by default, suppress everything except fatal things
	// the watcher in the agent will spit out a lot of errors because we don't give it rbac to
	// list secrets initially.
	klogLevel := 3
	if logLevel != "" {
		logrusLevel, err := logutil.ParseLogLevel(logLevel)
		if err != nil {
			dlog.Errorf(ctx, "error parsing log level, running with default level: %+v", err)
		} else {
			busy.SetLogLevel(logrusLevel)
		}
		klogLevel = logutil.LogrusToKLogLevel(logrusLevel)
	}
	klogFlags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	klog.InitFlags(klogFlags)
	if err := klogFlags.Parse([]string{fmt.Sprintf("-stderrthreshold=%d", klogLevel), "-v=2", "-logtostderr=false"}); err != nil {
		return err
	}
	snapshotURL := os.Getenv("AES_SNAPSHOT_URL")
	if snapshotURL == "" {
		snapshotURL = fmt.Sprintf(DefaultSnapshotURLFmt, ExternalSnapshotPort)
	}

	diagnosticsURL := os.Getenv("AES_DIAGNOSTICS_URL")
	if diagnosticsURL == "" {
		diagnosticsURL = fmt.Sprintf(DefaultDiagnosticsURLFmt, AdminDiagnosticsPort)
	}

	reportDiagnostics := os.Getenv("AES_REPORT_DIAGNOSTICS_TO_CLOUD")
	if reportDiagnostics == "true" {
		ambAgent.SetReportDiagnosticsAllowed(true)
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
		return ambAgent.Watch(ctx, snapshotURL, diagnosticsURL)
	})

	return grp.Wait()
}
