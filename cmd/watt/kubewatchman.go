package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/supervisor"
)

type k8sEvent struct {
	watchId   string
	kind      string
	resources []k8s.Resource
}

type KubernetesWatchMaker struct {
	kubeAPI *k8s.Client
	notify  chan<- k8sEvent
}

func (m *KubernetesWatchMaker) MakeKubernetesWatch(spec KubernetesWatchSpec) (*supervisor.Worker, error) {
	var worker *supervisor.Worker
	var err error

	worker = &supervisor.Worker{
		Name: fmt.Sprintf("kubernetes:%s", spec.WatchId()),
		Work: func(p *supervisor.Process) error {
			watcher := m.kubeAPI.Watcher()
			watchFunc := func(watchId, ns, kind string) func(watcher *k8s.Watcher) {
				return func(watcher *k8s.Watcher) {
					resources := watcher.List(kind)
					p.Logf("found %d %q in namespace %q", len(resources), kind, fmtNamespace(ns))
					m.notify <- k8sEvent{watchId: watchId, kind: kind, resources: resources}
					p.Logf("sent %q to receivers", kind)
				}
			}

			watcherErr := watcher.SelectiveWatch(spec.Namespace, spec.Kind, spec.FieldSelector, spec.LabelSelector,
				watchFunc(spec.WatchId(), spec.Namespace, spec.Kind))

			if watcherErr != nil {
				return watcherErr
			}

			watcher.Start()
			<-p.Shutdown()
			watcher.Stop()
			return nil
		},

		Retry: true,
	}

	return worker, err
}

type kubewatchman struct {
	WatchMaker IKubernetesWatchMaker
	watched    map[string]*supervisor.Worker
	in         <-chan []KubernetesWatchSpec
}

func (w *kubewatchman) Work(p *supervisor.Process) error {
	p.Ready()

	w.watched = make(map[string]*supervisor.Worker)

	for {
		select {
		case watches := <-w.in:
			found := make(map[string]*supervisor.Worker)
			p.Logf("processing %d kubernetes watch specs", len(watches))
			for _, spec := range watches {
				worker, err := w.WatchMaker.MakeKubernetesWatch(spec)
				if err != nil {
					p.Logf("failed to create kubernetes watcher: %v", err)
					continue
				}

				if _, exists := w.watched[worker.Name]; exists {
					found[worker.Name] = w.watched[worker.Name]
				} else {
					p.Logf("add kubernetes watcher %s\n", worker.Name)
					p.Supervisor().Supervise(worker)
					w.watched[worker.Name] = worker
					found[worker.Name] = worker
				}
			}

			for workerName, worker := range w.watched {
				if _, exists := found[workerName]; !exists {
					p.Logf("remove kubernetes watcher %s\n", workerName)
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

type kubebootstrap struct {
	aggregator     *Aggregator
	namespace      string
	kinds          []string
	fieldSelector  string
	labelSelector  string
	notify         []chan<- k8sEvent
	kubeAPIWatcher *k8s.Watcher
}

func fmtNamespace(ns string) string {
	if ns == "" {
		return "*"
	}
	return ns
}

func (b *kubebootstrap) Work(p *supervisor.Process) error {
	pendingResources := map[string]struct{}{}
	var pendingMutex = &sync.Mutex{}

	addWatcher := func(kind string, watcherFunc func(*k8s.Watcher)) error {
		return b.kubeAPIWatcher.SelectiveWatch(b.namespace, kind, b.fieldSelector, b.labelSelector, watcherFunc)
	}

	// try to install watchers for all the resources in our pending list
	tryToWatchAllPending := func(allowRun bool) error {
		// Only allow one round through here.
		// p.Logf("pendingMutex lock")
		pendingMutex.Lock()

		defer func() {
			// p.Logf("pendingMutex unlock")
			pendingMutex.Unlock()
		}()

		if len(pendingResources) == 0 {
			p.Logf("no resource types are pending")
			return nil
		}

		for kind := range pendingResources {
			watcherFunc := func(ns, kind string) func(watcher *k8s.Watcher) {
				return func(watcher *k8s.Watcher) {
					resources := watcher.List(kind)
					p.Logf("found %d %q in namespace %q", len(resources), kind, fmtNamespace(ns))
					for _, n := range b.notify {
						n <- k8sEvent{kind: kind, resources: resources}
					}
					p.Logf("sent %q to %d receivers", kind, len(b.notify))
				}
			}

			if err := addWatcher(kind, watcherFunc(b.namespace, kind)); err != nil {
				b.aggregator.MarkRequired(kind, false)

				if errors.Is(err, k8s.ErrUnkResource) {
					p.Logf("%q does not exist in the cluster at this time: will try later on...", kind)
				} else {
					return err
				}
			} else {
				if allowRun {
					p.Logf("Starting watcher for %q", kind)
					phase, err := b.kubeAPIWatcher.StartWatcherForKind(kind)

					if err != nil {
						p.Logf("error starting watcher for %q: %q, %q", kind, phase, err)
						return err
					}
				}

				b.aggregator.MarkRequired(kind, true)
				// p.Logf("watcher for %q successfully installed", kind)
				delete(pendingResources, kind)
			}
		}
		p.Logf("%d resources are pending", len(pendingResources))
		return nil
	}

	// fill the list of pending resources
	for _, kind := range b.kinds {
		pendingResources[kind] = struct{}{}
	}

	// ... and try to add watchers for all of them.
	//
	// Don't allow immediately starting all the watchers here -- this is bootstrap
	// code, and we want to start everything running later, all at once.
	if err := tryToWatchAllPending(false); err != nil {
		return err
	}

	// some CRDs can be missing from the cluster (like after an upgrade, where users have changed only the image).
	// in those cases we should not fail but install a watcher for CRDs that installs the
	// watcher when the CRD is available.
	if len(pendingResources) > 0 {
		p.Logf("setting up to watch for new CRDs...")

		watcherFunc := func(watcher *k8s.Watcher) {
			p.Logf("new CRD! spawning goroutine to refresh...")

			go func() {
				_ = b.kubeAPIWatcher.Refresh()
				p.Logf("retrying all pending types...")
				_ = tryToWatchAllPending(true)
			}()
		}

		if err := b.kubeAPIWatcher.Watch("customresourcedefinitions.v1beta1.apiextensions.k8s.io", watcherFunc); err != nil {
			return fmt.Errorf("could not watch v1beta1 CRDs: %w", err)
		}
	}

	p.Logf("Watching resources...")
	b.kubeAPIWatcher.StartWithErrorHandler(func(kind string, stage string, err error) {
		p.Logf("could not watch %q at stage %q: %q", kind, stage, err)
	})

	p.Logf("Marking kubewatchman ready")
	p.Ready()

	for range p.Shutdown() {
		p.Logf("shutdown initiated")
		b.kubeAPIWatcher.Stop()
	}

	return nil
}
