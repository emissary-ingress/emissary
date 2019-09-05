package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/dev-portal-server/server"
	"github.com/datawire/apro/lib/licensekeys"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func licenseEnforce() {
	devportal := &cobra.Command{
		Use: "dev-portal-server [command]",
	}
	keycheck := licensekeys.InitializeCommandFlags(devportal.PersistentFlags(), "dev-portal-server", Version)
	devportal.SilenceUsage = true // https://github.com/spf13/cobra/issues/340
	licenseClaims, err := keycheck(devportal.PersistentFlags())
	if err == nil {
		err = licenseClaims.RequireFeature(licensekeys.FeatureDevPortal)
	}
	if err == nil {
		log.Printf("License validated")
		return
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	licenseEnforce()
	var diagdURL, ambassadorURL, publicURL, pollEverySecsStr, contentURL string
	var pollEverySecs time.Duration = 60 * time.Second
	var set bool
	diagdURL, set = os.LookupEnv("DIAGD_URL")
	if !set {
		// Typically will be run in same Pod; customizable only in order
		// to support running outside of Kubernetes.
		diagdURL = "http://localhost:8877"
	}
	ambassadorURL, set = os.LookupEnv("AMBASSADOR_URL")
	if !set {
		// Ambassador's Envoy running in the same Pod:
		ambassadorURL = "http://localhost:8080"
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
	contentURL, set = os.LookupEnv("CODE_CONTENT_URL")
	if !set {
		// We need whoever is installing the Dev Portal to supply this,
		// but since it ends up in documentation only it's OK to have a
		// placeholder.
		publicURL = "dev-server-content-root"
	}
	server.Main(Version, diagdURL, ambassadorURL, publicURL, pollEverySecs,
		contentURL)
}
