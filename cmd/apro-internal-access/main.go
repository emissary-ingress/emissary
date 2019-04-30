package main

import (
	"github.com/datawire/apro/cmd/apro-internal-access/server"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func main() {
	// TODO Do license enforcement.
	server.Main(Version, "/etc/apro-internal-access/shared-secret")
}
