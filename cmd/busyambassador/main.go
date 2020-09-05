// Ambassador combines the various Golang binaries used in the Ambassador
// container, dispatching on os.Args[0] like BusyBox. Note that the
// capabilities_wrapper binary is _not_ included here. That one has special
// permissions magic applied to it that is not appropriate for these other
// binaries.
package main

import (
	"github.com/datawire/ambassador/pkg/busy"

	"github.com/datawire/ambassador/cmd/ambex"
	"github.com/datawire/ambassador/cmd/entrypoint"
	"github.com/datawire/ambassador/cmd/kubestatus"
	"github.com/datawire/ambassador/cmd/watt"
)

// Version is inserted at build-time using --ldflags -X
var Version = "(unknown version)"

func main() {
	busy.Main("busyambassador", "Ambassador", Version, map[string]busy.Command{
		"ambex":      ambex.Main,
		"watt":       watt.Main,
		"kubestatus": kubestatus.Main,
		"entrypoint": entrypoint.Main,
	})
}
