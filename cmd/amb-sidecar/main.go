package amb_sidecar

import (
	"github.com/datawire/apro/cmd/amb-sidecar/runner"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func Main() {
	runner.Main(Version)
}
