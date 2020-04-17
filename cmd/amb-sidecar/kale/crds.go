package kale

import (
	// standard library
	"encoding/json"
	"fmt"
	"time"

	// 3rd party
	"github.com/gogo/protobuf/proto"
	libgitPlumbing "gopkg.in/src-d/go-git.v4/plumbing"

	// k8s types
	aproTypesV2 "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Project struct {
	k8sTypesMetaV1.TypeMeta
	k8sTypesMetaV1.ObjectMeta `json:"metadata"`
	Spec                      ProjectSpec   `json:"spec"`
	Status                    ProjectStatus `json:"status"`
}

type ProjectSpec struct {
	Host        string `json:"host"`
	Prefix      string `json:"prefix"`
	GithubRepo  string `json:"githubRepo"`
	GithubToken string `json:"githubToken"` // todo: make this a secret ref
}

type ProjectStatus struct {
	Phase       ProjectPhase `json:"phase"`
	LastWebhook time.Time    `json:"lastWebhook"`
}

type ProjectPhase int32

const (
	ProjectPhase_Received         ProjectPhase = 0
	ProjectPhase_WebhookCreated   ProjectPhase = 1
	ProjectPhase_WebhookConfirmed ProjectPhase = 2
)

var ProjectPhase_name = map[int32]string{
	0: "Received",
	1: "WebhookCreated",
	2: "WebhookConfirmed",
}

var ProjectPhase_value = map[string]int32{
	"Received":         0,
	"WebhookCreated":   1,
	"WebhookConfirmed": 2,
}

func (x ProjectPhase) String() string {
	return proto.EnumName(ProjectPhase_name, int32(x))
}

func (x ProjectPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *ProjectPhase) UnmarshalJSON(bs []byte) error {
	var str string
	if err := json.Unmarshal(bs, &str); err != nil {
		return err
	}
	val, ok := ProjectPhase_value[str]
	if !ok {
		// non-fatal, for now?
		val = 0
	}
	*x = ProjectPhase(val)
	return nil
}

func (p Project) Key() string {
	return p.GetNamespace() + "/" + p.GetName()
}

const CODE = "butterscotch"

func (p Project) PreviewUrl(revision *ProjectRevision) string {
	return fmt.Sprintf("https://%s/.previews%s%s/", p.Spec.Host, p.Spec.Prefix, revision.Spec.Rev)
}

func (p Project) ServerLogUrl(revision *ProjectRevision) string {
	return fmt.Sprintf("https://%s/edge_stack/admin/#projects?code=%s&log=deploy/%s.%s",
		p.Spec.Host, CODE, revision.GetName(), revision.GetNamespace())
}

func (p Project) BuildLogUrl(revision *ProjectRevision) string {
	return fmt.Sprintf("https://%s/edge_stack/admin/#projects?code=%s&log=build/%s.%s",
		p.Spec.Host, CODE, revision.GetName(), revision.GetNamespace())

}

type ProjectRevision struct {
	k8sTypesMetaV1.TypeMeta
	k8sTypesMetaV1.ObjectMeta `json:"metadata"`
	Spec                      ProjectRevisionSpec   `json:"spec"`
	Status                    ProjectRevisionStatus `json:"status"`
}

type ProjectRevisionSpec struct {
	Project   k8sTypesCoreV1.LocalObjectReference `json:"project"`
	Ref       libgitPlumbing.ReferenceName        `json:"ref"` // string
	Rev       string                              `json:"rev"` // libgitPlumbing.Hash
	IsPreview bool                                `json:"isPreview"`
}

type ProjectRevisionStatus struct {
	Phase RevisionPhase `json:"phase"`
}

type RevisionPhase int32

const (
	RevisionPhase_Received     RevisionPhase = 0
	RevisionPhase_BuildQueued  RevisionPhase = 1
	RevisionPhase_Building     RevisionPhase = 2
	RevisionPhase_BuildFailed  RevisionPhase = 3
	RevisionPhase_Deploying    RevisionPhase = 4
	RevisionPhase_DeployFailed RevisionPhase = 5
	RevisionPhase_Deployed     RevisionPhase = 6
)

var RevisionPhase_name = map[int32]string{
	0: "Received",
	1: "BuildQueued",
	2: "Building",
	3: "BuildFailed",
	4: "Deploying",
	5: "DeployFailed",
	6: "Deployed",
}

var RevisionPhase_value = map[string]int32{
	"Received":     0,
	"BuildQueued":  1,
	"Building":     2,
	"BuildFailed":  3,
	"Deploying":    4,
	"DeployFailed": 5,
	"Deployed":     6,
}

func (x RevisionPhase) String() string {
	return proto.EnumName(RevisionPhase_name, int32(x))
}

func (x RevisionPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *RevisionPhase) UnmarshalJSON(bs []byte) error {
	var str string
	if err := json.Unmarshal(bs, &str); err != nil {
		return err
	}
	val, ok := RevisionPhase_value[str]
	if !ok {
		// non-fatal, for now?
		val = 0
	}
	*x = RevisionPhase(val)
	return nil
}

type Mapping struct {
	k8sTypesMetaV1.TypeMeta
	k8sTypesMetaV1.ObjectMeta `json:"metadata"`
	Spec                      MappingSpec `json:"spec"`
}

type MappingSpec struct {
	AmbassadorID aproTypesV2.AmbassadorID `json:"ambassador_id"`

	Prefix  string `json:"prefix"`
	Service string `json:"service"`
}

type ProjectController struct {
	k8sTypesMetaV1.TypeMeta
	k8sTypesMetaV1.ObjectMeta `json:"metadata"`
	Spec                      ProjectControllerSpec   `json:"spec"`
	Status                    ProjectControllerStatus `json:"status"`
}

type ProjectControllerSpec struct {
	MaximumConcurrentBuilds *int
}

func (pc *ProjectController) GetMaximumConcurrentBuilds() int {
	if pc != nil && pc.Spec.MaximumConcurrentBuilds != nil {
		return *pc.Spec.MaximumConcurrentBuilds
	}
	// default
	return 3
}

type ProjectControllerStatus struct{}
