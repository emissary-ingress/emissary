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

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/limiter"
	"github.com/datawire/ambassador/pkg/supervisor"
)

// Version holds the version of the code. This is intended to be overridden at build time.
var Version = "(unknown version)"

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

func Main() {
	var flags wattFlags

	rootCmd := &cobra.Command{
		Use:           "watt",
		Short:         "watt - watch all the things",
		SilenceErrors: true, // we'll handle it after .Execute() returns
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

	ctx := context.Background()

	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runWatt(ctx, flags, args)
	}

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		if err == nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "%s\nSee '%s --help'.\n", err, cmd.CommandPath())
		os.Exit(2)
		return nil
	})

	if err := rootCmd.Execute(); err != nil {
		dlog.GetLogger(ctx).Errorln(err)
		os.Exit(1)
	}
}

func runWatt(ctx context.Context, flags wattFlags, args []string) error {
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

	dlog.GetLogger(ctx).Printf("starting watt...")

	// The aggregator sends the current consul resolver set to the
	// consul watch manager.
	aggregatorToConsulwatchmanCh := make(chan []ConsulWatchSpec, 100)

	// The aggregator sends the current k8s watch set to the
	// kubernetes watch manager.
	aggregatorToKubewatchmanCh := make(chan []KubernetesWatchSpec, 100)

	apiServerAuthority := flags.listenAddress
	if strings.HasPrefix(apiServerAuthority, ":") {
		apiServerAuthority = "localhost" + apiServerAuthority
	}
	invoker := NewInvoker(apiServerAuthority, flags.notifyReceivers)
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
	aggregator := NewAggregator(invoker.Snapshots, aggregatorToKubewatchmanCh, aggregatorToConsulwatchmanCh,
		flags.initialSources, ExecWatchHook(flags.watchHooks), limiter, validator)

	kubebootstrap := kubebootstrap{
		namespace:      flags.kubernetesNamespace,
		kinds:          flags.initialSources,
		fieldSelector:  flags.initialFieldSelector,
		labelSelector:  flags.initialLabelSelector,
		kubeAPIWatcher: kubeAPIWatcher,
		notify:         []chan<- k8sEvent{aggregator.KubernetesEvents},
	}

	consulwatchman := consulwatchman{
		WatchMaker: &ConsulWatchMaker{aggregatorCh: aggregator.ConsulEvents},
		watchesCh:  aggregatorToConsulwatchmanCh,
		watched:    make(map[string]*supervisor.Worker),
	}

	kubewatchman := kubewatchman{
		WatchMaker: &KubernetesWatchMaker{kubeAPI: client, notify: aggregator.KubernetesEvents},
		in:         aggregatorToKubewatchmanCh,
	}

	apiServer := &apiServer{
		listenNetwork: flags.listenNetwork,
		listenAddress: flags.listenAddress,
		invoker:       invoker,
	}

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
		Work: invoker.Work,
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
