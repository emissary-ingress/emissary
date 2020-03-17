package kale

import (
	// standard library
	"fmt"
	"time"

	// k8s types
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	aproTypesV2 "github.com/datawire/apro/apis/getambassador.io/v1beta2"
)

type Project struct {
	Metadata k8sTypesMetaV1.ObjectMeta `json:"metadata"`
	Spec     struct {
		Host        string `json:"host"`
		Prefix      string `json:"prefix"`
		GithubRepo  string `json:"githubRepo"`
		GithubToken string `json:"githubToken"` // todo: make this a secret ref
	} `json:"spec"`
	Status struct {
		LastPush time.Time `json:"lastPush"`
	} `json:"status"`
}

func (p Project) Key() string {
	return p.Metadata.Namespace + "/" + p.Metadata.Name
}

func (p Project) PreviewUrl(commit string) string {
	return fmt.Sprintf("https://%s/.previews/%s/%s/", p.Spec.Host, p.Spec.Prefix, commit)
}

func (p Project) ServerLogUrl(commit string) string {
	return fmt.Sprintf("https://%s/edge_stack/admin/#projects?log=deploy/%s/%s/%s", p.Spec.Host,
		p.Metadata.Namespace, p.Metadata.Name, commit)
}

func (p Project) BuildLogUrl(commit string) string {
	return fmt.Sprintf("https://%s/edge_stack/admin/#projects?log=build/%s/%s/%s", p.Spec.Host,
		p.Metadata.Namespace, p.Metadata.Name, commit)
}

type Mapping struct {
	k8sTypesMetaV1.TypeMeta   `json:",inline"`
	k8sTypesMetaV1.ObjectMeta `json:"metadata"`
	Spec                      MappingSpec `json:"spec"`
}

type MappingSpec struct {
	AmbassadorID aproTypesV2.AmbassadorID `json:"ambassador_id"`

	Prefix  string `json:"prefix"`
	Service string `json:"service"`
}
