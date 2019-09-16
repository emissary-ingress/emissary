package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/cmd/amb-sidecar/devportal/server"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/licensekeys"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func licenseEnforce() {
	devportal := &cobra.Command{
		Use: "local-devportal [command]",
	}
	keycheck := licensekeys.InitializeCommandFlags(devportal.PersistentFlags(), "local-devportal", Version)
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

func getenvDefault(varname, def string) string {
	ret := os.Getenv(varname)
	if ret == "" {
		ret = def
	}
	return ret
}

func main() {
	licenseEnforce()

	// Typically will be run in same Pod; customizable only in order
	// to support running outside of Kubernetes.
	ambassadorAdminURLStr := getenvDefault("AMBASSADOR_ADMIN_URL", "http://localhost:8877")
	ambassadorAdminURL, err := url.Parse(ambassadorAdminURLStr)
	if err != nil {
		log.Fatal(err)
	}

	// Ambassador's Envoy running in the same Pod:
	ambassadorInternalURLStr := getenvDefault("AMBASSADOR_INTERNAL_URL", "http://localhost:8080")
	ambassadorInternalURL, err := url.Parse(ambassadorInternalURLStr)
	if err != nil {
		log.Fatal(err)
	}

	// We need whoever is installing the Dev Portal to supply this,
	// but since it ends up in documentation only it's OK to have a
	// placeholder.
	ambassadorExternalURLStr := getenvDefault("AMBASSADOR_URL", "https://api.example.com")
	ambassadorExternalURL, err := url.Parse(ambassadorExternalURLStr)
	if err != nil {
		log.Fatal(err)
	}

	pollEverySecsStr := getenvDefault("POLL_EVERY_SECS", "60")
	pollEverySecs, err := strconv.Atoi(pollEverySecsStr)
	if err != nil {
		log.Fatal(err)
	}
	pollInterval := time.Duration(pollEverySecs) * time.Second

	// We need whoever is installing the Dev Portal to supply this,
	// but since it ends up in documentation only it's OK to have a
	// placeholder.
	contentURLStr := getenvDefault("CODE_CONTENT_URL", "dev-server-content-root")
	contentURL, err := url.Parse(contentURLStr)
	if err != nil {
		log.Fatal(err)
	}

	config := types.Config{
		AmbassadorAdminURL:    ambassadorAdminURL,
		AmbassadorInternalURL: ambassadorInternalURL,
		AmbassadorExternalURL: ambassadorExternalURL,
		DevPortalPollInterval: pollInterval,
		DevPortalContentURL:   contentURL,
	}

	s, err := server.MakeServer("", context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe("0.0.0.0:8680", s.Router()))
}
