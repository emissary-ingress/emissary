package kubeapply

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/datawire/ambassador/v2/pkg/k8s"
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
func (w *Waiter) Scan(ctx context.Context, path string) (err error) {
	resources, err := LoadResources(ctx, path)
	for _, res := range resources {
		err = w.add(res)
		if err != nil {
			return
		}
	}
	return
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
func (w *Waiter) Wait(ctx context.Context, deadline time.Time) bool {
	start := time.Now()
	printed := make(map[string]bool)
	err := w.watcher.WatchQuery(k8s.Query{Kind: "events", Namespace: k8s.NamespaceAll}, func(watcher *k8s.Watcher) {
		for _, r := range watcher.List("events") {
			if lastStr, ok := r["lastTimestamp"].(string); ok {
				last, err := time.Parse("2006-01-02T15:04:05Z", lastStr)
				if err != nil {
					log.Println(err)
					continue
				}
				if last.Before(start) {
					continue
				}
			}
			if !printed[r.QName()] {
				var name string
				if obj, ok := r["involvedObject"].(map[string]interface{}); ok {
					res := k8s.Resource(obj)
					name = fmt.Sprintf("%s/%s", res.QKind(), res.QName())
				} else {
					name = r.QName()
				}
				dlog.Printf(ctx, "event: %s %s\n", name, r["message"])
				printed[r.QName()] = true
			}
		}
	})
	if err != nil {
		panic(err)
	}

	listener := func(watcher *k8s.Watcher) {
		for kind, names := range w.kinds {
			for name := range names {
				r := watcher.Get(kind.String(), name)
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
	}

	for k := range w.kinds {
		err := w.watcher.WatchQuery(k8s.Query{Kind: k.String(), Namespace: k8s.NamespaceAll}, listener)
		if err != nil {
			panic(err)
		}
	}

	w.watcher.Start(ctx)

	go func() {
		time.Sleep(time.Until(deadline))
		w.watcher.Stop()
	}()

	w.watcher.Wait(ctx)

	result := true

	for kind, names := range w.kinds {
		for name := range names {
			fmt.Printf("not ready: %s/%s\n", kind, name)
			result = false
		}
	}

	return result
}
