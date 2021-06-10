package thingconsul

import (
	"fmt"
	"sync"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/datawire/ambassador/v2/cmd/watt/watchapi"
	"github.com/datawire/ambassador/v2/pkg/consulwatch"
	"github.com/datawire/ambassador/v2/pkg/supervisor"
)

type ConsulEvent struct {
	WatchId   string
	Endpoints consulwatch.Endpoints
}

type consulwatchman struct {
	WatchMaker watchapi.IConsulWatchMaker
	watchesCh  <-chan []watchapi.ConsulWatchSpec

	mu      sync.RWMutex
	watched map[string]*supervisor.Worker
}

type ConsulWatchMan interface {
	Work(*supervisor.Process) error
	NumWatched() int
	WithWatched(func(map[string]*supervisor.Worker))
}

func NewConsulWatchMan(eventsCh chan<- ConsulEvent, watchesCh <-chan []watchapi.ConsulWatchSpec) ConsulWatchMan {
	return &consulwatchman{
		WatchMaker: &ConsulWatchMaker{aggregatorCh: eventsCh},
		watchesCh:  watchesCh,
		watched:    make(map[string]*supervisor.Worker),
	}
}

type ConsulWatchMaker struct {
	aggregatorCh chan<- ConsulEvent
}

func (m *ConsulWatchMaker) MakeConsulWatch(spec watchapi.ConsulWatchSpec) (*supervisor.Worker, error) {
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = spec.ConsulAddress

	// TODO: Should we really allocated a Consul client per Service watch? Not sure... there some design stuff here
	// May be multiple consul clusters
	// May be different connection parameters on the consulConfig
	// Seems excessive...
	consul, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}

	worker := &supervisor.Worker{
		Name: fmt.Sprintf("consul:%s", spec.WatchId()),
		Work: func(p *supervisor.Process) error {
			w, err := consulwatch.New(consul, spec.Datacenter, spec.ServiceName, true)
			if err != nil {
				p.Logf("failed to setup new consul watch %v", err)
				return err
			}

			w.Watch(func(endpoints consulwatch.Endpoints, e error) {
				endpoints.Id = spec.Id
				m.aggregatorCh <- ConsulEvent{spec.WatchId(), endpoints}
			})
			_ = p.Go(func(p *supervisor.Process) error {
				x := w.Start(p.Context())
				if x != nil {
					p.Logf("failed to start service watcher %v", x)
					return x
				}

				return nil
			})

			<-p.Shutdown()
			w.Stop()
			return nil
		},
		Retry: true,
	}

	return worker, nil
}

func (w *consulwatchman) Work(p *supervisor.Process) error {
	p.Ready()
	for {
		select {
		case watches := <-w.watchesCh:
			w.mu.Lock()
			found := make(map[string]*supervisor.Worker)
			p.Debugf("processing %d consul watches", len(watches))
			for _, cw := range watches {
				worker, err := w.WatchMaker.MakeConsulWatch(cw)
				if err != nil {
					p.Logf("failed to create consul watch %v", err)
					continue
				}

				if _, exists := w.watched[worker.Name]; exists {
					found[worker.Name] = w.watched[worker.Name]
				} else {
					p.Debugf("add consul watcher %s\n", worker.Name)
					p.Supervisor().Supervise(worker)
					w.watched[worker.Name] = worker
					found[worker.Name] = worker
				}
			}

			// purge the watches that no longer are needed because they did not come through the in the latest
			// report
			for workerName, worker := range w.watched {
				if _, exists := found[workerName]; !exists {
					p.Debugf("remove consul watcher %s\n", workerName)
					worker.Shutdown()
					worker.Wait()
				}
			}

			w.watched = found
			w.mu.Unlock()
		case <-p.Shutdown():
			p.Debugf("shutdown initiated")
			return nil
		}
	}
}

func (w *consulwatchman) NumWatched() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.watched)
}

func (w *consulwatchman) WithWatched(fn func(map[string]*supervisor.Worker)) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	fn(w.watched)
}
