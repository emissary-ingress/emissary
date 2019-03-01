package main

import (
	"github.com/datawire/apro/cmd/amb-sidecar/runner"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func main() {
	runner.Main(Version)
}
