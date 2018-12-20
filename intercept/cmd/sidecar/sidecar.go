package main

// Based on https://github.com/jcuga/golongpoll/blob/master/go-client/glpclient/client.go

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"
)

// PollResponse is what the server sends back
type PollResponse struct {
	Events    []PollEvent `json:"events"`
	Timestamp int64       `json:"timestamp"`
}

// PollEvent represent one event in the stream
type PollEvent struct {
	Timestamp int64           `json:"timestamp"` // in ms to match JS Date.getTime()
	Category  string          `json:"category"`
	Data      json.RawMessage `json:"data"` // anything that json.Marshal() accepts
}

// Client represent the long poll client configuration and state
type Client struct {
	url               *url.URL
	category          string
	Timeout           int // polling timeout
	Reattempt         int // time between reconnects after failure
	EventsChan        chan *PollEvent
	runID             uint64 // Something something
	HTTPClient        *http.Client
	BasicAuthUsername string
	BasicAuthPassword string

	// Whether or not logging should be enabled
	LoggingEnabled bool
}

// NewClient creates a long poll client
func NewClient(url *url.URL, category string) *Client {
	return &Client{
		url:            url,
		category:       category,
		Timeout:        120,
		Reattempt:      30,
		EventsChan:     make(chan *PollEvent),
		HTTPClient:     &http.Client{},
		LoggingEnabled: true,
	}
}

// Start the polling of the events on the URL defined in the client
// Will send the events in the EventsChan of the client
func (c *Client) Start() {
	u := c.url
	if c.LoggingEnabled {
		log.Println("Now observing changes on", u.String())
	}

	atomic.AddUint64(&(c.runID), 1)
	currentRunID := atomic.LoadUint64(&(c.runID))

	go func(runID uint64, u *url.URL) {
		since := int64(0)
		for {
			pr, err := c.fetchEvents(since)

			if err != nil {
				if c.LoggingEnabled {
					log.Println(err)
					log.Printf("Reattempting to connect to %s in %d seconds", u.String(), c.Reattempt)
				}
				c.EventsChan <- nil
				time.Sleep(time.Duration(c.Reattempt) * time.Second)
				continue
			}

			// We check that its still the same runID as when this goroutine was started
			clientRunID := atomic.LoadUint64(&(c.runID))
			if clientRunID != runID {
				if c.LoggingEnabled {
					log.Printf("Client on URL %s has been stopped, not sending events", u.String())
				}
				return
			}

			if len(pr.Events) > 0 {
				if c.LoggingEnabled {
					log.Println("Got", len(pr.Events), "event(s) from URL", u.String())
				}
				for _, event := range pr.Events {
					since = event.Timestamp
					c.EventsChan <- &event
				}
			} else {
				// Only push timestamp forward if its greater than the last we checked
				if pr.Timestamp > since {
					since = pr.Timestamp
				}
			}
		}
	}(currentRunID, u)
}

// Stop the client for some reason
func (c *Client) Stop() {
	// Changing the runID will have any previous goroutine ignore any events it may receive
	atomic.AddUint64(&(c.runID), 1)
}

// Call the longpoll server to get the events since a specific timestamp
func (c Client) fetchEvents(since int64) (PollResponse, error) {
	u := c.url
	if c.LoggingEnabled {
		log.Println("Checking for events since", since, "on URL", u.String())
	}

	query := u.Query()
	query.Set("category", c.category)
	query.Set("since_time", fmt.Sprintf("%d", since))
	query.Set("timeout", fmt.Sprintf("%d", c.Timeout))
	u.RawQuery = query.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	if c.BasicAuthUsername != "" && c.BasicAuthPassword != "" {
		req.SetBasicAuth(c.BasicAuthUsername, c.BasicAuthPassword)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		msg := fmt.Sprintf("Error while connecting to %s to observe changes. Error was: %s", u, err)
		return PollResponse{}, errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Wrong status code received from longpoll server: %d", resp.StatusCode)
		return PollResponse{}, errors.New(msg)
	}

	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()

	var pr PollResponse
	err = decoder.Decode(&pr)
	if err != nil {
		if c.LoggingEnabled {
			log.Printf("Error while decoding poll response: %s", err)
		}
		return PollResponse{}, err
	}

	return pr, nil
}

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

	contents := fmt.Sprintf(listenerTemplate, string(routesJSON))
	err = ioutil.WriteFile("temp/listener.json", []byte(contents), 0644)
	if err != nil {
		panic(err)
	}
	err = os.Rename("temp/listener.json", "data/listener.json")
}

func main() {
	log.SetPrefix("SIDECAR: ")

	os.Mkdir("temp", 0775)

	empty := make([]InterceptInfo, 0)
	intercepts := empty

	appName := os.Getenv("APPNAME")
	if appName == "" {
		log.Print("ERROR: APPNAME env var not configured.")
		log.Print("Running without intercept capabilities.")
		processIntercepts(intercepts)
		select {} // Block forever
		// not reached
	}

	log.SetPrefix(fmt.Sprintf("%s(%s) ", log.Prefix(), appName))

	u, _ := url.Parse("http://telepresence-proxy:8081/routes")
	c := NewClient(u, appName)
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

const listenerTemplate = `
{
	"@type": "/envoy.api.v2.Listener",
	"name": "test-listener",
	"address": {
		"socket_address": {
			"address": "0.0.0.0",
			"port_value": 9900
		}
	},
	"filter_chains": [
		{
			"filters": [
				{
					"name": "envoy.http_connection_manager",
					"config": {
						"stat_prefix": "sidecar",
						"http_filters": [
							{
								"name": "envoy.router"
							}
						],
						"route_config": {
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
					}
				}
			]
		}
	]
}
`
