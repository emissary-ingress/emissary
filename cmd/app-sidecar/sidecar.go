package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_log "log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/datawire/apro/cmd/app-sidecar/longpoll"
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/datawire/apro/lib/metriton"
)

var log = _log.New(os.Stderr, "", _log.LstdFlags)

// PatternInfo represents one Envoy header regex_match
type PatternInfo struct {
	Name       string `json:"name"`
	RegexMatch string `json:"regex_match"`
}

// InterceptInfo tracks one intercept operation
type InterceptInfo struct {
	Name     string
	Patterns []PatternInfo
	Port     int
}

type envoyMatch struct {
	Prefix  string        `json:"prefix"`
	Headers []PatternInfo `json:"headers"`
}

type envoyRoute struct {
	Cluster string `json:"cluster"`
}

type envoyRouteBlob struct {
	Match envoyMatch `json:"match"`
	Route envoyRoute `json:"route"`
}

var defaultRoute = map[string]map[string]string{
	"match": {"prefix": "/"},
	"route": {"cluster": "app"},
}

func processIntercepts(intercepts []InterceptInfo) error {
	routes := make([]interface{}, 0, len(intercepts)+1)
	for idx, intercept := range intercepts {
		log.Printf("%2d Sending to proxy:%d (%s) when:",
			idx+1, intercept.Port, intercept.Name)
		for _, pattern := range intercept.Patterns {
			log.Printf("   %s: %s", pattern.Name, pattern.RegexMatch)
		}
		route := envoyRoute{Cluster: fmt.Sprintf("tel-proxy-%d", intercept.Port)}
		match := envoyMatch{Prefix: "/", Headers: intercept.Patterns}
		blob := envoyRouteBlob{Match: match, Route: route}
		routes = append(routes, blob)
	}
	routes = append(routes, defaultRoute)
	routesJSON, err := json.Marshal(routes)
	if err != nil {
		return err
	}

	if len(intercepts) == 0 {
		log.Print("No intercepts in play.")
	}
	log.Print("Computed routes blob is")
	log.Print(string(routesJSON))
	log.Print("---")

	contents := fmt.Sprintf(routeTemplate, string(routesJSON))
	err = ioutil.WriteFile("temp/route.json", []byte(contents), 0644)
	if err != nil {
		return err
	}
	err = os.Rename("temp/route.json", "data/route.json")
	if err != nil {
		return err
	}
	return nil
}

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func main() {
	log.SetPrefix("SIDECAR: ")
	log.Printf("Sidecar version %s", Version)

	argparser := &cobra.Command{
		Use:     os.Args[0],
		Version: Version,
		RunE:    Main,
	}

	licenseContext := &licensekeys.LicenseContext{}
	if err := licenseContext.AddFlagsTo(argparser); err != nil {
		panic(err)
	}

	argparser.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		licenseClaims, err := licenseContext.GetClaims()
		if err == nil {
			err = licenseClaims.RequireFeature(licensekeys.FeatureTraffic)
		}
		if err == nil {
			go metriton.PhoneHome(licenseClaims, nil, "application-sidecar", Version)
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err := argparser.Execute()
	if err != nil {
		os.Exit(2)
	}
}

func Main(flags *cobra.Command, args []string) error {
	os.Mkdir("temp", 0775)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	wg, ctx := errgroup.WithContext(context.Background())
	wg.Go(func() error { return signalHandler(ctx, sigs) })
	wg.Go(func() error { return sidecar(ctx) })
	return wg.Wait()
}

func signalHandler(ctx context.Context, sigs <-chan os.Signal) error {
	defer func() {
		go func() {
			// keep logging signals
			for sig := range sigs {
				log.Printf("received signal %v", sig)
			}
		}()
	}()

	select {
	case sig := <-sigs:
		return errors.Errorf("received signal %v", sig)
	case <-ctx.Done():
		return nil
	}
}

func sidecar(ctx context.Context) error {
	empty := make([]InterceptInfo, 0)
	intercepts := empty

	appName := os.Getenv("APPNAME")
	if appName == "" {
		log.Print("ERROR: APPNAME env var not configured.")
		log.Print("Running without intercept capabilities.")
		err := processIntercepts(intercepts)
		if err != nil {
			return err
		}
		<-ctx.Done() // Block forever (or until shutdown)
		return nil
	}

	log.SetPrefix(fmt.Sprintf("%s(%s) ", log.Prefix(), appName))

	u, _ := url.Parse("http://telepresence-proxy:8081/routes")
	c := longpoll.NewClient(u, appName)
	c.Logger = log
	c.Start()
	defer c.Stop()

	for {
		err := processIntercepts(intercepts)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case e := <-c.EventsChan:
			if e == nil {
				log.Print("No connection to the proxy")
				intercepts = empty
			} else {
				err := json.Unmarshal(e.Data, &intercepts)
				if err != nil {
					log.Println("Failed to unmarshal event", string(e.Data))
					log.Println("Because", err)
					intercepts = empty
				}
			}
		}
	}
}

const routeTemplate = `
{
	"@type": "/envoy.api.v2.RouteConfiguration",
	"name": "application_route",
	"virtual_hosts": [
		{
			"name": "all-the-hosts",
			"domains": [
				"*"
			],
			"routes": %s
		}
	]
}
`
