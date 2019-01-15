package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"time"

	"github.com/datawire/apro/pkg/longpoll"
)

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
	"match": map[string]string{"prefix": "/"},
	"route": map[string]string{"cluster": "app"}}

func processIntercepts(intercepts []InterceptInfo) {
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
		panic(err)
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
		panic(err)
	}
	err = os.Rename("temp/route.json", "data/route.json")
}

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func main() {
	log.SetPrefix("SIDECAR: ")
	log.Printf("Sidecar version %s", Version)

	os.Mkdir("temp", 0775)

	empty := make([]InterceptInfo, 0)
	intercepts := empty

	appName := os.Getenv("APPNAME")
	if appName == "" {
		log.Print("ERROR: APPNAME env var not configured.")
		log.Print("Running without intercept capabilities.")
		processIntercepts(intercepts)
		<-time.After(time.Duration(math.MaxInt64)) // Block forever-ish
		// not reached for a long time
	}

	log.SetPrefix(fmt.Sprintf("%s(%s) ", log.Prefix(), appName))

	u, _ := url.Parse("http://telepresence-proxy:8081/routes")
	c := longpoll.NewClient(u, appName)
	c.Start()

	for {
		processIntercepts(intercepts)
		select {
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

	//c.Stop()

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
