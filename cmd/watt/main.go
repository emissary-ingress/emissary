package watt

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/datawire/ambassador/pkg/k8s"
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
	port                 int
	interval             time.Duration
	showVersion          bool
}

func Main() {
	var flags wattFlags

	rootCmd := &cobra.Command{
		Use:              "watt",
		Short:            "watt",
		Long:             "watt - watch all the things",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}

	rootCmd.Flags().StringVarP(&flags.kubernetesNamespace, "namespace", "n", "", "namespace to watch (default: all)")
	rootCmd.Flags().StringSliceVarP(&flags.initialSources, "source", "s", []string{}, "configure an initial static source")
	rootCmd.Flags().StringVar(&flags.initialFieldSelector, "fields", "", "configure an initial field selector string")
	rootCmd.Flags().StringVar(&flags.initialLabelSelector, "labels", "", "configure an initial label selector string")
	rootCmd.Flags().StringSliceVarP(&flags.watchHooks, "watch", "w", []string{}, "configure watch hook(s)")
	rootCmd.Flags().StringSliceVar(&flags.notifyReceivers, "notify", []string{},
		"invoke the program with the given arguments as a receiver")
	rootCmd.Flags().IntVarP(&flags.port, "port", "p", 7000, "configure the snapshot server port")
	rootCmd.Flags().DurationVarP(&flags.interval, "interval", "i", 250*time.Millisecond,
		"configure the rate limit interval")
	rootCmd.Flags().BoolVarP(&flags.showVersion, "version", "", false, "display version information")

	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		os.Exit(runWatt(flags, args))
	}

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func runWatt(flags wattFlags, args []string) int {
	if flags.showVersion {
		fmt.Println("watt", Version)
		return 0
	}

	if len(flags.initialSources) == 0 {
		log.Println("no initial sources configured")
		return 1
	}

	// XXX: we don't need to create this here anymore
	client, err := k8s.NewClient(nil)
	if err != nil {
		log.Println(err)
		return 1
	}
	kubeAPIWatcher := client.Watcher()
	/*for idx := range initialSources {
		initialSources[idx] = kubeAPIWatcher.Canonical(initialSources[idx])
	}*/

	log.Printf("starting watt...")

	// The aggregator sends the current consul resolver set to the
	// consul watch manager.
	aggregatorToConsulwatchmanCh := make(chan []ConsulWatchSpec, 100)

	// The aggregator sends the current k8s watch set to the
	// kubernetes watch manager.
	aggregatorToKubewatchmanCh := make(chan []KubernetesWatchSpec, 100)

	invoker := NewInvoker(flags.port, flags.notifyReceivers)
	limiter := limiter.NewComposite(limiter.NewUnlimited(), limiter.NewInterval(flags.interval), flags.interval)
	aggregator := NewAggregator(invoker.Snapshots, aggregatorToKubewatchmanCh, aggregatorToConsulwatchmanCh,
		flags.initialSources, ExecWatchHook(flags.watchHooks), limiter)

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
		port:    flags.port,
		invoker: invoker,
	}

	ctx := context.Background()
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
		log.Printf("ERROR(s): %s", strings.Join(msgs, "\n    "))
		return 1
	}

	return 0
}
