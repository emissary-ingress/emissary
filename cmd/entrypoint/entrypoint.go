package entrypoint

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/datawire/ambassador/cmd/ambex"
	"github.com/datawire/ambassador/pkg/acp"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/memory"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
)

// This is the main ambassador entrypoint. It launches and manages two other
// processes:
//
//  1. The diagd process.
//  2. Envoy
//
// The entrypoint process manages two other goroutines:
//
//  1. The watcher goroutine that watches for changes in ambassador inputs and
//     notifies diagd.
//  2. The ambex goroutine that feeds envoy configuration updates via ADS.
//
// Dataflow Diagram
//
//   Kubernetes Watches
//          |
//          | (k8s resources, subscription)
//          |
//         \|/               consul endpoints, subscription)
//     entrypoint[watcher]<----------------------------------- Consul Watches
//          |
//          | (Snapshot, POST)
//          |
//         \|/
//        diagd
//          |
//          | (envoy config resources, pushed via writing files + SIGHUP)
//          |
//         \|/
//     entrypoint[ambex]
//          |
//          | (envoy config resources, ADS subscription)
//          |
//         \|/
//        envoy
//
// Notation:
//
//   The arrows point in the direction that data flows. Each arrow is labeled
//   with a tuple of the data type, and a short description of the nature of
//   communication.
//
// The golang entrypoint process assembles all the ambassador inputs from
// kubernetes and consul. When it has a complete/consistent set of inputs, it
// passes the complete snapshot of inputs along to diagd along with a list of
// deltas and invalid objects. This snapshot is fully detailed in snapshot.go
//
// The entrypoint goes to some trouble to ensure shared fate between all three
// processes as well as all the goroutines it manages, i.e. if any one of them
// dies for any reason, the whole process will shutdown and some larger process
// manager (e.g. kubernetes) is expected to take note and restart if
// appropriate.
func Main(ctx context.Context, Version string, args ...string) error {

	// TODO:
	//  - figure out a better way to get the envoy image
	//  - logging
	//  - error/crash analysis
	//  - how to get errors to users?
	//  - fork e2e tests

	log.Println("Started Ambassador")

	demoMode := false

	// XXX Yes, this is a disgusting hack. We can switch to a legit argument
	// parser later, when we have a second argument.
	if (len(args) == 1) && (args[0] == "--demo") {
		// Demo mode!
		log.Printf("DEMO MODE")
		demoMode = true
	}

	clusterID := GetClusterID(ctx)
	os.Setenv("AMBASSADOR_CLUSTER_ID", clusterID)
	log.Printf("AMBASSADOR_CLUSTER_ID=%s", clusterID)

	pec := "PYTHON_EGG_CACHE"
	if os.Getenv(pec) == "" {
		os.Setenv(pec, path.Join(GetAmbassadorConfigBaseDir(), ".cache"))
	}
	os.Setenv("PYTHONUNBUFFERED", "true")

	ensureDir(GetAmbassadorConfigBaseDir())
	ensureDir(GetSnapshotDir())
	ensureDir(GetEnvoyDir())

	// We use this to wait until the bootstrap config has been written before starting envoy.
	envoyHUP := make(chan os.Signal, 1)
	signal.Notify(envoyHUP, syscall.SIGHUP)

	// Go ahead and create an AmbassadorWatcher now, since we'll need it later.
	ambwatch := acp.NewAmbassadorWatcher(acp.NewEnvoyWatcher(), acp.NewDiagdWatcher())

	group := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		EnableSignalHandling: true,
		SoftShutdownTimeout:  10 * time.Second,
		HardShutdownTimeout:  10 * time.Second,
	})

	// Demo mode: start the demo services. Starting the demo stuff first is
	// kind of important: it's nice to give them a chance to start running before
	// Ambassador really gets running.
	if demoMode {
		bootDemoMode(ctx, group, ambwatch)
	}

	group.Go("diagd", func(ctx context.Context) error {
		cmd := subcommand(ctx, "diagd", GetDiagdArgs(ctx, demoMode)...)
		if envbool("DEV_SHUTUP_DIAGD") {
			cmd.Stdout = nil
			cmd.Stderr = nil
		}
		return cmd.Run()
	})

	usage := memory.GetMemoryUsage()
	group.Go("memory", func(ctx context.Context) error {
		usage.Watch(ctx)
		return nil
	})

	group.Go("ambex", func(ctx context.Context) error {
		return ambex.Main2(ctx, Version, usage.PercentUsed, "--ads-listen-address", "127.0.0.1:8003", GetEnvoyDir())
	})

	group.Go("envoy", func(ctx context.Context) error {
		return runEnvoy(ctx, envoyHUP)
	})

	snapshot := &atomic.Value{}
	group.Go("snapshot_server", func(ctx context.Context) error {
		return snapshotServer(ctx, snapshot)
	})

	if !demoMode {
		group.Go("watcher", func(ctx context.Context) error {
			// We need to pass the AmbassadorWatcher to this (Kubernetes/Consul) watcher, so
			// that it can tell the AmbassadorWatcher when snapshots are posted.
			watcher(ctx, ambwatch, snapshot)
			return nil
		})
	}

	// Finally, fire up the health check handler.
	group.Go("healthchecks", func(ctx context.Context) error {
		return healthCheckHandler(ctx, ambwatch)
	})

	// Launch every file in the sidecar directory. Note that this is "bug compatible" with
	// entrypoint.sh for now, e.g. we don't check execute bits or anything like that.
	sidecarDir := "/ambassador/sidecars"
	sidecars, err := ioutil.ReadDir(sidecarDir)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	}
	for _, sidecar := range sidecars {
		group.Go(sidecar.Name(), func(ctx context.Context) error {
			cmd := subcommand(ctx, path.Join(sidecarDir, sidecar.Name()))
			return cmd.Run()
		})
	}

	return group.Wait()
}

func clusterIDFromRootID(rootID string) string {
	clusterUrl := fmt.Sprintf("d6e_id://%s/%s", rootID, GetAmbassadorId())
	uid := uuid.NewSHA1(uuid.NameSpaceURL, []byte(clusterUrl))

	return strings.ToLower(uid.String())
}

func GetClusterID(ctx context.Context) (clusterID string) {
	clusterID = env("AMBASSADOR_CLUSTER_ID", env("AMBASSADOR_SCOUT_ID", ""))
	if clusterID != "" {
		return clusterID
	}

	rootID := "00000000-0000-0000-0000-000000000000"

	// *!&@*#!& kates panics.
	defer func() {
		err := recover()

		if err != nil {
			// Sigh.
			dlog.Errorf(ctx, "GetClusterID: couldn't get ID: %s", err)
			clusterID = clusterIDFromRootID(rootID)
		}
	}()

	client, err := kates.NewClient(kates.ClientConfig{})
	if err == nil {
		nsName := "default"
		if IsAmbassadorSingleNamespace() {
			nsName = GetAmbassadorNamespace()
		}
		ns := &kates.Namespace{
			TypeMeta:   kates.TypeMeta{Kind: "Namespace"},
			ObjectMeta: kates.ObjectMeta{Name: nsName},
		}

		err := client.Get(ctx, ns, ns)
		if err == nil {
			rootID = string(ns.GetUID())
		}
	}

	return clusterIDFromRootID(rootID)
}
