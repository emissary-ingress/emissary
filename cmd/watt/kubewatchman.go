package watt

import (
	"fmt"

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
					p.Debugf("found %d %q in namespace %q", len(resources), kind, fmtNamespace(ns))
					m.notify <- k8sEvent{watchId: watchId, kind: kind, resources: resources}
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
	notify         []chan<- k8sEvent
	kubeAPIWatcher *k8s.Watcher
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
		p.Debugf("adding kubernetes watch for %q in namespace %q", kind, fmtNamespace(kubernetesNamespace))

		watcherFunc := func(ns, kind string) func(watcher *k8s.Watcher) {
			return func(watcher *k8s.Watcher) {
				resources := watcher.List(kind)
				p.Debugf("found %d %q in namespace %q", len(resources), kind, fmtNamespace(ns))
				for _, n := range b.notify {
					n <- k8sEvent{kind: kind, resources: resources}
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

	b.kubeAPIWatcher.Start()
	p.Ready()

	for range p.Shutdown() {
		p.Debugf("shutdown initiated")
		b.kubeAPIWatcher.Stop()
	}

	return nil
}
