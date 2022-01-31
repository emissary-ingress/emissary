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
func (k *K8sStore) Upsert(resource kates.Object) error {
	var un *kates.Unstructured
	bytes, err := json.Marshal(resource)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, &un)
	if err != nil {
		return err
	}

	kind, apiVersion, err := canonGVK(un.GetKind())
	if err != nil {
		return err
	}
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
	return nil
}

// Delete will remove the identified resource from the store.
func (k *K8sStore) Delete(kind, namespace, name string) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	canonKind, err := canon(kind)
	if err != nil {
		return err
	}
	key := K8sKey{canonKind, namespace, name}
	old, ok := k.resources[key]
	if ok {
		delta, err := kates.NewDeltaFromObject(kates.ObjectDelete, old)
		if err != nil {
			return err
		}
		k.deltas = append(k.deltas, delta)
	}
	delete(k.resources, key)
	return nil
}

// UpsertFile will parse the yaml manifests in the referenced file and Upsert each resource from the
// file.
func (k *K8sStore) UpsertFile(filename string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	return k.UpsertYAML(string(content))
}

// UpsertYAML will parse the provided YAML and feed the resources in it into the control plane,
// creating or updating any overlapping resources that exist.
func (k *K8sStore) UpsertYAML(yaml string) error {
	objs, err := kates.ParseManifests(yaml)
	if err != nil {
		return err
	}

	for _, obj := range objs {
		if err := k.Upsert(obj); err != nil {
			return err
		}
	}
	return nil
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
func (kc *K8sStoreCursor) Get() (map[K8sKey]kates.Object, []*kates.Delta, error) {
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
			delta, err := kates.NewDeltaFromObject(kates.ObjectAdd, resource)
			if err != nil {
				return nil, nil, err
			}
			deltas = append(deltas, delta)
		}
	}

	if kc.offset >= 0 {
		deltas = append(deltas, kc.store.deltas[kc.offset:len(kc.store.deltas)]...)
	}
	kc.offset = len(kc.store.deltas)

	return resources, deltas, nil
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

func canonGVK(rawString string) (canonKind string, canonGroupVersion string, err error) {
	// XXX: there is probably a better way to do this, but this is good enough for now, we just need
	// this to work well for ambassador and core types.

	rawParts := strings.SplitN(rawString, ".", 2)
	var rawKind, rawVG string
	switch len(rawParts) {
	case 1:
		rawKind = rawParts[0]
	case 2:
		rawKind = rawParts[0]
		rawVG = rawParts[1]
	}

	// Each case should be `case "singular", "plural":`
	switch strings.ToLower(rawKind) {
	// Native Kubernetes types
	case "service", "services":
		return "Service", "v1", nil
	case "endpoints":
		return "Endpoints", "v1", nil
	case "secret", "secrets":
		return "Secret", "v1", nil
	case "ingress", "ingresses":
		if strings.HasSuffix(rawVG, ".knative.dev") {
			return "Ingress", "networking.internal.knative.dev/v1alpha1", nil
		}
		return "Ingress", "networking.k8s.io/v1", nil
	case "ingressclass", "ingressclasses":
		return "IngressClass", "networking.k8s.io/v1", nil
	// Gateway API
	case "gatewayclass", "gatewayclasses":
		return "GatewayClass", "networking.x-k8s.io/v1alpha1", nil
	case "gateway", "gateways":
		return "Gateway", "networking.x-k8s.io/v1alpha1", nil
	case "httproute", "httproutes":
		return "HTTPRoute", "networking.x-k8s.io/v1alpha1", nil
	// Knative types
	case "clusteringress", "clusteringresses":
		return "ClusterIngress", "networking.internal.knative.dev/v1alpha1", nil
	// Native Emissary types
	case "authservice", "authservices":
		return "AuthService", "getambassador.io/v3alpha1", nil
	case "consulresolver", "consulresolvers":
		return "ConsulResolver", "getambassador.io/v3alpha1", nil
	case "devportal", "devportals":
		return "DevPortal", "getambassador.io/v3alpha1", nil
	case "host", "hosts":
		return "Host", "getambassador.io/v3alpha1", nil
	case "kubernetesendpointresolver", "kubernetesendpointresolvers":
		return "KubernetesEndpointResolver", "getambassador.io/v3alpha1", nil
	case "kubernetesserviceresolver", "kubernetesserviceresolvers":
		return "KubernetesServiceResolver", "getambassador.io/v3alpha1", nil
	case "listener", "listeners":
		return "Listener", "getambassador.io/v3alpha1", nil
	case "logservice", "logservices":
		return "LogService", "getambassador.io/v3alpha1", nil
	case "mapping", "mappings":
		return "Mapping", "getambassador.io/v3alpha1", nil
	case "module", "modules":
		return "Module", "getambassador.io/v3alpha1", nil
	case "ratelimitservice", "ratelimitservices":
		return "RateLimitServices", "getambassador.io/v3alpha1", nil
	case "tcpmapping", "tcpmappings":
		return "TCPMapping", "getambassador.io/v3alpha1", nil
	case "tlscontext", "tlscontexts":
		return "TLSContext", "getambassador.io/v3alpha1", nil
	case "tracingservice", "tracingservices":
		return "TracingService", "getambassador.io/v3alpha1", nil
	default:
		return "", "", fmt.Errorf("I don't know how to canonicalize kind: %q", rawString)
	}
}

func canon(kind string) (string, error) {
	canonKind, _, err := canonGVK(kind)
	if err != nil {
		return "", err
	}
	return canonKind, nil
}
