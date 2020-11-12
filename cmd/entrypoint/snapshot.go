package entrypoint

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
	"github.com/datawire/ambassador/pkg/watt"
)

// The snapshot type represents a complete configuration snapshot as sent to
// diagd.
type Snapshot struct {
	// The Kubernetes field contains all the ambassador inputs from kubernetes.
	Kubernetes *AmbassadorInputs
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
}

type AmbassadorInputs struct {
	// k8s resources
	IngressClasses []*kates.IngressClass `json:"ingressclasses"`
	Ingresses      []*kates.Ingress      `json:"ingresses"`
	Services       []*kates.Service      `json:"service"`
	Endpoints      []*kates.Endpoints    `json:"Endpoints"`

	// ambassador resources
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

	// It is safe to ignore AmbassadorInstallation, ambassador doesn't need to look at those, just
	// the operator.

	KNativeClusterIngresses []*kates.Unstructured `json:"clusteringresses.networking.internal.knative.dev,omitempty"`
	KNativeIngresses        []*kates.Unstructured `json:"ingresses.networking.internal.knative.dev,omitempty"`

	AllSecrets []*kates.Secret `json:"-"`
	Secrets    []*kates.Secret `json:"secret"`

	annotations []kates.Object `json:"-"`
}

func (a *AmbassadorInputs) Render() string {
	result := &strings.Builder{}
	v := reflect.ValueOf(a)
	t := v.Type().Elem()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		result.WriteString(fmt.Sprintf("%s: %d\n", f.Name, reflect.Indirect(v).Field(i).Len()))
	}
	return result.String()
}

// GetAmbId extracts the AmbassadorId from the kubernetes resource.
func GetAmbId(resource kates.Object) amb.AmbassadorID {
	switch r := resource.(type) {
	case *amb.Host:
		var id amb.AmbassadorID
		if r.Spec != nil {
			if len(r.Spec.AmbassadorID) > 0 {
				id = r.Spec.AmbassadorID
			} else {
				id = r.Spec.DeprecatedAmbassadorID
			}
		}
		return id

	case *amb.Mapping:
		return r.Spec.AmbassadorID
	case *amb.TCPMapping:
		return r.Spec.AmbassadorID
	case *amb.Module:
		return r.Spec.AmbassadorID
	case *amb.TLSContext:
		return r.Spec.AmbassadorID
	case *amb.AuthService:
		return r.Spec.AmbassadorID
	case *amb.RateLimitService:
		return r.Spec.AmbassadorID
	case *amb.LogService:
		return r.Spec.AmbassadorID
	case *amb.TracingService:
		return r.Spec.AmbassadorID
	case *amb.DevPortal:
		return r.Spec.AmbassadorID
	case *amb.ConsulResolver:
		return r.Spec.AmbassadorID
	case *amb.KubernetesEndpointResolver:
		return r.Spec.AmbassadorID
	case *amb.KubernetesServiceResolver:
		return r.Spec.AmbassadorID
	}

	ann := resource.GetAnnotations()
	idstr, ok := ann["getambassador.io/ambassador-id"]
	if ok {
		var id amb.AmbassadorID
		err := json.Unmarshal([]byte(idstr), &id)
		if err != nil {
			log.Printf("%s: error parsing ambassador-id '%s'", location(resource), idstr)
		} else {
			return id
		}
	}

	return amb.AmbassadorID{}
}

func location(obj kates.Object) string {
	return fmt.Sprintf("%s %s in namespace %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(),
		obj.GetNamespace())
}
