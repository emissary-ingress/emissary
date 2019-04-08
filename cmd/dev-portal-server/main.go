package main

import (
	"github.com/datawire/apro/cmd/dev-portal-server/server"
	"os"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func main() {
	var diagdURL, ambassadorURL, publicURL string
	var set bool
	diagdURL, set = os.LookupEnv("DIAGD_URL")
	if !set {
		// Typically will be run in same Pod; customizable only in order
		// to support running outside of Kubernetes.
		diagdURL = "http://localhost:8877"
	}
	ambassadorURL, set = os.LookupEnv("AMBASSADOR_URL")
	if !set {
		// The default service name and namespace for Ambassador:
		ambassadorURL = "http://ambassador.default"
	}
	publicURL, set = os.LookupEnv("PUBLIC_API_URL")
	if !set {
		// We need whoever is installing the Dev Portal to supply this,
		// but since it ends up in documentation only it's OK to have a
		// placeholder.
		publicURL = "https://api.example.com"
	}
	server.Main(Version, diagdURL, ambassadorURL, publicURL)
}
