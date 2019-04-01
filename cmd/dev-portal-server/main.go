package main

import (
	"github.com/datawire/apro/cmd/dev-portal-server/server"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func main() {
	server.Main(Version)
}
