// Ambassador combines the various Golang binaries used in the Ambassador
// container, dispatching on os.Args[0] like BusyBox. Note that the
// capabilities_wrapper binary is _not_ included here. That one has special
// permissions magic applied to it that is not appropriate for these other
// binaries.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/datawire/ambassador/v2/pkg/busy"
	"github.com/datawire/ambassador/v2/pkg/environment"

	amb_agent "github.com/datawire/ambassador/v2/cmd/agent"
	"github.com/datawire/ambassador/v2/cmd/ambex"
	"github.com/datawire/ambassador/v2/cmd/apiext"
	"github.com/datawire/ambassador/v2/cmd/entrypoint"
	metricssink "github.com/datawire/ambassador/v2/cmd/example-envoy-metrics-sink"
	"github.com/datawire/ambassador/v2/cmd/kubestatus"
	"github.com/datawire/ambassador/v2/cmd/reproducer"
)

// Version is inserted at build-time using --ldflags -X
var Version = "(unknown version)"

func noop(_ context.Context) {}

// Builtin for showing this image's version.
func showVersion(ctx context.Context, Version string, args ...string) error {
	fmt.Printf("Version %s\n", Version)

	return nil
}

func main() {
	// Allow ambassador.version to override the compiled-in Version.
	//
	// "Wait wait wait wait wait," I hear you cry. "Why in the world are you
	// doing this??" Two reasons:
	//
	// 1. ambassador.version is updated during the RC and GA process to always
	//    contain the Most Polite Version of the version number -- this is the
	//    ONE thing that should be shown to users.
	// 2. We do _not_ recompile busyambassador during the RC and GA process, and
	//    we don't want to: we want to ship the bits we tested, and while we're
	//    OK with altering a text file after that, recompiling feels weirder. So
	//    we don't.
	//
	// End result: fall back on the compiled-in version, but let ambassador.version
	// be the primary.

	// THIS IS A CLOSURE CALL, not just an anonymous function definition. Making the
	// call lets me defer file.Close().
	func() {
		file, err := os.Open("/buildroot/ambassador/python/ambassador.version")

		if err != nil {
			// We DON'T log errors here; we just silently fall back to the
			// compiled-in version. This is because the code in main() here is
			// running _wicked_ early, and logging setup happens _after_ this
			// function.
			//
			// XXX Letting the logging setup happen here, instead, would likely
			// be an improvement.
			return
		}

		defer file.Close()

		// Read line by line and hunt for BUILD_VERSION.
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "BUILD_VERSION=") {
				// The BUILD_VERSION line should be e.g.
				//
				// BUILD_VERSION="2.0.4-rc.2"
				//
				// so... cheat. Split on " and look for the second field.
				v := strings.Split(line, "\"")

				// If we don't get exactly three fields, though, something
				// is wrong and we'll give up.
				if len(v) == 3 {
					Version = v[1]
				}
				// See comments toward the top of this function for why there's no
				// logging here.
				// else {
				// 	fmt.Printf("VERSION OVERRIDE: got %#v ?", v)
				// }
			}
		}

		// Again, see comments toward the top of this function for why there's no
		// logging here.
		// if err := scanner.Err(); err != nil {
		// 	fmt.Printf("VERSION OVERRIDE: scanner error %s", err)
		// }
	}()

	busy.Main("busyambassador", "Ambassador", Version, map[string]busy.Command{
		"ambex":       {Setup: environment.EnvironmentSetupEntrypoint, Run: ambex.Main},
		"kubestatus":  {Setup: environment.EnvironmentSetupEntrypoint, Run: kubestatus.Main},
		"entrypoint":  {Setup: noop, Run: entrypoint.Main},
		"reproducer":  {Setup: noop, Run: reproducer.Main},
		"agent":       {Setup: environment.EnvironmentSetupEntrypoint, Run: amb_agent.Main},
		"metricssink": {Setup: environment.EnvironmentSetupEntrypoint, Run: metricssink.Main},
		"version":     {Setup: noop, Run: showVersion},
		"apiext":      {Setup: noop, Run: apiext.Main},
	})
}
