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
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/cache/v3"
	conf "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/config"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/sotw/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/server/v3"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/test"
	"github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/test/resource/v3"
	testv3 "github.com/emissary-ingress/emissary/v3/pkg/envoy-control-plane/test/v3"
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

	mode                string
	clusters            int
	httpListeners       int
	scopedHTTPListeners int
	vhdsHTTPListeners   int
	tcpListeners        int
	runtimes            int
	tls                 bool
	mux                 bool
	extensionNum        int

	nodeID string

	pprofEnabled bool
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

	// The control plane accesslog server port
	flag.UintVar(&alsPort, "als", 18090, "Control plane accesslog server port")

	//
	// These parameters control Envoy configuration
	//

	// Tell Envoy to request configurations from the control plane using
	// this protocol
	flag.StringVar(&mode, "xds", resource.Ads, "Management protocol to test (ADS, xDS, REST, DELTA, DELTA-ADS)")

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
	// Test this many scoped HTTP listeners per snapshot
	flag.IntVar(&scopedHTTPListeners, "scopedhttp", 2, "Number of HTTP listeners (and SRDS configs)")
	// Test this many VHDS HTTP listeners per snapshot
	flag.IntVar(&vhdsHTTPListeners, "vhdshttp", 2, "Number of VHDS HTTP listeners")
	// Test this many TCP listeners per snapshot
	flag.IntVar(&tcpListeners, "tcp", 2, "Number of TCP pass-through listeners")

	// Enable a muxed cache with partial snapshots
	flag.BoolVar(&mux, "mux", false, "Enable muxed linear cache for EDS")

	// Number of ExtensionConfig
	flag.IntVar(&extensionNum, "extension", 1, "Number of Extension")
	//
	// These parameters control the the use of the pprof profiler
	//

	// Enable use of the pprof profiler
	flag.BoolVar(&pprofEnabled, "pprof", false, "Enable use of the pprof profiler")
}

// main returns code 1 if any of the batches failed to pass all requests
func main() {
	flag.Parse()
	ctx := context.Background()

	if pprofEnabled {
		runtime.SetBlockProfileRate(1)
		for _, prof := range []string{"block", "goroutine", "mutex"} {
			log.Printf("turn on pprof %s profiler", prof)
			if pprof.Lookup(prof) == nil {
				pprof.NewProfile(prof)
			}
		}
	}

	// create a cache
	signal := make(chan struct{})
	cb := &testv3.Callbacks{Signal: signal, Debug: debug}

	// mux integration
	// nil for logger uses default logger
	config := cache.NewSnapshotCache(mode == resource.Ads, cache.IDHash{}, nil)
	var configCache cache.Cache = config
	typeURL := "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment"
	eds := cache.NewLinearCache(typeURL)
	if mux {
		configCache = &cache.MuxCache{
			Classify: func(req *cache.Request) string {
				if req.GetTypeUrl() == typeURL {
					return "eds"
				}
				return "default"
			},
			Caches: map[string]cache.Cache{
				"default": config,
				"eds":     eds,
			},
		}
	}

	opts := []conf.XDSOption{}
	if mode == resource.Ads {
		log.Println("enabling ordered ADS mode...")
		// Enable resource ordering if we enter ADS mode.
		opts = append(opts, sotw.WithOrderedADS())
	}
	srv := server.NewServer(context.Background(), configCache, cb, opts...)
	als := &testv3.AccessLogService{}

	if mode != resource.Delta {
		vhdsHTTPListeners = 0
	}

	// create a test snapshot
	snapshots := resource.TestSnapshot{
		Xds:                    mode,
		UpstreamPort:           uint32(upstreamPort),
		BasePort:               uint32(basePort),
		NumClusters:            clusters,
		NumHTTPListeners:       httpListeners,
		NumScopedHTTPListeners: scopedHTTPListeners,
		NumVHDSHTTPListeners:   vhdsHTTPListeners,
		NumTCPListeners:        tcpListeners,
		TLS:                    tls,
		NumRuntimes:            runtimes,
		NumExtension:           extensionNum,
	}

	// start the xDS server
	go test.RunAccessLogServer(ctx, als, alsPort)
	go test.RunManagementServer(ctx, srv, port)
	go test.RunManagementGateway(ctx, srv, gatewayPort)

	log.Println("waiting for the first request...")
	select {
	case <-signal:
		break
	case <-time.After(1 * time.Minute):
		log.Println("timeout waiting for the first request")
		os.Exit(1)
	}
	log.Printf("initial snapshot %+v\n", snapshots)
	log.Printf("executing sequence updates=%d request=%d\n", updates, requests)

	for i := 0; i < updates; i++ {
		snapshots.Version = fmt.Sprintf("v%d", i)
		log.Printf("update snapshot %v\n", snapshots.Version)

		snapshot := snapshots.Generate()
		if err := snapshot.Consistent(); err != nil {
			log.Printf("snapshot inconsistency: %+v\n%+v\n", snapshot, err)
		}

		err := config.SetSnapshot(context.Background(), nodeID, snapshot)
		if err != nil {
			log.Printf("snapshot error %q for %+v\n", err, snapshot)
			os.Exit(1)
		}

		if mux {
			for name, res := range snapshot.GetResources(typeURL) {
				if err := eds.UpdateResource(name, res); err != nil {
					log.Printf("update error %q for %+v\n", err, name)
					os.Exit(1)
				}
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

		als.Dump(func(s string) {
			if debug {
				log.Println(s)
			}
		})
		cb.Report()

		if !pass {
			log.Printf("failed all requests in a run %d\n", i)
			os.Exit(1)
		}
	}

	if pprofEnabled {
		for _, prof := range []string{"block", "goroutine", "mutex"} {
			p := pprof.Lookup(prof)
			filePath := fmt.Sprintf("%s_profile_%s.pb.gz", prof, mode)
			log.Printf("storing %s profile for %s in %s", prof, mode, filePath)
			f, err := os.Create(filePath)
			if err != nil {
				log.Fatalf("could not create %s profile %s: %s", prof, filePath, err)
			}
			p.WriteTo(f, 1) // nolint:errcheck
			f.Close()
		}
	}

	log.Printf("Test for %s passed!\n", mode)
}

// callEcho calls upstream echo service on all listener ports and returns an error
// if any of the listeners returned an error.
func callEcho() (int, int) {
	total := httpListeners + scopedHTTPListeners + tcpListeners + vhdsHTTPListeners
	ok, failed := 0, 0
	ch := make(chan error, total)

	client := http.Client{
		Timeout: 100 * time.Millisecond,
		Transport: &http.Transport{
			TLSClientConfig: &cryptotls.Config{InsecureSkipVerify: true}, // nolint:gosec
		},
	}

	get := func(count int) (*http.Response, error) {
		proto := "http"
		if tls {
			proto = "https"
		}

		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			fmt.Sprintf("%s://127.0.0.1:%d", proto, basePort+uint(count)),
			nil,
		)
		if err != nil {
			return nil, err
		}
		return client.Do(req)
	}

	// spawn requests
	for i := 0; i < total; i++ {
		go func(i int) {
			resp, err := get(i)
			if err != nil {
				ch <- err
				return
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				ch <- err
				return
			}
			if err := resp.Body.Close(); err != nil {
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
