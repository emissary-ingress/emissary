package entrypoint

import (
	"context"
	"net/http"
	"time"

	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/acp"
)

// bootDemoMode: start demo mode running. This is more obnoxious than one might
// think, since we have to start our two demo services _and_ we have to prime
// the acp.AmbassadorWatcher so that health checks work.
func bootDemoMode(ctx context.Context, group *dgroup.Group, ambwatch *acp.AmbassadorWatcher) {
	group.Go("demo_auth", func(ctx context.Context) error {
		cmd := subcommand(ctx, "/usr/bin/python3", "/ambassador/demo-services/auth.py")
		return cmd.Run()
	})

	group.Go("demo_qotm", func(ctx context.Context) error {
		cmd := subcommand(ctx, "/usr/bin/python3", "/ambassador/demo-services/qotm.py")
		return cmd.Run()
	})

	// XXX More disgusting hackery. Here's the thing: in demo mode, we don't actually
	// push any snapshots through to diagd, because diagd just reads its config from
	// the filesystem. Since we don't push any snapshots, the acp.AmbassadorWatcher
	// never decides that Ambassador is ready.
	//
	// To fake that out, we're going to wait for Envoy to be alive, then just inform
	// the acp.AmbassadorWatcher that _clearly_ we've pushed and processed a snapshot.
	// Ewwww.
	go func() {
		for {
			resp, err := http.Get("http://localhost:8001/ready")

			if err == nil {
				if resp.StatusCode == 200 {
					// Done! Tell the acp.AmbassadorWatcher that we've sent and processed
					// a snapshot, so that it'll actually take Ambassador to ready status.
					ambwatch.NoteSnapshotSent()
					time.Sleep(5 * time.Millisecond)
					ambwatch.NoteSnapshotProcessed()
					break
				}
			}

			time.Sleep(5 * time.Second)
		}
	}()

	go func() {
		// Wait for Ambassador to claim that it's ready.
		for {
			resp, err := http.Get("http://localhost:8877/ambassador/v0/check_ready")

			if err == nil {
				if resp.StatusCode == 200 {
					// Done! Spit out the magic string that the demo test needs to see.
					dlog.Infof(ctx, "AMBASSADOR DEMO RUNNING")
					break
				}
			}

			time.Sleep(5 * time.Second)
		}
	}()
}
