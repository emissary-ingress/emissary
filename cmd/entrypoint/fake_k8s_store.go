package entrypoint

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"sync"

	"github.com/datawire/ambassador/v2/pkg/kates"
)

// A K8sStore is implement just enough data structures to mock the watch aspect of kubernetes for
// testing purposes. It holds a map of kubernetes resources. Whenever any of these resources change
// it computes a delta and adds it to the list of deltas. The store is also capable of creating
// cursors that can be used to track multiple watches independently consuming the deltas at
// different rates.
type K8sStore struct {
	// The mutex protects the entire struct, including any cursors that may have been created.
	mutex     sync.Mutex
	resources map[K8sKey]kates.Object
	// This tracks every delta forever. That's ok because we only use this for tests, so we want to
	// favor simplicity over efficiency. Also tests don't run that long, so it's not a big deal.
	deltas []*kates.Delta
}

type K8sKey struct {
	Kind      string
	Namespace string
	Name      string
}

func (k K8sKey) sortKey() string {
	return fmt.Sprintf("%s:%s:%s", k.Kind, k.Namespace, k.Name)
}

// NewK8sStore creates a new and empty store.
func NewK8sStore() *K8sStore {
	return &K8sStore{resources: map[K8sKey]kates.Object{}}
}

// Upsert will either update or insert the given object depending on whether or not an object with
// that key already exists. Note that this is currently done based solely on the key (namespace,
// name) of the resource. Theoretically resources are assigned UUIDs and so in theory we could
// detect changes to the name and namespace, however I'm not even sure how kubernetes handles this
// or if it even permits that, so I am not going to attempt to consider those cases, and that may
// well result in some very obscure edgecases around changing names/namespaces that behave
// differently different from kubernetes.
func (k *K8sStore) Upsert(resource kates.Object) {
	var un *kates.Unstructured
	bytes, err := json.Marshal(resource)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(bytes, &un)
	if err != nil {
		panic(err)
	}

	kind, apiVersion := canonGVK(un.GetKind())
	un.SetKind(kind)
	un.SetAPIVersion(apiVersion)
	if un.GetNamespace() == "" {
		un.SetNamespace("default")
	}

	k.mutex.Lock()
	defer k.mutex.Unlock()

	key := K8sKey{un.GetKind(), un.GetNamespace(), un.GetName()}
	_, ok := k.resources[key]
	if ok {
		k.deltas = append(k.deltas, kates.NewDelta(kates.ObjectUpdate, un))
	} else {
		k.deltas = append(k.deltas, kates.NewDelta(kates.ObjectAdd, un))
	}
	k.resources[key] = un
}

// Delete will remove the identified resource from the store.
func (k *K8sStore) Delete(kind, namespace, name string) {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	key := K8sKey{canon(kind), namespace, name}
	old, ok := k.resources[key]
	if ok {
		k.deltas = append(k.deltas, kates.NewDeltaFromObject(kates.ObjectDelete, old))
	}
	delete(k.resources, key)
}

// UpsertFile will parse the yaml manifests in the referenced file and Upsert each resource from the
// file.
func (k *K8sStore) UpsertFile(filename string) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	k.UpsertYAML(string(content))
}

// UpsertYAML will parse the provided YAML and feed the resources in it into the control plane,
// creating or updating any overlapping resources that exist.
func (k *K8sStore) UpsertYAML(yaml string) {
	objs, err := kates.ParseManifests(yaml)
	if err != nil {
		panic(err)
	}

	for _, obj := range objs {
		k.Upsert(obj)
	}
}

// A Cursor allows multiple views of the same stream of deltas. The cursors implement a bootstrap
// semantic where they will generate synthetic Add deltas for every resource that currently exists,
// and from that point on report the real deltas that actually occur on the store.
func (k *K8sStore) Cursor() *K8sStoreCursor {
	k.mutex.Lock()
	defer k.mutex.Unlock()
	return &K8sStoreCursor{store: k, offset: -1}
}

type K8sStoreCursor struct {
	store *K8sStore
	// Offset into the deltas slice, or negative one if the cursor is brand new.
	offset int
}

// Get returns a map of resources plus all the deltas that lead to the map being in its current
// state.
func (kc *K8sStoreCursor) Get() (map[K8sKey]kates.Object, []*kates.Delta) {
	kc.store.mutex.Lock()
	defer kc.store.mutex.Unlock()

	var deltas []*kates.Delta

	resources := map[K8sKey]kates.Object{}
	for _, key := range sortedKeys(kc.store.resources) {
		resource := kc.store.resources[key]
		resources[key] = resource
		// This is the first time Get() has been called, so we shall create a synthetic ADD delta
		// for every resource that currently exists.
		if kc.offset < 0 {
			deltas = append(deltas, kates.NewDeltaFromObject(kates.ObjectAdd, resource))
		}
	}

	if kc.offset >= 0 {
		deltas = append(deltas, kc.store.deltas[kc.offset:len(kc.store.deltas)]...)
	}
	kc.offset = len(kc.store.deltas)

	return resources, deltas
}

func sortedKeys(resources map[K8sKey]kates.Object) []K8sKey {
	var keys []K8sKey
	for k := range resources {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].sortKey() < keys[j].sortKey()
	})

	return keys
}

func canonGVK(kind string) (canonKind string, canonGroupVersion string) {
	// XXX: there is probably a better way to do this, but this is good enough for now, we just need
	// this to work well for ambassador and core types.

	switch strings.ToLower(kind) {
	case "service":
		fallthrough
	case "services":
		fallthrough
	case "services.":
		return "Service", "v1"
	case "secret":
		fallthrough
	case "secrets":
		fallthrough
	case "secrets.":
		return "Secret", "v1"
	case "endpoints":
		fallthrough
	case "endpoints.":
		return "Endpoints", "v1"
	case "ingress":
		fallthrough
	case "ingresses":
		fallthrough
	case "ingresses.extensions":
		return "Ingress", "v1"
	case "ingressclass":
		fallthrough
	case "ingressclasses":
		fallthrough
	case "ingressclasses.networking.k8s.io":
		return "IngressClass", "v1"
	case "authservice":
		fallthrough
	case "authservices":
		fallthrough
	case "authservices.getambassador.io":
		return "AuthService", "getambassador.io/v2"
	case "consulresolver":
		fallthrough
	case "consulresolvers":
		fallthrough
	case "consulresolvers.getambassador.io":
		return "ConsulResolver", "getambassador.io/v2"
	case "devportal":
		fallthrough
	case "devportals":
		fallthrough
	case "devportals.getambassador.io":
		return "DevPortal", "getambassador.io/v2"
	case "ambassadorhost":
		fallthrough
	case "ambassadorhosts":
		fallthrough
	case "ambassadorhosts.x.getambassador.io":
		return "AmbassadorHost", "x.getambassador.io/v3alpha1"
	case "kubernetesendpointresolver":
		fallthrough
	case "kubernetesendpointresolvers":
		fallthrough
	case "kubernetesendpointresolvers.getambassador.io":
		return "KubernetesEndpointResolver", "getambassador.io/v2"
	case "kubernetesserviceresolver":
		fallthrough
	case "kubernetesserviceresolvers":
		fallthrough
	case "kubernetesserviceresolvers.getambassador.io":
		return "KubernetesServiceResolver", "getambassador.io/v2"
	case "ambassadorlistener":
		fallthrough
	case "ambassadorlisteners":
		fallthrough
	case "ambassadorlisteners.x.getambassador.io":
		return "AmbassadorListener", "x.getambassador.io/v3alpha1"
	case "logservice":
		fallthrough
	case "logservices":
		fallthrough
	case "logservices.getambassador.io":
		return "LogService", "getambassador.io/v2"
	case "ambassadormapping":
		fallthrough
	case "ambassadormappings":
		fallthrough
	case "ambassadormappings.x.getambassador.io":
		return "AmbassadorMapping", "x.getambassador.io/v3alpha1"
	case "module":
		fallthrough
	case "modules":
		fallthrough
	case "modules.getambassador.io":
		return "Module", "getambassador.io/v2"
	case "ratelimitservice":
		fallthrough
	case "ratelimitservices":
		fallthrough
	case "ratelimitservices.getambassador.io":
		return "RateLimitServices", "getambassador.io/v2"
	case "ambassadortcpmapping":
		fallthrough
	case "ambassadortcpmappings":
		fallthrough
	case "ambassadortcpmappings.x.getambassador.io":
		return "AmbassadorTCPMapping", "x.getambassador.io/v3alpha1"
	case "tlscontext":
		fallthrough
	case "tlscontexts":
		fallthrough
	case "tlscontexts.getambassador.io":
		return "TLSContext", "getambassador.io/v2"
	case "tracingservice":
		fallthrough
	case "tracingservices":
		fallthrough
	case "tracingservices.getambassador.io":
		return "TracingService", "getambassador.io/v2"
	case "gatewayclasses.networking.x-k8s.io":
		return "GatewayClass", "networking.x-k8s.io/v1alpha1"
	case "gateways.networking.x-k8s.io":
		return "Gateway", "networking.x-k8s.io/v1alpha1"
	case "httproutes.networking.x-k8s.io":
		return "HTTPRoute", "networking.x-k8s.io/v1alpha1"
	default:
		panic(fmt.Errorf("I don't know how to canonicalize kind: %q", kind))
	}
}

func canon(kind string) string {
	canonKind, _ := canonGVK(kind)
	return canonKind
}
