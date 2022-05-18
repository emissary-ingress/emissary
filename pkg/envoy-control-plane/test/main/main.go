// Copyright 2018 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

// Package main contains the test driver for testing xDS manually.
package main

import (
	"context"
	cryptotls "crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	cachev2 "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v2"
	cachev3 "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/cache/v3"
	serverv2 "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/v2"
	serverv3 "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/server/v3"
	"github.com/datawire/ambassador/v2/pkg/envoy-control-plane/test"
	testv2 "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/test/v2"
	testv3 "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/test/v3"

	resourcev2 "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/test/resource/v2"
	resourcev3 "github.com/datawire/ambassador/v2/pkg/envoy-control-plane/test/resource/v3"
)

var (
	debug bool

	port            uint
	gatewayPort     uint
	upstreamPort    uint
	upstreamMessage string
	basePort        uint
	alsPort         uint

	delay    time.Duration
	requests int
	updates  int

	mode          string
	clusters      int
	httpListeners int
	tcpListeners  int
	runtimes      int
	tls           bool
	mux           bool

	nodeID string
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Use debug logging")

	//
	// These parameters control the ports that the integration test
	// components use to talk to one another
	//

	// The port that the Envoy xDS client uses to talk to the control
	// plane xDS server (part of this program)
	flag.UintVar(&port, "port", 18000, "xDS management server port")

	// The port that the Envoy REST client uses to talk to the control
	// plane gateway (which translates from REST to xDS)
	flag.UintVar(&gatewayPort, "gateway", 18001, "Management HTTP gateway (from HTTP to xDS) server port")

	// The port that Envoy uses to talk to the upstream http "echo"
	// server
	flag.UintVar(&upstreamPort, "upstream", 18080, "Upstream HTTP/1.1 port")

	// The port that the tests below use to talk to Envoy's proxy of the
	// upstream server
	flag.UintVar(&basePort, "base", 9000, "Envoy Proxy listener port")

	// The control plane accesslog server port (currently unused)
	flag.UintVar(&alsPort, "als", 18090, "Control plane accesslog server port")

	//
	// These parameters control Envoy configuration
	//

	// Tell Envoy to request configurations from the control plane using
	// this protocol
	flag.StringVar(&mode, "xds", resourcev2.Ads, "Management protocol to test (ADS, xDS, REST)")

	// Tell Envoy to use this Node ID
	flag.StringVar(&nodeID, "nodeID", "test-id", "Node ID")

	// Tell Envoy to use TLS to talk to the control plane
	flag.BoolVar(&tls, "tls", false, "Enable TLS on all listeners and use SDS for secret delivery")

	// Tell Envoy to configure this many clusters for each snapshot
	flag.IntVar(&clusters, "clusters", 4, "Number of clusters")

	// Tell Envoy to configure this many Runtime Discovery Service
	// layers for each snapshot
	flag.IntVar(&runtimes, "runtimes", 1, "Number of RTDS layers")

	//
	// These parameters control the test harness
	//

	// The message that the tests expect to receive from the upstream
	// server
	flag.StringVar(&upstreamMessage, "message", "Default message", "Upstream HTTP server response message")

	// Time to wait between test request batches
	flag.DurationVar(&delay, "delay", 500*time.Millisecond, "Interval between request batch retries")

	// Each test loads a configuration snapshot into the control plane
	// which is then picked up by Envoy.  This parameter specifies how
	// many snapshots to test
	flag.IntVar(&updates, "u", 3, "Number of snapshot updates")

	// Each snapshot test sends this many requests to the upstream
	// server for each snapshot for each listener port
	flag.IntVar(&requests, "r", 5, "Number of requests between snapshot updates")

	// Test this many HTTP listeners per snapshot
	flag.IntVar(&httpListeners, "http", 2, "Number of HTTP listeners (and RDS configs)")
	// Test this many TCP listeners per snapshot
	flag.IntVar(&tcpListeners, "tcp", 2, "Number of TCP pass-through listeners")

	// Enable a muxed cache with partial snapshots
	flag.BoolVar(&mux, "mux", false, "Enable muxed linear cache for EDS")
}

// main returns code 1 if any of the batches failed to pass all requests
func main() {
	flag.Parse()
	ctx := context.Background()

	// create a cache
	signal := make(chan struct{})
	cbv2 := &testv2.Callbacks{Signal: signal, Debug: debug}
	cbv3 := &testv3.Callbacks{Signal: signal, Debug: debug}

	configv2 := cachev2.NewSnapshotCache(mode == resourcev2.Ads, cachev2.IDHash{}, logger{})
	configv3 := cachev3.NewSnapshotCache(mode == resourcev2.Ads, cachev3.IDHash{}, logger{})
	srv2 := serverv2.NewServer(context.Background(), configv2, cbv2)

	// mux integration
	var configCachev3 cachev3.Cache = configv3
	typeURL := "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment"
	eds := cachev3.NewLinearCache(typeURL)
	if mux {
		configCachev3 = &cachev3.MuxCache{
			Classify: func(req cachev3.Request) string {
				if req.TypeUrl == typeURL {
					return "eds"
				}
				return "default"
			},
			Caches: map[string]cachev3.Cache{
				"default": configv3,
				"eds":     eds,
			},
		}
	}
	srv3 := serverv3.NewServer(context.Background(), configCachev3, cbv3)
	alsv2 := &testv2.AccessLogService{}
	alsv3 := &testv3.AccessLogService{}

	// create a test snapshot
	snapshotsv2 := resourcev2.TestSnapshot{
		Xds:              mode,
		UpstreamPort:     uint32(upstreamPort),
		BasePort:         uint32(basePort),
		NumClusters:      clusters,
		NumHTTPListeners: httpListeners,
		NumTCPListeners:  tcpListeners,
		TLS:              tls,
		NumRuntimes:      runtimes,
	}
	snapshotsv3 := resourcev3.TestSnapshot{
		Xds:              mode,
		UpstreamPort:     uint32(upstreamPort),
		BasePort:         uint32(basePort),
		NumClusters:      clusters,
		NumHTTPListeners: httpListeners,
		NumTCPListeners:  tcpListeners,
		TLS:              tls,
		NumRuntimes:      runtimes,
	}

	// start the xDS server
	go test.RunAccessLogServer(ctx, alsv2, alsv3, alsPort)
	go test.RunManagementServer(ctx, srv2, srv3, port)
	go test.RunManagementGateway(ctx, srv2, srv3, gatewayPort, logger{})

	log.Println("waiting for the first request...")
	select {
	case <-signal:
		break
	case <-time.After(1 * time.Minute):
		log.Println("timeout waiting for the first request")
		os.Exit(1)
	}
	log.Printf("initial snapshot %+v\n", snapshotsv2)
	log.Printf("executing sequence updates=%d request=%d\n", updates, requests)

	for i := 0; i < updates; i++ {
		snapshotsv2.Version = fmt.Sprintf("v%d", i)
		log.Printf("update snapshot %v\n", snapshotsv2.Version)
		snapshotsv3.Version = fmt.Sprintf("v%d", i)
		log.Printf("update snapshot %v\n", snapshotsv3.Version)

		snapshotv2 := snapshotsv2.Generate()
		snapshotv3 := snapshotsv3.Generate()
		if err := snapshotv2.Consistent(); err != nil {
			log.Printf("snapshot inconsistency: %+v\n", snapshotv2)
		}
		if err := snapshotv3.Consistent(); err != nil {
			log.Printf("snapshot inconsistency: %+v\n", snapshotv3)
		}

		err := configv2.SetSnapshot(nodeID, snapshotv2)
		if err != nil {
			log.Printf("snapshot error %q for %+v\n", err, snapshotv2)
			os.Exit(1)
		}

		err = configv3.SetSnapshot(nodeID, snapshotv3)
		if err != nil {
			log.Printf("snapshot error %q for %+v\n", err, snapshotv3)
			os.Exit(1)
		}
		if mux {
			for name, res := range snapshotv3.GetResources(typeURL) {
				eds.UpdateResource(name, res)
			}
		}

		// pass is true if all requests succeed at least once in a run
		pass := false
		for j := 0; j < requests; j++ {
			ok, failed := callEcho()
			if failed == 0 && !pass {
				pass = true
			}
			log.Printf("request batch %d, ok %v, failed %v, pass %v\n", j, ok, failed, pass)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		alsv2.Dump(func(s string) {
			if debug {
				log.Println(s)
			}
		})
		cbv2.Report()

		alsv3.Dump(func(s string) {
			if debug {
				log.Println(s)
			}
		})
		cbv3.Report()

		if !pass {
			log.Printf("failed all requests in a run %d\n", i)
			os.Exit(1)
		}
	}

	log.Printf("Test for %s passed!\n", mode)
}

// callEcho calls upstream echo service on all listener ports and returns an error
// if any of the listeners returned an error.
func callEcho() (int, int) {
	total := httpListeners + tcpListeners
	ok, failed := 0, 0
	ch := make(chan error, total)

	// spawn requests
	for i := 0; i < total; i++ {
		go func(i int) {
			client := http.Client{
				Timeout: 100 * time.Millisecond,
				Transport: &http.Transport{
					TLSClientConfig: &cryptotls.Config{InsecureSkipVerify: true},
				},
			}
			proto := "http"
			if tls {
				proto = "https"
			}
			req, err := client.Get(fmt.Sprintf("%s://127.0.0.1:%d", proto, basePort+uint(i)))
			if err != nil {
				ch <- err
				return
			}
			defer req.Body.Close()
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				ch <- err
				return
			}
			if string(body) != upstreamMessage {
				ch <- fmt.Errorf("unexpected return %q", string(body))
				return
			}
			ch <- nil
		}(i)
	}

	for {
		out := <-ch
		if out == nil {
			ok++
		} else {
			failed++
		}
		if ok+failed == total {
			return ok, failed
		}
	}
}

type logger struct{}

func (logger logger) Debugf(format string, args ...interface{}) {
	if debug {
		log.Printf(format+"\n", args...)
	}
}

func (logger logger) Infof(format string, args ...interface{}) {
	if debug {
		log.Printf(format+"\n", args...)
	}
}

func (logger logger) Warnf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

func (logger logger) Errorf(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}
