package server

import (
	"github.com/datawire/apro/cmd/dev-portal-server/kubernetes"
	"time"
)

// Add a new/updated service.
type AddServiceFunc func(
	service kubernetes.Service, prefix string, hasDoc bool,
	jsonDoc interface{})

// Delete a service.
type DeleteServiceFunc func(service kubernetes.Service)

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

type fetcher struct {
	add    AddServiceFunc
	delete DeleteServiceFunc
	done   chan bool
	ticker *time.Ticker
	diff   *diffCalculator
}

// Object that retrieves service info and OpenAPI docs (if available) and
// adds/deletes changes from last run.
func NewFetcher(
	add AddServiceFunc, delete DeleteServiceFunc,
	known []kubernetes.Service, duration time.Duration) *fetcher {
	f := &fetcher{
		add:    add,
		delete: delete,
		done:   make(chan bool),
		ticker: time.NewTicker(duration),
		diff:   NewDiffCalculator(known),
	}
	go func() {
		for {
			select {
			case <-f.done:
				return
			case <-f.ticker.C:
				// Retrieve all services:
				f.retrieve()

				// Finished retrieving services, so delete any
				// we don't recognize:
				for _, service := range f.diff.NewRound() {
					f.delete(service)
				}
			}
		}
	}()
	return f
}

func (f *fetcher) retrieve() {
	// XXX logic to talk diagd and get info
}

func (f *fetcher) Stop() {
	f.ticker.Stop()
	close(f.done)
}
