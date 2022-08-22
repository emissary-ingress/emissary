package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/envoytest"
)

func runEnvoy(ctx context.Context, envoyHUP chan os.Signal) error {
	// Wait until we get a SIGHUP to start envoy.
	//var bootstrap string
	select {
	case <-ctx.Done():
		return nil
	case <-envoyHUP:
	}

	// Try to run envoy directly, but fallback to running it inside docker if there is
	// no envoy executable available.
	if IsEnvoyAvailable() {
		cmd := subcommand(ctx, "envoy", GetEnvoyFlags()...)
		if envbool("DEV_SHUTUP_ENVOY") {
			cmd.Stdout = nil
			cmd.Stderr = nil
		}
		return cmd.Run()
	} else {
		// For some reason docker only sometimes passes the signal onto the process inside
		// the container, so we setup this cleanup function so that in the docker case we
		// can do a docker kill, just to be sure it is really dead and we don't leave an
		// envoy lying around.

		// Create a label unique to this invocation so we can use it to do a docker
		// kill for cleanup.
		label := fmt.Sprintf("amb-envoy-label-%d", os.Getpid())

		snapdir := GetSnapshotDir()

		group := dgroup.NewGroup(ctx, dgroup.GroupConfig{
			ShutdownOnNonError: true,
		})
		group.Go("envoy", func(ctx context.Context) error {
			dockerFlags := []string{
				"--label=" + label,
				// FIXME(lukeshu): --network=host doesn't work on macOS, so that
				// will inconvenience our macOS-using contributors.
				"--network=host",
				"--volume=" + snapdir + ":" + snapdir,
				"--volume=" + GetEnvoyBootstrapFile() + ":" + GetEnvoyBootstrapFile(),
			}
			if envbool("DEV_ENVOY_DOCKER_PRIVILEGED") {
				dockerFlags = append(dockerFlags,
					"--privileged")
			}
			cmd, err := envoytest.LocalEnvoyCmd(ctx, dockerFlags, GetEnvoyFlags())
			if err != nil {
				return err
			}
			return cmd.Run()
		})
		group.Go("cleanup", func(ctx context.Context) error {
			<-ctx.Done()
			ctx = dcontext.HardContext(ctx)

			for {
				cids, err := cidsForLabel(ctx, label)
				if err != nil {
					return err
				}
				if len(cids) == 0 {
					return nil
				}

				// Give the container one second to exit
				tctx, cancel := context.WithTimeout(ctx, 1*time.Second)
				defer cancel()
				wait := subcommand(tctx, "docker", append([]string{"wait"}, cids...)...)
				wait.Stdout = nil
				if err := wait.Run(); err != nil {
					var exitErr *dexec.ExitError
					if errors.As(err, &exitErr) {
						dlog.Errorf(ctx, "docker wait: %v\n%s", err, string(exitErr.Stderr))
					} else {
						return err
					}
				}

				cids, err = cidsForLabel(ctx, label)
				if err != nil {
					return err
				}
				if len(cids) == 0 {
					return nil
				}

				kill := subcommand(ctx, "docker", append([]string{"kill"}, cids...)...)
				kill.Stdout = nil
				if err := wait.Run(); err != nil {
					var exitErr *dexec.ExitError
					if errors.As(err, &exitErr) {
						dlog.Errorf(ctx, "docker kill: %v\n%s", err, string(exitErr.Stderr))
					} else {
						return err
					}
				}
			}
		})
		return group.Wait()
	}
}
