package server

import (
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	"github.com/datawire/apro/cmd/dev-portal-server/openapi"
	"time"
)

// Add a new/updated service.
type AddServiceFunc func(
	service kubernetes.Service, prefix string,
	doc *openapi.OpenAPIDoc)

// Delete a service.
type DeleteServiceFunc func(service kubernetes.Service)

type serviceMap map[kubernetes.Service]bool

// Figure out what services no longer exist and need to be deleted.
type deleteCalculator struct {
	previous serviceMap
	current  serviceMap
}

func NewDeleteCalculator(known []kubernetes.Service) *deleteCalculator {
	knownMap := make(serviceMap)
	for _, service := range known {
		knownMap[service] = true
	}
	return &deleteCalculator{previous: make(serviceMap), current: knownMap}
}

// Done retrieving all known services: this will return list of services to
// delete.
func (d *deleteCalculator) NewRound() []kubernetes.Service {
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

type fetcher struct {
	add    AddServiceFunc
	ticker *time.Ticker
	delete *deleteCalculator
}

// Object that retrieves service info and OpenAPI docs (if available) and
// adds/deletes changes from last run.
func NewFetcher(
	add AddServiceFunc, known []kubernetes.Service, duration time.Duration) *fetcher {
	f := &fetcher{
		add:    add,
		ticker: time.NewTicker(duration),
		delete: NewDeleteCalculator(known),
	}
	// XXX run goroutine that retrieves.
	return f
}

func (f *fetcher) Stop() {
	f.ticker.Stop()
}
