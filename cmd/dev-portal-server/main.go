package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/datawire/apro/cmd/dev-portal-server/server"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func main() {
	// TODO Do license enforcement.
	var diagdURL, ambassadorURL, publicURL, pollEverySecsStr string
	var pollEverySecs time.Duration = 60 * time.Second
	var set bool
	diagdURL, set = os.LookupEnv("DIAGD_URL")
	if !set {
		// Typically will be run in same Pod; customizable only in order
		// to support running outside of Kubernetes.
		diagdURL = "http://localhost:" + os.Getenv("AMBASSADOR_ADMIN_SERVICE_PORT")
	}
	ambassadorURL, set = os.LookupEnv("AMBASSADOR_URL")
	if !set {
		// Ambassador's Envoy running in the same Pod:
		ambassadorURL = "http://localhost:" + os.Getenv("AMBASSADOR_SERVICE_PORT")
	}
	publicURL, set = os.LookupEnv("PUBLIC_API_URL")
	if !set {
		// We need whoever is installing the Dev Portal to supply this,
		// but since it ends up in documentation only it's OK to have a
		// placeholder.
		publicURL = "https://api.example.com"
	}
	pollEverySecsStr, set = os.LookupEnv("POLL_EVERY_SECS")
	if set {
		p, err := strconv.Atoi(pollEverySecsStr)
		if err == nil {
			pollEverySecs = time.Duration(p) * time.Second
		} else {
			log.Print(err)
		}
	}
	server.Main(Version, diagdURL, ambassadorURL, publicURL, pollEverySecs)
}
