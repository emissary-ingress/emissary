package entrypoint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func runEnvoy(ctx context.Context, envoyHUP chan os.Signal) {
	// Wait until we get a SIGHUP to start envoy.
	//var bootstrap string
	select {
	case <-envoyHUP:
		/*bytes, err := ioutil.ReadFile(GetEnvoyBootstrapFile())
		if err != nil {
			panic(err)
		}
		bootstrap = string(bytes)*/
	case <-ctx.Done():
		return
	}

	// Try to run envoy directly, but fallback to running it inside docker if there is
	// no envoy executable available.
	var cmd *exec.Cmd
	// For some reason docker only sometimes passes the signal onto the process inside
	// the container, so we setup this cleanup function so that in the docker case we
	// can do a docker kill, just to be sure it is really dead and we don't leave an
	// envoy lying around.
	var dieharder func()
	if IsEnvoyAvailable() {
		cmd = subcommand(ctx, "envoy", GetEnvoyFlags()...)
		dieharder = func() {}
	} else {
		// Create a label unique to this invocation so we can use it to do a docker
		// kill for cleanup.
		label := fmt.Sprintf("amb-envoy-label-%d", os.Getpid())
		// XXX: will host networking work on a mac? (probably not)
		snapdir := GetSnapshotDir()
		cmd = subcommand(ctx, "docker", append([]string{"run", "-l", label, "--rm", "--network", "host",
			"-v", fmt.Sprintf("%s:%s", snapdir, snapdir),
			"-v", fmt.Sprintf("%s:%s", GetEnvoyBootstrapFile(), GetEnvoyBootstrapFile()),
			"--entrypoint", "envoy", "docker.io/datawire/aes:1.6.2"},
			GetEnvoyFlags()...)...)
		dieharder = func() {
			cids := cidsForLabel(label)
			if len(cids) == 0 {
				return
			}

			// Give the container two seconds to exit
			tctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
			wait := subcommand(tctx, "docker", append([]string{"wait"}, cids...)...)
			wait.Stdout = nil
			logExecError("docker wait", wait.Run())

			cids = cidsForLabel(label)

			if len(cids) > 0 {
				kill := subcommand(context.Background(), "docker", append([]string{"kill"}, cids...)...)
				kill.Stdout = nil
				logExecError("docker kill", kill.Run())
			}
		}
	}
	if envbool("DEV_SHUTUP_ENVOY") {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}
	err := cmd.Run()
	defer dieharder()
	logExecError("envoy exited", err)
}
