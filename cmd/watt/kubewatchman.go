package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/supervisor"
	"github.com/datawire/ambassador/pkg/watt"
)

type k8sEvent struct {
	watchId   string
	kind      string
	resources []k8s.Resource
	errors    []watt.Error
}

// makeErrorEvent returns a k8sEvent that contains one error entry for each
// message passed in, all attributed to the same source.
func makeErrorEvent(source string, messages ...string) k8sEvent {
	errors := make([]watt.Error, len(messages))
	for idx, message := range messages {
		errors[idx] = watt.NewError(source, message)
	}
	return k8sEvent{errors: errors}
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
	namespace      string
	kinds          []string
	fieldSelector  string
	labelSelector  string
	notify         []chan<- k8sEvent
	kubeAPIWatcher *k8s.Watcher

	// markRequired is a function that we'll use to indicate whether a given kind of
	// resource is required (or not) for a successful bootstrap.
	markRequired func(key string, required bool)

	// pendingResources is the set of things that we would like to watch for.
	// It will vary over time -- it starts as the full set that we want to
	// work with, and then we remove things as we successfully bootstrap their
	// watchers. (I wish we had a set type here...)
	pendingResources map[string]bool

	// pendingMutex prevents multiple simultaneous passes through the code
	// that adds a watcher. (Why? First, because having multiple watchers for
	// the same resource is silly. Second, because I've often seen the CRD
	// watcher fire two events when a new CRD appears.)
	pendingMutex sync.Mutex

	// logger is a function to log things for us.
	logger func(format string, args ...interface{})
}

func fmtNamespace(ns string) string {
	if ns == "" {
		return "*"
	}
	return ns
}

// SaveError emits an error from kubebootstrap with the given message
func (b *kubebootstrap) SaveError(message string) {
	evt := makeErrorEvent("kubebootstrap", message)
	for _, n := range b.notify {
		n <- evt
	}
}

// makeWatcherFunc returns a watcher function tailored to a particular namespace
// and kind, suitable for passing to any of the kubeAPIWatcher watch methods.
func (b *kubebootstrap) makeWatcherFunc(ns, kind string) func(watcher *k8s.Watcher) {
	return func(watcher *k8s.Watcher) {
		resources := watcher.List(kind)

		b.logger("found %d %q in namespace %q", len(resources), kind, fmtNamespace(ns))

		for _, n := range b.notify {
			n <- k8sEvent{kind: kind, resources: resources}
		}

		b.logger("sent %q to %d receivers", kind, len(b.notify))
	}
}

// tryToWatchAllPending walks over all of our pendingResources and tries
// to get their watchers running.
//
// If runImmediately is true, start the watcher running as soon as we add
// it. Otherwise assume that our caller will start the watcher running
// later.
func (b *kubebootstrap) tryToWatchAllPending(runImmediately bool) error {
	// We don't want to run in parallel, so use b.pendingMutex to force
	// serialization through here. (Running in parallel could result in
	// multiple watchers for the same CRD, which would be silly at best.)

	// b.logger("pendingMutex lock")
	b.pendingMutex.Lock()

	defer func() {
		// b.logger("pendingMutex unlock")
		b.pendingMutex.Unlock()
	}()

	// If we have no pendingResources, we're done.
	if len(b.pendingResources) == 0 {
		b.logger("no resource types are pending")
		return nil
	}

	// OK, we do indeed have some pendingResources. Set up the watcher for each of
	// them.

	for kind := range b.pendingResources {
		// Add a SelectiveWatch for this resource kind.
		err := b.kubeAPIWatcher.SelectiveWatch(b.namespace, kind, b.fieldSelector, b.labelSelector,
			b.makeWatcherFunc(b.namespace, kind))

		if err != nil {
			// Hmmm, this isn't good. Mark this resource type as _not_ required
			// (since we'll never get results for it)...
			b.markRequired(kind, false)

			// ...and look at the error.
			if errors.Is(err, k8s.ErrUnkResource) {
				// The resource type doesn't exist in the cluster. We'll assume that it's a
				// missing CRD type, and try again later.
				b.logger("%q does not exist in the cluster at this time: will try later on...", kind)
			} else {
				// Oops. This is a Real Error.
				return err
			}
		} else {
			// No errors! Mark this resource type as required...
			b.markRequired(kind, true)

			// ...remove it from the set of pendingResources...
			// b.logger("watcher for %q successfully installed", kind)
			delete(b.pendingResources, kind)

			// ...and if we're allowed to runImmediately...
			if runImmediately {
				// ...then go ahead and start the watcher. This will actually do the
				// initial synchronization of any extant resources and then start watching
				// for new resources being created.
				b.logger("Starting watcher for %q", kind)
				phase, err := b.kubeAPIWatcher.StartWatcherForKind(kind)

				if err != nil {
					b.logger("error starting watcher for %q: %q, %q", kind, phase, err)
					return err
				}
			}
		}
	}

	b.logger("%d resources are pending", len(b.pendingResources))
	return nil
}

// Work starts the Kubernetes bootstrapping process.
func (b *kubebootstrap) Work(p *supervisor.Process) error {
	b.logger = p.Logf
	b.pendingResources = make(map[string]bool)

	// Start by marking all the kinds we're interested in as pending...
	for _, kind := range b.kinds {
		b.pendingResources[kind] = true
	}

	// ...then try to add watchers for all of them.
	//
	// Don't allow immediately starting all the watchers here -- this is bootstrap
	// code, and we want to start everything running later, all at once.
	if err := b.tryToWatchAllPending(false); err != nil {
		return err
	}

	// Some CRDs can be missing from the cluster (like after an upgrade, where users have
	// changed only the image). In those cases, we should not fail -- instead we'll install
	// a watcher for CRDs so that we can notice when the new CRD type becomes available, and
	// start watching for individual CRs at that point.

	if len(b.pendingResources) > 0 {
		p.Logf("setting up to watch for new CRDs...")

		// We'll use this watcherFunc as the function to be called when we see a new CRD...
		watcherFunc := func(watcher *k8s.Watcher) {
			p.Logf("new CRD! spawning goroutine to refresh...")

			// ...which means that when this function is actually called, we'll be inside
			// the .invoke() call for the CRD watch. That .invoke() call will have locked
			// the K8s Watcher's mutex, which means that it would be a Bad Idea to just
			// naively call tryToWatchAllPendingTypes(true) here: starting the new watcher
			// would involve calling the new watcher's .invoke(), which would try to lock
			// the K8s Watcher's mutex again, which would cause a deadlock.
			//
			// So we spin this off into another goroutine, so we can return and thus allow
			// the mutex to be unlocked.
			go func() {
				_ = b.kubeAPIWatcher.Refresh()
				p.Logf("retrying all pending types...")
				_ = b.tryToWatchAllPending(true)
			}()
		}

		// OK, we have our watcherFunc. Install a simple watch here -- no namespaces, no
		// selectors, just look for all customresourcedefinitions.
		if err := b.kubeAPIWatcher.Watch("customresourcedefinitions", watcherFunc); err != nil {
			return fmt.Errorf("could not watch CRDs: %w", err)
		}
	}

	// At this point we've installed watchers for all the resources that we can, but we
	// haven't started any of them running. Time to do that now.
	p.Logf("Starting resource watchers...")
	b.kubeAPIWatcher.StartWithErrorHandler(func(kind string, stage string, err error) {
		p.Logf("could not watch %q at stage %q: %q", kind, stage, err)
	})

	// Finally, we can mark the kubewatchman ready...
	p.Logf("Marking kubewatchman ready")
	p.Ready()

	// ...and set up for shutdown.
	for range p.Shutdown() {
		p.Logf("shutdown initiated")
		b.kubeAPIWatcher.Stop()
	}

	return nil
}
