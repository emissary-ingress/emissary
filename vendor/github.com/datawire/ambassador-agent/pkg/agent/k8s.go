package agent

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/datawire/dlib/dlog"
)

// CallbackEventType defines the possible callback types of events.
type CallbackEventType string

const (
	CallbackEventAdded   CallbackEventType = "ADDED"
	CallbackEventDeleted CallbackEventType = "DELETED"
	CallbackEventUpdated CallbackEventType = "UPDATED"
)

// GenericCallback is used to be returned in the channel managed by the WatchGeneric method.
type GenericCallback struct {
	// EventType is the event type that originated this callback.
	EventType CallbackEventType

	// Obj has the new resource state for this event type. If event type is CallbackEventDeleted
	// it will contain the last resource state before being deleted.
	Obj *unstructured.Unstructured

	// Sotw has the state of the world for all resources of the type being watched.
	Sotw []interface{}
}

// DynamicClient is the struct that provides the main functionality of watching
// generic Kubernetes resources that may of may not be available (installed) in
// the cluster.
type DynamicClient struct {
	newInformer InformerFunc
	di          dynamic.Interface
	done        bool
	mux         sync.Mutex
}

// NewDynamicClient is the main contructor of DynamicClient
func NewDynamicClient(di dynamic.Interface, informerFn InformerFunc) *DynamicClient {
	return &DynamicClient{
		newInformer: informerFn,
		di:          di,
	}
}

// Informer holds the operations necessary from a k8s informer in order to
// provide the functionality to watch a generic resource.
type Informer interface {
	AddEventHandler(handler cache.ResourceEventHandler)
	Run(stopCh <-chan struct{})
	ListCache() []interface{}
}

type InformerFunc func(dynamic.Interface, string, *schema.GroupVersionResource) Informer

// K8sInformer is a real Informer implementation.
type K8sInformer struct {
	cache.SharedIndexInformer
}

// ListCache will return the current state of the cache store from the Kubernetes
// informer.
func (i *K8sInformer) ListCache() []interface{} {
	return i.GetStore().List()
}

// NewK8sInformer builds and returns a real Kubernetes Informer implementation.
func NewK8sInformer(cli dynamic.Interface, ns string, gvr *schema.GroupVersionResource) Informer {
	f := dynamicinformer.NewFilteredDynamicSharedInformerFactory(cli, 0, ns, nil)
	i := f.ForResource(*gvr).Informer()
	return &K8sInformer{
		SharedIndexInformer: i,
	}
}

func (dc *DynamicClient) sendCallback(callbackChan chan<- *GenericCallback, callback *GenericCallback) {
	dc.mux.Lock()
	defer dc.mux.Unlock()
	if dc.done {
		return
	}
	callbackChan <- callback
}

// WatchGeneric will watch any resource existing in the cluster or not. This is usefull for
// watching CRDs that may or may not be available in the cluster.
func (dc *DynamicClient) WatchGeneric(ctx context.Context, ns string, gvr *schema.GroupVersionResource) <-chan *GenericCallback {
	callbackChan := make(chan *GenericCallback)
	go func() {
		<-ctx.Done()
		dc.mux.Lock()
		defer dc.mux.Unlock()
		dc.done = true
		close(callbackChan)
	}()
	i := dc.newInformer(dc.di, ns, gvr)
	i.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				dlog.Debugf(ctx, "WatchGeneric: AddFunc called for resource %q", gvr.String())
				new := obj.(*unstructured.Unstructured)
				sotw := i.ListCache()
				callback := &GenericCallback{EventType: CallbackEventAdded, Obj: new, Sotw: sotw}
				dc.sendCallback(callbackChan, callback)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				dlog.Debugf(ctx, "WatchGeneric: UpdateFunc called for resource %q", gvr.String())
				new := newObj.(*unstructured.Unstructured)
				sotw := i.ListCache()
				callback := &GenericCallback{EventType: CallbackEventUpdated, Obj: new, Sotw: sotw}
				dc.sendCallback(callbackChan, callback)
			},
			DeleteFunc: func(obj interface{}) {
				dlog.Debugf(ctx, "WatchGeneric: DeleteFunc called for resource %q", gvr.String())
				var old *unstructured.Unstructured
				switch o := obj.(type) {
				case cache.DeletedFinalStateUnknown:
					old = o.Obj.(*unstructured.Unstructured)
				case *unstructured.Unstructured:
					old = o
				}
				sotw := i.ListCache()
				callback := &GenericCallback{EventType: CallbackEventDeleted, Obj: old, Sotw: sotw}
				dc.sendCallback(callbackChan, callback)
			},
		},
	)
	go i.Run(ctx.Done())
	dlog.Infof(ctx, "WatchGeneric: Listening for events from resouce %q", gvr.String())
	return callbackChan
}

func newK8sRestClient() (*rest.Config, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}
