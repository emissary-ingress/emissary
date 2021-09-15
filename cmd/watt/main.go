package watt

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/datawire/ambassador/v2/cmd/watt/aggregator"
	"github.com/datawire/ambassador/v2/cmd/watt/invoker"
	"github.com/datawire/ambassador/v2/cmd/watt/thingconsul"
	"github.com/datawire/ambassador/v2/cmd/watt/thingkube"
	"github.com/datawire/ambassador/v2/cmd/watt/watchapi"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/limiter"
	"github.com/datawire/ambassador/v2/pkg/supervisor"
	"github.com/datawire/dlib/dlog"
)

type wattFlags struct {
	kubernetesNamespace  string
	initialSources       []string
	initialFieldSelector string
	initialLabelSelector string
	watchHooks           []string
	notifyReceivers      []string
	listenNetwork        string
	listenAddress        string
	legacyListenPort     int
	interval             time.Duration
	showVersion          bool
}

func Main(ctx context.Context, Version string, args ...string) error {
	var flags wattFlags

	rootCmd := &cobra.Command{
		Use:           "watt",
		Short:         "watt - watch all the things",
		SilenceErrors: true, // we'll handle it after .ExecuteContext() returns
		SilenceUsage:  true, // our FlagErrorFunc will handle it
	}

	rootCmd.Flags().StringVarP(&flags.kubernetesNamespace, "namespace", "n", "", "namespace to watch (default: all)")
	rootCmd.Flags().StringSliceVarP(&flags.initialSources, "source", "s", []string{}, "configure an initial static source")
	rootCmd.Flags().StringVar(&flags.initialFieldSelector, "fields", "", "configure an initial field selector string")
	rootCmd.Flags().StringVar(&flags.initialLabelSelector, "labels", "", "configure an initial label selector string")
	rootCmd.Flags().StringSliceVarP(&flags.watchHooks, "watch", "w", []string{}, "configure watch hook(s)")
	rootCmd.Flags().StringSliceVar(&flags.notifyReceivers, "notify", []string{},
		"invoke the program with the given arguments as a receiver")
	rootCmd.Flags().DurationVarP(&flags.interval, "interval", "i", 250*time.Millisecond,
		"configure the rate limit interval")
	rootCmd.Flags().BoolVarP(&flags.showVersion, "version", "", false, "display version information")

	rootCmd.Flags().StringVar(&flags.listenNetwork, "listen-network", "tcp", "network for the snapshot server to listen on")
	rootCmd.Flags().StringVar(&flags.listenAddress, "listen-address", ":7000", "address (on --listen-network) for the snapshot server to listen on")

	rootCmd.Flags().IntVarP(&flags.legacyListenPort, "port", "p", 0, "configure the snapshot server port")
	rootCmd.Flags().MarkHidden("port")

	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runWatt(cmd.Context(), Version, flags)
	}

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err == nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "%s\nSee '%s --help'.\n", err, cmd.CommandPath())
		os.Exit(2)
		return nil
	})

	rootCmd.SetArgs(args)
	return rootCmd.ExecuteContext(ctx)
}

func runWatt(ctx context.Context, Version string, flags wattFlags) error {
	if flags.showVersion {
		fmt.Println("watt", Version)
		return nil
	}

	if flags.legacyListenPort != 0 {
		flags.listenAddress = fmt.Sprintf(":%v", flags.legacyListenPort)
	}

	if len(flags.initialSources) == 0 {
		return errors.New("no initial sources configured")
	}

	// XXX: we don't need to create this here anymore
	client, err := k8s.NewClient(nil)
	if err != nil {
		return err
	}
	kubeAPIWatcher := client.Watcher()
	/*for idx := range initialSources {
		initialSources[idx] = kubeAPIWatcher.Canonical(initialSources[idx])
	}*/

	dlog.Printf(ctx, "starting watt...")

	// The aggregator sends the current consul resolver set to the
	// consul watch manager.
	aggregatorToConsulwatchmanCh := make(chan []watchapi.ConsulWatchSpec, 100)

	// The aggregator sends the current k8s watch set to the
	// kubernetes watch manager.
	aggregatorToKubewatchmanCh := make(chan []watchapi.KubernetesWatchSpec, 100)

	apiServerAuthority := flags.listenAddress
	if strings.HasPrefix(apiServerAuthority, ":") {
		apiServerAuthority = "localhost" + apiServerAuthority
	}
	invokerObj := invoker.NewInvoker(apiServerAuthority, flags.notifyReceivers)
	limiter := limiter.NewComposite(limiter.NewUnlimited(), limiter.NewInterval(flags.interval), flags.interval)

	crdYAML, err := ioutil.ReadFile("/opt/ambassador/etc/crds.yaml")
	if err != nil {
		return err
	}
	crdObjs, err := kates.ParseManifests(string(crdYAML))
	if err != nil {
		return err
	}
	validator, err := kates.NewValidator(nil, crdObjs)
	if err != nil {
		return err
	}
	aggregator := aggregator.NewAggregator(invokerObj.Snapshots, aggregatorToKubewatchmanCh, aggregatorToConsulwatchmanCh,
		flags.initialSources, aggregator.ExecWatchHook(flags.watchHooks), limiter, validator)

	kubebootstrap := thingkube.NewKubeBootstrap(
		flags.kubernetesNamespace,                                // namespace
		flags.initialSources,                                     // kinds
		flags.initialFieldSelector,                               // fieldSelector
		flags.initialLabelSelector,                               // labelSelector
		[]chan<- thingkube.K8sEvent{aggregator.KubernetesEvents}, // notify
		kubeAPIWatcher,                                           // kubeAPIWatcher
	)

	consulwatchman := thingconsul.NewConsulWatchMan(
		aggregator.ConsulEvents,
		aggregatorToConsulwatchmanCh,
	)

	kubewatchman := thingkube.NewKubeWatchMan(
		client,                      // k8s client
		aggregator.KubernetesEvents, // eventsCh
		aggregatorToKubewatchmanCh,  // watchesCh
	)

	apiServer := invoker.NewAPIServer(
		flags.listenNetwork,
		flags.listenAddress,
		invokerObj,
	)

	s := supervisor.WithContext(ctx)

	s.Supervise(&supervisor.Worker{
		Name: "kubebootstrap",
		Work: kubebootstrap.Work,
	})

	s.Supervise(&supervisor.Worker{
		Name: "consulwatchman",
		Work: consulwatchman.Work,
	})

	s.Supervise(&supervisor.Worker{
		Name: "kubewatchman",
		Work: kubewatchman.Work,
	})

	s.Supervise(&supervisor.Worker{
		Name: "aggregator",
		Work: aggregator.Work,
	})

	s.Supervise(&supervisor.Worker{
		Name: "invoker",
		Work: invokerObj.Work,
	})

	s.Supervise(&supervisor.Worker{
		Name: "api",
		Work: apiServer.Work,
	})

	if errs := s.Run(); len(errs) > 0 {
		msgs := []string{}
		for _, err := range errs {
			msgs = append(msgs, err.Error())
		}
		return fmt.Errorf("ERROR(s): %s", strings.Join(msgs, "\n    "))
	}

	return nil
}
