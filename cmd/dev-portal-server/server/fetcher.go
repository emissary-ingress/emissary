package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
)

// Add a new/updated service.
type AddServiceFunc func(
	service kubernetes.Service, baseURL string, prefix string,
	openAPIDoc []byte)

// Delete a service.
type DeleteServiceFunc func(service kubernetes.Service)

// Retrieve a URL.
type HTTPGetFunc func(url string) ([]byte, error)

type serviceMap map[kubernetes.Service]bool

// Figure out what services no longer exist and need to be deleted.
type diffCalculator struct {
	previous serviceMap
	current  serviceMap
}

func NewDiffCalculator(known []kubernetes.Service) *diffCalculator {
	knownMap := make(serviceMap)
	for _, service := range known {
		knownMap[service] = true
	}
	return &diffCalculator{current: make(serviceMap), previous: knownMap}
}

// Done retrieving all known services: this will return list of services to
// delete.
func (d *diffCalculator) NewRound() []kubernetes.Service {
	toDelete := make([]kubernetes.Service, 0)
	for service := range d.previous {
		if !d.current[service] {
			toDelete = append(toDelete, service)
		}
	}
	d.previous = d.current
	d.current = make(serviceMap)
	return toDelete
}

// Add a Service that was successfully retrieved this round
func (d *diffCalculator) Add(s kubernetes.Service) {
	d.current[s] = true
}

type fetcher struct {
	add     AddServiceFunc
	delete  DeleteServiceFunc
	httpGet HTTPGetFunc
	done    chan bool
	ticker  *time.Ticker
	diff    *diffCalculator
	// diagd's URL
	diagURL string
	// ambassador's URL
	ambassadorURL string
	// The public default base URL for the APIs, e.g. https://api.example.com
	publicBaseURL string
}

// Object that retrieves service info and OpenAPI docs (if available) and
// adds/deletes changes from last run.
func NewFetcher(
	add AddServiceFunc, delete DeleteServiceFunc, httpGet HTTPGetFunc,
	known []kubernetes.Service, diagURL string, ambassadorURL string, duration time.Duration, publicBaseURL string) *fetcher {
	f := &fetcher{
		add:           add,
		delete:        delete,
		httpGet:       httpGet,
		done:          make(chan bool),
		ticker:        time.NewTicker(duration),
		diff:          NewDiffCalculator(known),
		diagURL:       strings.TrimRight(diagURL, "/"),
		ambassadorURL: strings.TrimRight(ambassadorURL, "/"),
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}
	go func() {
		for {
			select {
			case <-f.done:
				return
			case <-f.ticker.C:
				// Retrieve all services:
				f.retrieve()
			}
		}
	}()
	return f
}

// Get a string attribute of a JSON object:
func getString(o *gabs.Container, attr string) string {
	return o.S(attr).Data().(string)
}

func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: time.Second * 2}
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d from %s", response.StatusCode, url)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (f *fetcher) retrieve() {
	buf, err := f.httpGet(f.diagURL + "/ambassador/v0/diag/?json=true")
	if err != nil {
		log.Print(err)
		return
	}
	// Don't bother looking at error; failed queries will result in service
	// being removed from Dev Portal.

	json, err := gabs.ParseJSON(buf)
	if err != nil {
		log.Print(err)
		return
	}
	children, err := json.S("groups").ChildrenMap()
	if err != nil {
		log.Print(err)
		return
	}
	for _, child := range children {
		// We don't consider inactive services:
		if !child.S("_active").Data().(bool) {
			continue
		}
		// We don't consider non-HTTP services:
		if getString(child, "kind") != "IRHTTPMappingGroup" {
			continue
		}
		mappings, err := child.S("mappings").Children()
		if err != nil {
			log.Print(err)
			return
		}
		for _, mapping := range mappings {
			if getString(mapping, "location") == "--internal--" {
				continue
			}
			location_parts := strings.Split(getString(mapping, "location"), ".")
			prefix := getString(mapping, "prefix")
			prefix = strings.TrimRight(prefix, "/")
			name := location_parts[0]
			namespace := location_parts[1]
			var baseURL string
			if mapping.Exists("host") {
				// TODO what if it's http? (arguably it should never be)
				baseURL = "https://" + getString(mapping, "host")
			} else {
				baseURL = f.publicBaseURL
			}
			// Get the OpenAPI documentation:
			var doc []byte
			docBuf, err := f.httpGet(f.ambassadorURL + prefix + "/.well-known/openapi-docs")
			if err == nil {
				doc = docBuf
			} else {
				doc = nil
			}
			service := kubernetes.Service{Namespace: namespace, Name: name}
			f.add(service, baseURL, prefix, doc)
			f.diff.Add(service)
		}
	}

	// Finished retrieving services, so delete any we don't recognize:
	for _, service := range f.diff.NewRound() {
		f.delete(service)
	}
}

func (f *fetcher) Stop() {
	f.ticker.Stop()
	close(f.done)
}
