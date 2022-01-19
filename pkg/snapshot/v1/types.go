package snapshot

import (
	"encoding/json"

	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/watt"
	gw "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

const ApiVersion = "v1"
const ContentTypeJSON = "application/json"

// SecretRef is a secret reference -- basically, a namespace/name pair.
type SecretRef struct {
	Namespace string
	Name      string
}

// The snapshot type represents a complete configuration snapshot as sent to
// diagd.
type Snapshot struct {
	// meta information to identify the ambassador
	AmbassadorMeta *AmbassadorMetaInfo
	// The Kubernetes field contains all the ambassador inputs from kubernetes.
	Kubernetes *KubernetesSnapshot
	// The Consul field contains endpoint data for any mappings setup to use a
	// consul resolver.
	Consul *watt.ConsulSnapshot
	// The Deltas field contains a list of deltas to indicate what has changed
	// since the prior snapshot. This is only computed for the Kubernetes
	// portion of the snapshot. Changes in the Consul endpoint data are not
	// reflected in this field.
	Deltas []*kates.Delta
	// The APIDocs field contains a list of OpenAPI documents scrapped from
	// Ambassador Mappings part of the KubernetesSnapshot
	APIDocs []*APIDoc `json:"APIDocs,omitempty"`
	// The Invalid field contains any kubernetes resources that have failed
	// validation.
	Invalid []*kates.Unstructured
	Raw     json.RawMessage `json:"-"`
}

type AmbassadorMetaInfo struct {
	ClusterID         string          `json:"cluster_id"`
	AmbassadorID      string          `json:"ambassador_id"`
	AmbassadorVersion string          `json:"ambassador_version"`
	KubeVersion       string          `json:"kube_version"`
	Sidecar           json.RawMessage `json:"sidecar"`
}

type KubernetesSnapshot struct {
	// k8s resources
	IngressClasses []*IngressClass    `json:"ingressclasses"`
	Ingresses      []*Ingress         `json:"ingresses"`
	Services       []*kates.Service   `json:"service"`
	Endpoints      []*kates.Endpoints `json:"Endpoints"`

	// ambassador resources
	Listeners   []*amb.Listener   `json:"Listener"`
	Hosts       []*amb.Host       `json:"Host"`
	Mappings    []*amb.Mapping    `json:"Mapping"`
	TCPMappings []*amb.TCPMapping `json:"TCPMapping"`
	Modules     []*amb.Module     `json:"Module"`
	TLSContexts []*amb.TLSContext `json:"TLSContext"`

	// plugin services
	AuthServices      []*amb.AuthService      `json:"AuthService"`
	RateLimitServices []*amb.RateLimitService `json:"RateLimitService"`
	LogServices       []*amb.LogService       `json:"LogService"`
	TracingServices   []*amb.TracingService   `json:"TracingService"`
	DevPortals        []*amb.DevPortal        `json:"DevPortal"`

	// resolvers
	ConsulResolvers             []*amb.ConsulResolver             `json:"ConsulResolver"`
	KubernetesEndpointResolvers []*amb.KubernetesEndpointResolver `json:"KubernetesEndpointResolver"`
	KubernetesServiceResolvers  []*amb.KubernetesServiceResolver  `json:"KubernetesServiceResolver"`

	// gateway api
	GatewayClasses []*gw.GatewayClass
	Gateways       []*gw.Gateway
	HTTPRoutes     []*gw.HTTPRoute

	// It is safe to ignore AmbassadorInstallation, ambassador doesn't need to look at those, just
	// the operator.

	KNativeClusterIngresses []*kates.Unstructured `json:"clusteringresses.networking.internal.knative.dev,omitempty"`
	KNativeIngresses        []*kates.Unstructured `json:"ingresses.networking.internal.knative.dev,omitempty"`

	K8sSecrets []*kates.Secret             `json:"-"`      // Secrets from Kubernetes
	FSSecrets  map[SecretRef]*kates.Secret `json:"-"`      // Secrets from the filesystem
	Secrets    []*kates.Secret             `json:"secret"` // Secrets we'll feed to Ambassador

	// [kind/name.namespace]AnnotationList
	Annotations map[string]AnnotationList `json:"annotations"`

	// Pods, Deployments and ConfigMaps were added to be used by Ambassador Agent so it can
	// report to AgentCom in Ambassador Cloud.
	Pods        []*kates.Pod        `json:"Pods,omitempty"`
	Deployments []*kates.Deployment `json:"Deployments,omitempty"`
	ConfigMaps  []*kates.ConfigMap  `json:"ConfigMaps,omitempty"`

	// ArgoRollouts represents the argo-rollout CRD state of the world that may or may not be present
	// in the client's cluster. For this reason, Rollouts resources are fetched making use of the
	// k8s dynamic client that returns an unstructured.Unstructured object. This is a better strategy
	// for Ambassador code base for the following reasons:
	//   - it is forward compatible
	//   - no need to maintain types defined by the Argo projects
	//   - no unnecessary overhead Marshaling/Unmarshaling it into json as the state is opaque to
	// Ambassador.
	ArgoRollouts []*kates.Unstructured `json:"ArgoRollouts,omitempty"`

	// ArgoApplications represents the argo-rollout CRD state of the world that may or may not be present
	// in the client's cluster. For reasons why this is defined as unstructured see ArgoRollouts attribute.
	ArgoApplications []*kates.Unstructured `json:"ArgoApplications,omitempty"`
}

type AnnotationList struct {
	Valid   []kates.Object        `json:"valid,omitempty"`
	Invalid []*kates.Unstructured `json:"invalid,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler, and exists because unmarshalling directly in to an
// interface (kates.Object) doesn't work.
func (al *AnnotationList) UnmarshalJSON(bs []byte) error {
	if string(bs) == "null" {
		*al = AnnotationList{}
		return nil
	}
	var ul struct {
		Valid   []*kates.Unstructured `json:"valid,omitempty"`
		Invalid []*kates.Unstructured `json:"invalid,omitempty"`
	}

	// Unmarshal as unstructured
	err := json.Unmarshal(bs, &ul)
	al.Invalid = ul.Invalid
	if err != nil {
		return err
	}

	// Convert the unstructured ul.Valid to the typed al.Valid.
	al.Valid = make([]kates.Object, 0, len(ul.Valid))
	for _, inObj := range ul.Valid {
		outObj, err := convertAnnotationObject(inObj)
		if err != nil {
			return err
		}
		al.Valid = append(al.Valid, outObj)
	}

	return nil
}

// The APIDoc type is custom object built in the style of a Kubernetes resource (name, type, version)
// which holds a reference to a Kubernetes object from which an OpenAPI document was scrapped (Data field)
type APIDoc struct {
	*kates.TypeMeta
	Metadata  *kates.ObjectMeta      `json:"metadata,omitempty"`
	TargetRef *kates.ObjectReference `json:"targetRef,omitempty"`
	Data      []byte                 `json:"data,omitempty"`
}
