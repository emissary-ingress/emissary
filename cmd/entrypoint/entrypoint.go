package entrypoint

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/datawire/ambassador/cmd/ambex"
)

func Main() {

	// TODO:
	//  - CRD optionality
	//  - add validation back
	//  - figure out a better way to get the envoy image
	//  - logging
	//  - error/crash analysis
	//  - how to get errors to users?
	//  - by not watch-hooking are we missing error checking?
	//  - fork e2e tests

	log.Println("Started")

	pec := "PYTHON_EGG_CACHE"
	if os.Getenv(pec) == "" {
		os.Setenv(pec, path.Join(GetAmbassadorConfigBaseDir(), ".cache"))
	}
	os.Setenv("PYTHONUNBUFFERED", "true")

	ensureDir(GetAmbassadorConfigBaseDir())

	// TODO: --demo

	// TODO: AMBASSADOR_CLUSTER_ID

	// TODO: demo_chimed

	ensureDir(GetSnapshotDir())
	ensureDir(GetEnvoyDir())

	// stopped at WORKER: traffic-agent

	// We use this to wait until the bootstrap config has been written before starting envoy.
	envoyHUP := make(chan os.Signal, 1)
	signal.Notify(envoyHUP, syscall.SIGHUP)

	group := NewGroup(context.Background(), 10*time.Second)

	group.Go("diagd", func(ctx context.Context) {
		cmd := subcommand(ctx, "diagd", GetDiagdArgs()...)
		if envbool("DEV_SHUTUP_DIAGD") {
			cmd.Stdout = nil
			cmd.Stderr = nil
		}
		err := cmd.Run()
		logExecError("diagd", err)
	})

	group.Go("ambex", func(ctx context.Context) {
		err := flag.CommandLine.Parse([]string{"--ads-listen-address", "127.0.0.1:8003", GetEnvoyDir()})
		if err != nil {
			panic(err)
		}
		ambex.MainContext(ctx)
	})

	group.Go("envoy", func(ctx context.Context) { runEnvoy(ctx, envoyHUP) })

	snapshot := &atomic.Value{}
	group.Go("snapshot_server", func(ctx context.Context) {
		snapshotServer(ctx, snapshot)
	})
	group.Go("watcher", func(ctx context.Context) {
		watcher(ctx, snapshot)
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
		group.Go(sidecar.Name(), func(ctx context.Context) {
			cmd := subcommand(ctx, path.Join(sidecarDir, sidecar.Name()))
			err := cmd.Run()
			logExecError(sidecar.Name(), err)
		})
	}

	results := group.Wait()
	for name, value := range results {
		log.Printf("%s: %s", name, value)
	}
}
