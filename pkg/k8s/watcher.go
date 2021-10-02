package k8s

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	pwatch "k8s.io/apimachinery/pkg/watch"

	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/cache"
)

type listWatchAdapter struct {
	resource      dynamic.ResourceInterface
	fieldSelector string
	labelSelector string
}

func (lw listWatchAdapter) List(options v1.ListOptions) (runtime.Object, error) {
	options.FieldSelector = lw.fieldSelector
	options.LabelSelector = lw.labelSelector
	// silently coerce the returned *unstructured.UnstructuredList
	// struct to a runtime.Object interface.
	return lw.resource.List(context.TODO(), options)
}

func (lw listWatchAdapter) Watch(options v1.ListOptions) (pwatch.Interface, error) {
	options.FieldSelector = lw.fieldSelector
	options.LabelSelector = lw.labelSelector
	return lw.resource.Watch(context.TODO(), options)
}

// Watcher is a kubernetes watcher that can watch multiple queries simultaneously
type Watcher struct {
	Client  *Client
	watches map[ResourceType]watch
	stop    chan struct{}
	wg      sync.WaitGroup
	mutex   sync.Mutex
	stopMu  sync.Mutex
	started bool
	stopped bool
}

type watch struct {
	query    Query
	resource dynamic.NamespaceableResourceInterface
	store    cache.Store
	invoke   func()
	runner   func()
}

// NewWatcher returns a Kubernetes watcher for the specified cluster.
func NewWatcher(info *KubeInfo) (*Watcher, error) {
	cli, err := NewClient(info)
	if err != nil {
		return nil, err
	}
	return cli.Watcher(), nil
}

// Watcher returns a Kubernetes Watcher for the specified client.
func (c *Client) Watcher() *Watcher {
	w := &Watcher{
		Client:  c,
		watches: make(map[ResourceType]watch),
		stop:    make(chan struct{}),
	}

	return w
}

// WatchQuery watches the set of resources identified by the supplied
// query and invokes the supplied listener whenever they change.
func (w *Watcher) WatchQuery(query Query, listener func(*Watcher) error) error {
	err := query.resolve(w.Client)
	if err != nil {
		return err
	}
	ri := query.resourceType

	dyn, err := dynamic.NewForConfig(w.Client.config)
	if err != nil {
		return err
	}

	resource := dyn.Resource(schema.GroupVersionResource{
		Group:    ri.Group,
		Version:  ri.Version,
		Resource: ri.Name,
	})

	var watched dynamic.ResourceInterface
	if ri.Namespaced && query.Namespace != "" {
		watched = resource.Namespace(query.Namespace)
	} else {
		watched = resource
	}

	invoke := func() {
		w.mutex.Lock()
		defer w.mutex.Unlock()
		if err := listener(w); err != nil {
			panic(fmt.Errorf("I'm sorry, the pkg/k8s API really painted us in to a hole and I couldn't handle this error properly: %w", err))
		}
	}

	store, controller := cache.NewInformer(
		listWatchAdapter{watched, query.FieldSelector, query.LabelSelector},
		nil,
		5*time.Minute,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				invoke()
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldUn := oldObj.(*unstructured.Unstructured)
				newUn := newObj.(*unstructured.Unstructured)
				// we ignore updates for objects
				// already in our store because we
				// assume this means we made the
				// change to them
				if oldUn.GetResourceVersion() != newUn.GetResourceVersion() {
					// kube-scheduler and kube-controller-manager endpoints are
					// updated almost every second, leading to terrible noise,
					// and hence constant listener invokation. So, here we
					// ignore endpoint updates from kube-system namespace. More:
					// https://github.com/kubernetes/kubernetes/issues/41635
					// https://github.com/kubernetes/kubernetes/issues/34627
					if oldUn.GetKind() == "Endpoints" &&
						newUn.GetKind() == "Endpoints" &&
						oldUn.GetNamespace() == "kube-system" &&
						newUn.GetNamespace() == "kube-system" {
						return
					}
					invoke()
				}
			},
			DeleteFunc: func(obj interface{}) {
				invoke()
			},
		},
	)

	runner := func() {
		controller.Run(w.stop)
		w.wg.Done()
	}

	w.watches[ri] = watch{
		query:    query,
		resource: resource,
		store:    store,
		invoke:   invoke,
		runner:   runner,
	}

	return nil
}

// Start starts the watcher
func (w *Watcher) Start(ctx context.Context) error {
	w.mutex.Lock()
	if w.started {
		w.mutex.Unlock()
		return nil
	} else {
		w.started = true
		w.mutex.Unlock()
	}
	for kind := range w.watches {
		if err := w.sync(ctx, kind); err != nil {
			return err
		}
	}

	for _, watch := range w.watches {
		watch.invoke()
	}

	w.wg.Add(len(w.watches))
	for _, watch := range w.watches {
		go watch.runner()
	}
	return nil
}

func (w *Watcher) sync(ctx context.Context, kind ResourceType) error {
	watch := w.watches[kind]
	resources, err := w.Client.ListQuery(ctx, watch.query)
	if err != nil {
		return err
	}
	for _, rsrc := range resources {
		var uns unstructured.Unstructured
		uns.SetUnstructuredContent(rsrc)
		err = watch.store.Update(&uns)
		if err != nil {
			return err
		}
	}
	return nil
}

// List lists all the resources with kind `kind`
func (w *Watcher) List(kind string) ([]Resource, error) {
	ri, err := w.Client.ResolveResourceType(kind)
	if err != nil {
		return nil, err
	}
	watch, ok := w.watches[ri]
	if ok {
		objs := watch.store.List()
		result := make([]Resource, len(objs))
		for idx, obj := range objs {
			result[idx] = obj.(*unstructured.Unstructured).UnstructuredContent()
		}
		return result, nil
	}
	return nil, nil
}

// UpdateStatus updates the status of the `resource` provided
func (w *Watcher) UpdateStatus(ctx context.Context, resource Resource) (Resource, error) {
	ri, err := w.Client.ResolveResourceType(resource.QKind())
	if err != nil {
		return nil, err
	}
	watch, ok := w.watches[ri]
	if !ok {
		return nil, fmt.Errorf("no watch: %v, %v", ri, w.watches)
	}

	var uns unstructured.Unstructured
	uns.SetUnstructuredContent(resource)

	var cli dynamic.ResourceInterface
	if ri.Namespaced {
		cli = watch.resource.Namespace(uns.GetNamespace())
	} else {
		cli = watch.resource
	}

	result, err := cli.UpdateStatus(ctx, &uns, v1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	if err := watch.store.Update(result); err != nil {
		return nil, err
	}
	return result.UnstructuredContent(), nil
}

// Get gets the `qname` resource (of kind `kind`)
func (w *Watcher) Get(kind, qname string) (Resource, error) {
	resources, err := w.List(kind)
	if err != nil {
		return Resource{}, err
	}
	for _, res := range resources {
		if strings.EqualFold(res.QName(), qname) {
			return res, nil
		}
	}
	return Resource{}, nil
}

// Exists returns true if the `qname` resource (of kind `kind`) exists
func (w *Watcher) Exists(kind, qname string) (bool, error) {
	resource, err := w.Get(kind, qname)
	if err != nil {
		return false, err
	}
	return resource.Name() != "", nil
}

// Stop stops a watch. It is safe to call Stop from multiple
// goroutines and call it multiple times. This is useful, e.g. for
// implementing a timed wait pattern. You can have your watch callback
// test for a condition and invoke Stop() when that condition is met,
// while simultaneously havin a background goroutine call Stop() when
// a timeout is exceeded and not worry about these two things racing
// each other (at least with respect to invoking Stop()).
func (w *Watcher) Stop() {
	// Use a separate lock for Stop so it is safe to call from a watch callback.
	w.stopMu.Lock()
	defer w.stopMu.Unlock()
	if !w.stopped {
		close(w.stop)
		w.stopped = true
	}
}

func (w *Watcher) Wait(ctx context.Context) error {
	if err := w.Start(ctx); err != nil {
		return err
	}
	w.wg.Wait()
	return nil
}
