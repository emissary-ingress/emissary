package controller

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/gobwas/glob"
	ms "github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// Controller is a custom Kubernetes controller that monitor k8s
// cluster and load all the rules used by this app for authorizing
// api calls.
type Controller struct {
	Logger *logrus.Logger
	Config *config.Config
	Rules  atomic.Value
}

// Watch monitor changes in k8s cluster and updates rules
func (c *Controller) Watch() {
	c.Logger.Debug("initializing k8s watcher..")
	c.Rules.Store(make([]Rule, 0))

	go controller(c.Config.Kubeconfig, func(uns []map[string]interface{}) {
		newRules := make([]Rule, 0)
		for _, un := range uns {
			spec, ok := un["spec"].(map[string]interface{})
			if !ok {
				c.Logger.Debugf("malformed object, bad spec: %v", uns)
				continue
			}

			unrules, ok := spec["rules"].([]interface{})
			if !ok {
				c.Logger.Debugf("malformed object, bad rules: %v", uns)
				continue
			}

			for _, ur := range unrules {
				rule := Rule{}
				err := ms.Decode(ur, &rule)
				if err != nil {
					c.Logger.Error(err)
				} else {
					c.Logger.Debugf("loading rule: host=%s, path=%s, public=%v, scopes=%s",
						rule.Host, rule.Path, rule.Public, rule.Scopes)
					newRules = append(newRules, rule)
				}
			}
		}
		c.Rules.Store(newRules)
	})
}

// Rule TODO(gsagula): comment
type Rule struct {
	Host   string
	Path   string
	Public bool
	Scopes string
}

// Match TODO(gsagula): comment
func (r Rule) Match(host, path string) bool {
	return match(r.Host, host) && match(r.Path, path)
}

func match(pattern, input string) bool {
	g, err := glob.Compile(pattern)
	if err != nil {
		log.Print(err)
		return false
	}
	return g.Match(input)
}

// LW TODO(gsagula): comment
type LW struct {
	resource dynamic.NamespaceableResourceInterface
}

// List TODO(gsagula): comment
func (lw LW) List(options v1.ListOptions) (runtime.Object, error) {
	return lw.resource.List(options)
}

// Watch TODO(gsagula): comment
func (lw LW) Watch(options v1.ListOptions) (watch.Interface, error) {
	return lw.resource.Watch(options)
}

// Controller TODO(gsagula): comment
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
		5*time.Minute,
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
