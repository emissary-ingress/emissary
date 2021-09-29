package thingkube

import (
	"fmt"
	"sync"

	"github.com/datawire/ambassador/v2/cmd/watt/watchapi"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/ambassador/v2/pkg/supervisor"
	"github.com/datawire/ambassador/v2/pkg/watt"
)

type K8sEvent struct {
	WatchID   string
	Kind      string
	Resources []k8s.Resource
	Errors    []watt.Error
}

// makeErrorEvent returns a K8sEvent that contains one error entry for each
// message passed in, all attributed to the same source.
func makeErrorEvent(source string, messages ...string) K8sEvent {
	errors := make([]watt.Error, len(messages))
	for idx, message := range messages {
		errors[idx] = watt.NewError(source, message)
	}
	return K8sEvent{Errors: errors}
}

type KubernetesWatchMaker struct {
	kubeAPI *k8s.Client
	notify  chan<- K8sEvent
}

func (m *KubernetesWatchMaker) MakeKubernetesWatch(spec watchapi.KubernetesWatchSpec) (*supervisor.Worker, error) {
	var worker *supervisor.Worker
	var err error

	worker = &supervisor.Worker{
		Name: fmt.Sprintf("kubernetes:%s", spec.WatchId()),
		Work: func(p *supervisor.Process) error {
			watcher := m.kubeAPI.Watcher()
			watchFunc := func(watchId, ns, kind string) func(watcher *k8s.Watcher) {
				return func(watcher *k8s.Watcher) {
					resources := watcher.List(kind)
					p.Debugf("found %d %q in namespace %q", len(resources), kind, fmtNamespace(ns))
					m.notify <- K8sEvent{WatchID: watchId, Kind: kind, Resources: resources}
					p.Debugf("sent %q to receivers", kind)
				}
			}

			watcherErr := watcher.WatchQuery(k8s.Query{
				Namespace:     spec.Namespace,
				Kind:          spec.Kind,
				FieldSelector: spec.FieldSelector,
				LabelSelector: spec.LabelSelector,
			}, watchFunc(spec.WatchId(), spec.Namespace, spec.Kind))

			if watcherErr != nil {
				return watcherErr
			}

			watcher.Start(p.Context())
			<-p.Shutdown()
			watcher.Stop()
			return nil
		},

		Retry: true,
	}

	return worker, err
}

type kubewatchman struct {
	WatchMaker watchapi.IKubernetesWatchMaker
	in         <-chan []watchapi.KubernetesWatchSpec

	mu      sync.RWMutex
	watched map[string]*supervisor.Worker
}

type KubeWatchMan interface {
	Work(*supervisor.Process) error
	NumWatched() int
	WithWatched(func(map[string]*supervisor.Worker))
}

func NewKubeWatchMan(
	client *k8s.Client,
	eventsCh chan<- K8sEvent,
	watchesCh <-chan []watchapi.KubernetesWatchSpec,
) KubeWatchMan {
	return &kubewatchman{
		WatchMaker: &KubernetesWatchMaker{
			kubeAPI: client,
			notify:  eventsCh,
		},
		in: watchesCh,
	}
}

func (w *kubewatchman) Work(p *supervisor.Process) error {
	p.Ready()

	w.mu.Lock()
	w.watched = make(map[string]*supervisor.Worker)
	w.mu.Unlock()

	for {
		select {
		case watches := <-w.in:
			w.mu.Lock()
			found := make(map[string]*supervisor.Worker)
			p.Debugf("processing %d kubernetes watch specs", len(watches))
			for _, spec := range watches {
				worker, err := w.WatchMaker.MakeKubernetesWatch(spec)
				if err != nil {
					p.Logf("failed to create kubernetes watcher: %v", err)
					continue
				}

				if _, exists := w.watched[worker.Name]; exists {
					found[worker.Name] = w.watched[worker.Name]
				} else {
					p.Debugf("add kubernetes watcher %s\n", worker.Name)
					p.Supervisor().Supervise(worker)
					w.watched[worker.Name] = worker
					found[worker.Name] = worker
				}
			}

			for workerName, worker := range w.watched {
				if _, exists := found[workerName]; !exists {
					p.Debugf("remove kubernetes watcher %s\n", workerName)
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

type kubebootstrap struct {
	namespace      string
	kinds          []string
	fieldSelector  string
	labelSelector  string
	notify         []chan<- K8sEvent
	kubeAPIWatcher *k8s.Watcher
}

type KubeBootstrap interface {
	SaveError(message string)
	Work(*supervisor.Process) error
}

func NewKubeBootstrap(
	namespace string,
	kinds []string,
	fieldSelector string,
	labelSelector string,
	notify []chan<- K8sEvent,
	kubeAPIWatcher *k8s.Watcher,
) KubeBootstrap {
	return &kubebootstrap{
		namespace:      namespace,
		kinds:          kinds,
		fieldSelector:  fieldSelector,
		labelSelector:  labelSelector,
		notify:         notify,
		kubeAPIWatcher: kubeAPIWatcher,
	}
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

func (b *kubebootstrap) Work(p *supervisor.Process) error {
	for _, kind := range b.kinds {
		p.Debugf("adding kubernetes watch for %q in namespace %q", kind, fmtNamespace(b.namespace))

		watcherFunc := func(ns, kind string) func(watcher *k8s.Watcher) {
			return func(watcher *k8s.Watcher) {
				resources := watcher.List(kind)
				p.Debugf("found %d %q in namespace %q", len(resources), kind, fmtNamespace(ns))
				for _, n := range b.notify {
					n <- K8sEvent{Kind: kind, Resources: resources}
				}
				p.Debugf("sent %q to %d receivers", kind, len(b.notify))
			}
		}

		err := b.kubeAPIWatcher.WatchQuery(k8s.Query{
			Namespace:     b.namespace,
			Kind:          kind,
			FieldSelector: b.fieldSelector,
			LabelSelector: b.labelSelector,
		}, watcherFunc(b.namespace, kind))

		if err != nil {
			return err
		}
	}

	b.kubeAPIWatcher.Start(p.Context())
	p.Ready()

	for range p.Shutdown() {
		p.Debugf("shutdown initiated")
		b.kubeAPIWatcher.Stop()
	}

	return nil
}

func (w *kubewatchman) NumWatched() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.watched)
}

func (w *kubewatchman) WithWatched(fn func(map[string]*supervisor.Worker)) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	fn(w.watched)
}
