package main

import (
	"fmt"
	"log"
	"os"
	"time"

	consulapi "github.com/hashicorp/consul/api"

	"github.com/datawire/ambassador/pkg/consulwatch"
	"github.com/datawire/ambassador/pkg/supervisor"
	"github.com/datawire/ambassador/pkg/watt"
)

const (
	// distLockKey is the key for the distributed lock
	distLockKey = "AMB_CONSUL_CONNECT_LEADER"

	// distLockTTL is the time-to-live for the lock session
	distLockTTL = 15 * time.Second
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

type consulEvent struct {
	WatchId   string
	Endpoints consulwatch.Endpoints
}

type consulwatchman struct {
	WatchMaker IConsulWatchMaker
	watchesCh  <-chan []ConsulWatchSpec
	watched    map[string]*supervisor.Worker
}

type ConsulWatchMaker struct {
	aggregatorCh chan<- consulEvent
}

// MakeConsulWatch watches Consul and sends events to the aggregator channel
func (m *ConsulWatchMaker) MakeConsulWatch(spec ConsulWatchSpec) (*supervisor.Worker, error) {
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = spec.ConsulAddress
	consulConfig.Datacenter = spec.Datacenter

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
			eventsWatcher, err := consulwatch.New(consul, logger, spec.Datacenter, spec.ServiceName, true)
			if err != nil {
				p.Logf("failed to setup new consul watch %v", err)
				return err
			}
			defer func() {
				eventsWatcher.Stop()
			}()

			eventsWatcher.Watch(func(endpoints consulwatch.Endpoints, e error) {
				endpoints.Id = spec.Id
				m.aggregatorCh <- consulEvent{spec.WatchId(), endpoints}
			})
			_ = p.Go(func(p *supervisor.Process) error {
				if err := eventsWatcher.Start(); err != nil {
					p.Logf("failed to start service watcher %v", err)
					return err
				}
				return nil
			})

			p.Logf("Creating distributed lock for Consul watchers.")
			distLock, err := watt.NewDistLock(consul, distLockKey, distLockTTL)
			if err != nil {
				p.Logf("failed to setup distributed lock for Consul %v", err)
				return err
			}
			defer func() {
				p.Log("Releasing distributed lock...")
				if err = distLock.DestroySession(); err != nil {
					p.Logf("failed to release lock %v", err)
				}
			}()

			var cc *consulwatch.ConnectWatcher
			acquireCh := make(chan bool)
			releaseCh := make(chan bool)
			for {
				// loop is to re-attempt for lock acquisition when
				// the lock was initially acquired but auto released after some time
				go distLock.RetryLockAcquire(acquireCh, releaseCh)

				p.Logf("Waiting to acquire Consul lock...")
				select {
				case <-acquireCh:
					p.Logf("Acquired Consul lock: we are the leaders watching Consul certificates")
					cc = consulwatch.NewConnectWatcher(p, consul)
					if err := cc.Watch(); err != nil {
						return err
					}

				case <-p.Shutdown():
					p.Logf("Supervisor is shutting down...")
					if cc != nil {
						cc.Close()
						cc = nil
					}
					return nil // we are done in the Worker: get out...
				}

				<-releaseCh
				p.Logf("Lost Consul lock: releasing watches and resources")
				if cc != nil {
					cc.Close()
					cc = nil
				}
				// we will iterate and try to acquire the lock again...
			}
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
			found := make(map[string]*supervisor.Worker)
			p.Logf("processing %d consul watches", len(watches))
			for _, cw := range watches {
				worker, err := w.WatchMaker.MakeConsulWatch(cw)
				if err != nil {
					p.Logf("failed to create consul watch %v", err)
					continue
				}

				if _, exists := w.watched[worker.Name]; exists {
					found[worker.Name] = w.watched[worker.Name]
				} else {
					p.Logf("add consul watcher %s\n", worker.Name)
					p.Supervisor().Supervise(worker)
					w.watched[worker.Name] = worker
					found[worker.Name] = worker
				}
			}

			// purge the watches that no longer are needed because they did not come through the in the latest
			// report
			for workerName, worker := range w.watched {
				if _, exists := found[workerName]; !exists {
					p.Logf("remove consul watcher %s\n", workerName)
					worker.Shutdown()
					worker.Wait()
				}
			}

			w.watched = found
		case <-p.Shutdown():
			p.Logf("shutdown initiated")
			return nil
		}
	}
}
