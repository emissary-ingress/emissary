package kubeapply

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/datawire/ambassador/v2/pkg/k8s"
	kates_internal "github.com/datawire/ambassador/v2/pkg/kates_internal"
	"github.com/datawire/dlib/dlog"
)

// Waiter takes some YAML and waits for all of the resources described
// in it to be ready.
type Waiter struct {
	watcher *k8s.Watcher
	kinds   map[k8s.ResourceType]map[string]struct{}
}

// NewWaiter constructs a Waiter object based on the supplied Watcher.
func NewWaiter(watcher *k8s.Watcher) (w *Waiter, err error) {
	if watcher == nil {
		cli, err := k8s.NewClient(nil)
		if err != nil {
			return nil, err
		}
		watcher = cli.Watcher()
	}
	return &Waiter{
		watcher: watcher,
		kinds:   make(map[k8s.ResourceType]map[string]struct{}),
	}, nil
}

func (w *Waiter) add(resource k8s.Resource) error {
	resourceType, err := w.watcher.Client.ResolveResourceType(resource.QKind())
	if err != nil {
		return err
	}

	resourceName := resource.Name()
	if resourceType.Namespaced {
		namespace := resource.Namespace()
		if namespace == "" {
			namespace = w.watcher.Client.Namespace
		}
		resourceName += "." + namespace
	}

	if _, ok := w.kinds[resourceType]; !ok {
		w.kinds[resourceType] = make(map[string]struct{})
	}
	w.kinds[resourceType][resourceName] = struct{}{}
	return nil
}

// Scan calls LoadResources(path), and add all resources loaded to the
// Waiter.
func (w *Waiter) Scan(ctx context.Context, path string) error {
	resources, err := LoadResources(ctx, path)
	if err != nil {
		return fmt.Errorf("LoadResources: %w", err)
	}
	for _, res := range resources {
		if err = w.add(res); err != nil {
			return fmt.Errorf("%s/%s: %w", res.QKind(), res.QName(), err)
		}
	}
	return nil
}

func (w *Waiter) remove(kind k8s.ResourceType, name string) {
	delete(w.kinds[kind], name)
}

func (w *Waiter) isEmpty() bool {
	for _, names := range w.kinds {
		if len(names) > 0 {
			return false
		}
	}

	return true
}

// Wait spews a bunch of crap on stdout, and waits for all of the
// Scan()ed resources to be ready.  If they all become ready before
// deadline, then it returns true.  If they don't become ready by
// then, then it bails early and returns false.
func (w *Waiter) Wait(ctx context.Context, deadline time.Time) (bool, error) {
	start := time.Now()
	printed := make(map[string]bool)
	err := w.watcher.WatchQuery(k8s.Query{Kind: "Events.v1.", Namespace: k8s.NamespaceAll}, func(watcher *k8s.Watcher) error {
		list, err := watcher.List("Events.v1.")
		if err != nil {
			return err
		}
		for _, untypedEvent := range list {
			var event corev1.Event
			if err := kates_internal.Convert(untypedEvent, &event); err != nil {
				dlog.Errorln(ctx, err)
				continue
			}
			if event.LastTimestamp.Time.Before(start) && !event.LastTimestamp.IsZero() {
				continue
			}
			eventQName := fmt.Sprintf("%s.%s", event.Name, event.Namespace)
			if !printed[eventQName] {
				involvedQKind := k8s.QKind(event.InvolvedObject.APIVersion, event.InvolvedObject.Kind)
				involvedQName := fmt.Sprintf("%s.%s", event.InvolvedObject.Name, event.InvolvedObject.Namespace)

				dlog.Printf(ctx, "event: %s/%s: %s\n", involvedQKind, involvedQName, event.Message)
				printed[eventQName] = true
			}
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	listener := func(watcher *k8s.Watcher) error {
		for kind, names := range w.kinds {
			for name := range names {
				r, err := watcher.Get(kind.String(), name)
				if err != nil {
					return err
				}
				if Ready(r) {
					if ReadyImplemented(r) {
						dlog.Printf(ctx, "ready: %s/%s\n", r.QKind(), r.QName())
					} else {
						dlog.Printf(ctx, "ready: %s/%s (UNIMPLEMENTED)\n",
							r.QKind(), r.QName())
					}
					w.remove(kind, name)
				}
			}
		}

		if w.isEmpty() {
			watcher.Stop()
		}
		return nil
	}

	for k := range w.kinds {
		if err := w.watcher.WatchQuery(k8s.Query{Kind: k.String(), Namespace: k8s.NamespaceAll}, listener); err != nil {
			return false, err
		}
	}

	if err := w.watcher.Start(ctx); err != nil {
		return false, err
	}

	go func() {
		time.Sleep(time.Until(deadline))
		w.watcher.Stop()
	}()

	if err := w.watcher.Wait(ctx); err != nil {
		return false, err
	}

	result := true

	for kind, names := range w.kinds {
		for name := range names {
			fmt.Printf("not ready: %s/%s\n", kind, name)
			result = false
		}
	}

	return result, nil
}
