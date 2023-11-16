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

	// 1st-party libs
	"github.com/emissary-ingress/emissary/v3/pkg/busy"
	"github.com/emissary-ingress/emissary/v3/pkg/environment"

	// commands
	"github.com/emissary-ingress/emissary/v3/cmd/apiext"
	"github.com/emissary-ingress/emissary/v3/cmd/entrypoint"
	"github.com/emissary-ingress/emissary/v3/cmd/kubestatus"
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
	//
	// We don't report or log errors here, we just silently fall back to a static "MISSING(XXX)"
	// string.  This is in-part because the code in main() here is running _wicked_ early, and
	// logging setup hasn't happened yet.  Also because any errors will be evident when the
	// version number gets logged and it's this static string.
	version := "MISSING(FILE)"
	if verBytes, err := os.ReadFile("/buildroot/ambassador/python/ambassador.version"); err == nil {
		verLines := strings.Split(string(verBytes), "\n")
		for len(verLines) < 2 {
			verLines = append(verLines, "MISSING(VAL)")
		}
		version = verLines[0]
	}

	busy.Main("busyambassador", "Ambassador", version, map[string]busy.Command{
		"kubestatus": {Setup: environment.EnvironmentSetupEntrypoint, Run: kubestatus.Main},
		"entrypoint": {Setup: noop, Run: entrypoint.Main},
		"version":    {Setup: noop, Run: showVersion},
		"apiext":     {Setup: noop, Run: apiext.Main},
	})
}
