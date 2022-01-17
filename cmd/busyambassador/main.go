// Ambassador combines the various Golang binaries used in the Ambassador
// container, dispatching on os.Args[0] like BusyBox. Note that the
// capabilities_wrapper binary is _not_ included here. That one has special
// permissions magic applied to it that is not appropriate for these other
// binaries.
package main

import (
	// stdlib
	"context"
	"fmt"
	"os"
	"strings"

	// 3rd-party libs
	"github.com/kballard/go-shellquote"

	// 1st-party libs
	"github.com/datawire/ambassador/v2/pkg/busy"
	"github.com/datawire/ambassador/v2/pkg/environment"

	// commands
	"github.com/datawire/ambassador/v2/cmd/agent"
	"github.com/datawire/ambassador/v2/cmd/ambex"
	"github.com/datawire/ambassador/v2/cmd/apiext"
	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	"github.com/datawire/ambassador/v2/cmd/kubestatus"
	"github.com/datawire/ambassador/v2/cmd/reproducer"
)

func noop(_ context.Context) {}

// Builtin for showing this image's version.
func showVersion(ctx context.Context, version string, args ...string) error {
	fmt.Printf("Version %s\n", version)

	return nil
}

func main() {
	// The version number is set at run-time by reading the 'ambassador.version' file.  We do
	// this instead of compiling in a version so that we can promote RC images to GA without
	// recompiling anything.
	//
	// Keep this parsing logic in-sync with VERSION.py.
	version := "dirty"
	if verBytes, err := os.ReadFile("/buildroot/ambassador/python/ambassador.version"); err == nil {
		verLines := strings.Split(string(verBytes), "\n")
		for _, line := range verLines {
			if strings.HasPrefix(line, "BUILD_VERSION=") {
				vals, err := shellquote.Split(strings.TrimPrefix(line, "BUILD_VERSION="))
				if err == nil && len(vals) > 0 {
					version = vals[0]
				}
			}
		}
	}

	busy.Main("busyambassador", "Ambassador", version, map[string]busy.Command{
		"ambex":      {Setup: environment.EnvironmentSetupEntrypoint, Run: ambex.Main},
		"kubestatus": {Setup: environment.EnvironmentSetupEntrypoint, Run: kubestatus.Main},
		"entrypoint": {Setup: noop, Run: entrypoint.Main},
		"reproducer": {Setup: noop, Run: reproducer.Main},
		"agent":      {Setup: environment.EnvironmentSetupEntrypoint, Run: agent.Main},
		"version":    {Setup: noop, Run: showVersion},
		"apiext":     {Setup: noop, Run: apiext.Main},
	})
}
