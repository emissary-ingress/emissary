package snapshot

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	amb "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	ambv3alpha1 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
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
	// The Invalid field contains any kubernetes resources that have failed
	// validation.
	Invalid []*kates.Unstructured
	Raw     json.RawMessage `json:"-"`
}

type AmbassadorMetaInfo struct {
	ClusterID         string `json:"cluster_id"`
	AmbassadorID      string `json:"ambassador_id"`
	AmbassadorVersion string `json:"ambassador_version"`
	KubeVersion       string `json:"kube_version"`
}

type KubernetesSnapshot struct {
	// k8s resources
	IngressClasses []*kates.IngressClass `json:"ingressclasses"`
	Ingresses      []*kates.Ingress      `json:"ingresses"`
	Services       []*kates.Service      `json:"service"`
	Endpoints      []*kates.Endpoints    `json:"Endpoints"`

	// ambassador resources
	Listeners   []*ambv3alpha1.AmbassadorListener `json:"Listener"`
	Hosts       []*ambv3alpha1.Host               `json:"Host"`
	Mappings    []*ambv3alpha1.Mapping            `json:"Mapping"`
	TCPMappings []*ambv3alpha1.TCPMapping         `json:"TCPMapping"`
	Modules     []*amb.Module                     `json:"Module"`
	TLSContexts []*amb.TLSContext                 `json:"TLSContext"`

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

	Annotations []kates.Object `json:"-"`

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

func (a *KubernetesSnapshot) Render() string {
	result := &strings.Builder{}
	v := reflect.ValueOf(a)
	t := v.Type().Elem()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		result.WriteString(fmt.Sprintf("%s: %d\n", f.Name, reflect.Indirect(v).Field(i).Len()))
	}
	return result.String()
}
