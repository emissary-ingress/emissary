package main

import (
	//	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"
	//	"k8s.io/api/core/v1"
	//	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	//	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type LW struct {
	resource dynamic.NamespaceableResourceInterface
}

func (lw LW) List(options v1.ListOptions) (runtime.Object, error) {
	return lw.resource.List(options)
}

func (lw LW) Watch(options v1.ListOptions) (watch.Interface, error) {
	return lw.resource.Watch(options)
}

func controller(kubeconfig string, reconciler func([]map[string]interface{})) {
	var config *rest.Config
	var err error
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	} else {
		if kubeconfig == "" {
			current, err := user.Current()
			if err != nil {
				panic(err)
			}
			home := current.HomeDir
			kubeconfig = filepath.Join(home, ".kube/config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	resource := dyn.Resource(schema.GroupVersionResource{
		Group:    "stable.datawire.io",
		Version:  "v1beta1",
		Resource: "policies",
	})
	/*	resource := dyn.Resource(schema.GroupVersionResource {
		Version: "v1",
		Resource: "services",
	})*/

	var store cache.Store

	reconcile := func() {
		objs := store.List()
		uns := make([]map[string]interface{}, len(objs))
		for idx, obj := range objs {
			uns[idx] = obj.(*unstructured.Unstructured).UnstructuredContent()
		}
		reconciler(uns)
	}

	store, controller := cache.NewInformer(
		LW{resource},
		nil,
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				reconcile()
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				reconcile()
			},
			DeleteFunc: func(obj interface{}) {
				reconcile()
			},
		},
	)
	controller.Run(make(chan struct{}))
}
