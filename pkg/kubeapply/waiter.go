package kubeapply

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
)

// Waiter takes some YAML and waits for all of the resources described
// in it to be ready.
type Waiter struct {
	client *kates.Client
	kinds  map[kates.GroupVersionKind]map[string]struct{}
}

// NewWaiter constructs a Waiter object based on the supplied Watcher.
func NewWaiter(client *kates.Client) (w *Waiter, err error) {
	return &Waiter{
		client: client,
		kinds:  make(map[kates.GroupVersionKind]map[string]struct{}),
	}, nil
}

func gvkStr(gvk kates.GroupVersionKind) string {
	return gvk.Kind + "." + gvk.Version + "." + gvk.Group
}

func (w *Waiter) qKind(resource kates.Object) string {
	return gvkStr(resource.GetObjectKind().GroupVersionKind())
}

func (w *Waiter) qName(resource kates.Object) (string, error) {
	qName := resource.GetName()

	namespaced, err := w.client.IsNamespaced(resource.GetObjectKind().GroupVersionKind())
	if err != nil {
		return "", err
	}

	if namespaced {
		namespace := resource.GetNamespace()
		if namespace == "" {
			namespace, err = w.client.CurrentNamespace()
			if err != nil {
				return "", err
			}
		}
		qName += "." + namespace
	}

	return qName, nil
}

func (w *Waiter) add(resource kates.Object) error {
	resourceType := resource.GetObjectKind().GroupVersionKind()

	resourceName, err := w.qName(resource)
	if err != nil {
		return err
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
			qName, err := w.qName(res)
			if err != nil {
				qName = res.GetName() + "." + res.GetNamespace()
			}
			return fmt.Errorf("%s/%s: %w", w.qKind(res), qName, err)
		}
	}
	return nil
}

func (w *Waiter) remove(kind kates.GroupVersionKind, name string) {
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
	ctx, cancelEverything := context.WithCancel(ctx)
	defer cancelEverything()

	deadlineTimer := time.NewTimer(time.Until(deadline))
	defer deadlineTimer.Stop()

	queries := []kates.Query{
		{Kind: "Events.v1.", Namespace: kates.NamespaceAll},
	}
	for gvk := range w.kinds {
		queries = append(queries, kates.Query{
			Kind:      gvk.Kind + "." + gvk.Version + "." + gvk.Group,
			Namespace: kates.NamespaceAll,
		})
	}

	acc, err := w.client.Watch(ctx, queries...)
	if err != nil {
		return false, err
	}

	defer func() {
		for kind, names := range w.kinds {
			for name := range names {
				dlog.Errorf(ctx, "not ready: %s/%s", gvkStr(kind), name)
			}
		}
	}()

	start := time.Now()
	printed := make(map[string]bool)
	for {
		if w.isEmpty() {
			return true, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-deadlineTimer.C:
			return false, errorDeadlineExceeded
		case <-acc.Changed():
			var events []*kates.Event
			if err := w.client.List(ctx, queries[0], &events); err != nil {
				return false, err
			}
			for _, event := range events {
				if event.LastTimestamp.Time.Before(start) && !event.LastTimestamp.IsZero() {
					continue
				}
				eventQName, err := w.qName(event)
				if err != nil {
					return false, err
				}
				if !printed[eventQName] {
					involvedGVK := kates.GroupVersionKindFromAPIVersionAndKind(event.InvolvedObject.APIVersion, event.InvolvedObject.Kind)
					involvedIsNamespaced, err := w.client.IsNamespaced(involvedGVK)
					involvedQName := event.InvolvedObject.Name
					if involvedIsNamespaced || err != nil {
						involvedQName += "." + event.InvolvedObject.Namespace
					}
					dlog.Printf(ctx, "event: %s/%s: %s", gvkStr(involvedGVK), involvedQName, event.Message)
					printed[eventQName] = true
				}
			}

			for gvk, names := range w.kinds {
				for name := range names {
					r := new(kates.Unstructured)
					r.GetObjectKind().SetGroupVersionKind(gvk)
					namespaced, err := w.client.IsNamespaced(gvk)
					if err != nil {
						return false, err
					}
					if namespaced {
						dot := strings.LastIndexByte(name, '.')
						r.SetName(name[:dot])
						r.SetNamespace(name[dot+1:])
					} else {
						r.SetName(name)
					}
					if err := w.client.Get(ctx, r, &r); err != nil {
						return false, err
					}
					if Ready(w.client, r) {
						qKind := w.qKind(r)
						qName, err := w.qName(r)
						if err != nil {
							return false, err
						}
						if ReadyImplemented(r) {
							dlog.Printf(ctx, "ready: %s/%s\n",
								qKind, qName)
						} else {
							dlog.Printf(ctx, "ready: %s/%s (UNIMPLEMENTED)\n",
								qKind, qName)
						}
						w.remove(gvk, name)
					}
				}
			}
		}
	}
}
