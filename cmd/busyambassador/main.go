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

	// 1st-party libs
	"github.com/emissary-ingress/emissary/v3/pkg/busy"
	"github.com/emissary-ingress/emissary/v3/pkg/environment"
	"github.com/emissary-ingress/emissary/v3/pkg/utils"

	// commands

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
	version := utils.GetVersion()

	busy.Main("busyambassador", "Ambassador", version, map[string]busy.Command{
		"kubestatus": {Setup: environment.EnvironmentSetupEntrypoint, Run: kubestatus.Main},
		"entrypoint": {Setup: noop, Run: entrypoint.Main},
		"version":    {Setup: noop, Run: showVersion},
	})
}
