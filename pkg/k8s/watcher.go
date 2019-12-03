package k8s

import (
	"fmt"
	"io" // for panic stuff
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	pwatch "k8s.io/apimachinery/pkg/watch"

	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/cache"

	"github.com/pkg/errors" // for panic stuff
)

//==== copypasta from apro/lib/util/panic.go
// package util

// import (
// 	"fmt"
// 	"io"

// 	"github.com/pkg/errors"
// )

// causer is not exported by github.com/pkg/errors.
type causer interface {
	Cause() error
}

// stackTracer is not exported by github.com/pkg/errors.
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// featurefulError documents the features of
// github.com/pkg/errors.Wrap().
type featurefulError interface {
	error
	//causer
	stackTracer
	fmt.Formatter
}

type panicError struct {
	err featurefulError
}

func (pe panicError) Error() string                 { return "PANIC: " + pe.err.Error() }
func (pe panicError) Cause() error                  { return pe.err }
func (pe panicError) StackTrace() errors.StackTrace { return pe.err.StackTrace()[1:] }
func (pe panicError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		io.WriteString(s, "PANIC: ")
		if s.Flag('+') {
			fmt.Fprintf(s, "%v", pe.err)
			pe.StackTrace().Format(s, verb)
			return
		}
		io.WriteString(s, pe.err.Error())
	case 's':
		io.WriteString(s, pe.Error())
	case 'q':
		fmt.Fprintf(s, "%q", pe.Error())
	}
}

var _ causer = panicError{}
var _ featurefulError = panicError{}

// PanicToError takes an arbitrary object returned from recover(), and
// returns an appropriate error.
//
// If the input is nil, then nil is returned.
//
// If the input is an error returned from a previus call to
// PanicToError(), then it is returned verbatim.
//
// If the input is an error, it is wrapped with the message "PANIC:"
// and has a stack trace attached to it.
//
// If the input is anything else, it is formatted with "%+v" and
// returned as an error with a stack trace attached.
func PanicToError(rec interface{}) error {
	if rec == nil {
		return nil
	}
	switch rec := rec.(type) {
	case panicError:
		return rec
	case error:
		return panicError{err: errors.WithStack(rec).(featurefulError)}
	default:
		return panicError{err: errors.Errorf("%+v", rec).(featurefulError)}
	}
}

//=== copypasta ends

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
	return lw.resource.List(options)
}

func (lw listWatchAdapter) Watch(options v1.ListOptions) (pwatch.Interface, error) {
	options.FieldSelector = lw.fieldSelector
	options.LabelSelector = lw.labelSelector
	return lw.resource.Watch(options)
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

// MustNewWatcher returns a Kubernetes watcher for the specified
// cluster or panics.
func MustNewWatcher(info *KubeInfo) *Watcher {
	w, err := NewWatcher(info)
	if err != nil {
		panic(err)
	}
	return w
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

func (w *Watcher) Watch(resources string, listener func(*Watcher)) error {
	return w.WatchNamespace("", resources, listener)
}

func (w *Watcher) WatchNamespace(namespace, resources string, listener func(*Watcher)) error {
	return w.SelectiveWatch(namespace, resources, "", "", listener)
}

func (w *Watcher) Refresh() error {
	return w.Client.Refresh()
}

func (w *Watcher) SelectiveWatch(namespace, resources, fieldSelector, labelSelector string,
	listener func(*Watcher)) error {
	return w.WatchQuery(Query{
		Kind:          resources,
		Namespace:     namespace,
		FieldSelector: fieldSelector,
		LabelSelector: labelSelector,
	}, listener)
}

// WatchQuery watches the set of resources identified by the supplied
// query and invokes the supplied listener whenever they change.
func (w *Watcher) WatchQuery(query Query, listener func(*Watcher)) error {
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
		log.Debugf("invoke %q lock", ri.String())
		w.mutex.Lock()

		defer func() {
			log.Debugf("invoke %q unlock", ri.String())
			w.mutex.Unlock()
		}()

		log.Debugf("invoke %q listener", ri.String())
		listener(w)
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
func (w *Watcher) Start() {
	w.StartWithErrorHandler(nil)
}

// StartWithErrorHandler starts the watcher, but allows supplying an error handler to call
// instead of panic()ing on errors.
func (w *Watcher) StartWithErrorHandler(handler func(kind string, stage string, err error)) {
	w.mutex.Lock()
	if w.started {
		w.mutex.Unlock()
		return
	} else {
		w.started = true
		w.mutex.Unlock()
	}

	// Make sure that we do all the initial types in a block and _then_ start everything
	// going, so that we don't reconfigure over and over again at boot.
	ableToWatch := make(map[ResourceType]watch)

	for kind, watch := range w.watches {
		stage, err := w.SyncWatcherForResourceType(kind)

		if err == nil {
			ableToWatch[kind] = watch
		} else {
			if handler != nil {
				log.Infof("handling %q err %q", stage, err)
				handler(kind.String(), stage, err)
			} else {
				log.Infof("unhandled %q err %q", stage, err)
				panic(err)
			}

			return
		}
	}

	for kind := range ableToWatch {
		w.RunWatcherForResourceType(kind)
	}
}

// StartWatcherForKind fully starts the watcher for a single kind of resource.
func (w *Watcher) StartWatcherForKind(kind string) (string, error) {
	resourceType, err := w.Client.ResolveResourceType(kind)

	if err != nil {
		return "resolve", err
	}

	return w.StartWatcherForResourceType(resourceType)
}

// StartWatcherForResourceType fully starts the watcher for a single ResourceType.
func (w *Watcher) StartWatcherForResourceType(resourceType ResourceType) (string, error) {
	log.Debugf("Sync %q", resourceType.String())
	stage, err := w.SyncWatcherForResourceType(resourceType)

	if err != nil {
		return stage, err
	}

	log.Debugf("Run %q", resourceType.String())
	w.RunWatcherForResourceType(resourceType)

	return "", nil
}

// SyncWatcherForResourceType does the initial synchronization for a single ResourceType.
func (w *Watcher) SyncWatcherForResourceType(resourceType ResourceType) (string, error) {
	watch := w.watches[resourceType]

	// First try to sync.
	log.Debugf("catcher and sync %q", resourceType.String())
	err := w.catcher(func() { w.sync(resourceType) })

	if err != nil {
		return "sync", err
	}

	// Next, try to start the watch running.
	log.Debugf("catcher and invoke %q", resourceType.String())
	err = w.catcher(func() { watch.invoke() })

	if err != nil {
		return "invoke", err
	}

	// Done.
	log.Debugf("sync done %q", resourceType.String())
	return "", nil
}

// RunWatcherForResourceType starts running the watcher for a single ResourceType, assuming
// that it has already been synchronized.
func (w *Watcher) RunWatcherForResourceType(resourceType ResourceType) {
	watch := w.watches[resourceType]

	// Note the presence of the additional wait for our waitgroup...
	w.wg.Add(1)

	// ...and start the watch's runner.
	log.Infof("starting watch runner for %q", resourceType.String())
	go watch.runner()

	log.Debugf("start done %q", resourceType.String())
}

func (w *Watcher) catcher(doSomething func()) (err error) {
	defer func() {
		if _err := PanicToError(recover()); _err != nil {
			err = _err
		}
	}()

	doSomething()

	return
}

func (w *Watcher) sync(kind ResourceType) {
	watch := w.watches[kind]
	resources, err := w.Client.ListQuery(watch.query)
	if err != nil {
		panic(err)
	}
	for _, rsrc := range resources {
		var uns unstructured.Unstructured
		uns.SetUnstructuredContent(rsrc)
		err = watch.store.Update(&uns)
		if err != nil {
			panic(err)
		}
	}
}

// List lists all the resources with kind `kind`
func (w *Watcher) List(kind string) []Resource {
	ri, err := w.Client.ResolveResourceType(kind)
	if err != nil {
		panic(err)
	}
	watch, ok := w.watches[ri]
	if ok {
		objs := watch.store.List()
		result := make([]Resource, len(objs))
		for idx, obj := range objs {
			result[idx] = obj.(*unstructured.Unstructured).UnstructuredContent()
		}
		return result
	} else {
		return nil
	}
}

// UpdateStatus updates the status of the `resource` provided
func (w *Watcher) UpdateStatus(resource Resource) (Resource, error) {
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

	result, err := cli.UpdateStatus(&uns, v1.UpdateOptions{})
	if err != nil {
		return nil, err
	} else {
		watch.store.Update(result)
		return result.UnstructuredContent(), nil
	}
}

// Get gets the `qname` resource (of kind `kind`)
func (w *Watcher) Get(kind, qname string) Resource {
	resources := w.List(kind)
	for _, res := range resources {
		if strings.EqualFold(res.QName(), qname) {
			return res
		}
	}
	return Resource{}
}

// Exists returns true if the `qname` resource (of kind `kind`) exists
func (w *Watcher) Exists(kind, qname string) bool {
	return w.Get(kind, qname).Name() != ""
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

func (w *Watcher) Wait() {
	w.Start()
	w.wg.Wait()
}
