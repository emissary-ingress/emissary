package entrypoint

import (
	"fmt"
	"reflect"
	"strings"

	amb "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/pkg/kates"
)

type AmbassadorInputs struct {
	CRDs []*kates.CustomResourceDefinition

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

	// resolvers
	ConsulResolvers             []*amb.ConsulResolver             `json:"ConsulResolver"`
	KubernetesEndpointResolvers []*amb.KubernetesEndpointResolver `json:"KubernetesEndpointResolver"`
	KubernetesServiceResolvers  []*amb.KubernetesServiceResolver  `json:"KubernetesServiceResolver"`

	// missing:
	// AmbassadorInstallation
	// KNative ClusterIngress
	// KNative Ingress

	AllSecrets        []*kates.Secret     `json:"-"`
	referencedSecrets map[string]struct{} `json:"-"`
	Secrets           []*kates.Secret     `json:"secret"`
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
