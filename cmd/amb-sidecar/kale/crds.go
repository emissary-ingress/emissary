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
	Spec                      struct {
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
	return p.GetNamespace() + "/" + p.GetName()
}

func (p Project) PreviewUrl(commit *ProjectCommit) string {
	return fmt.Sprintf("https://%s/.previews/%s/%s/", p.Spec.Host, p.Spec.Prefix, commit.Spec.Rev)
}

func (p Project) ServerLogUrl(commit *ProjectCommit) string {
	return fmt.Sprintf("https://%s/edge_stack/admin/#projects?log=deploy/%s.%s",
		p.Spec.Host, commit.GetName(), commit.GetNamespace())
}

func (p Project) BuildLogUrl(commit *ProjectCommit) string {
	return fmt.Sprintf("https://%s/edge_stack/admin/#projects?log=build/%s.%s",
		p.Spec.Host, commit.GetName(), commit.GetNamespace())

}

type ProjectCommit struct {
	k8sTypesMetaV1.TypeMeta
	k8sTypesMetaV1.ObjectMeta `json:"metadata"`
	Spec                      ProjectCommitSpec   `json:"spec"`
	Status                    ProjectCommitStatus `json:"status"`
}

type ProjectCommitSpec struct {
	Project   k8sTypesCoreV1.LocalObjectReference `json:"project"`
	Ref       libgitPlumbing.ReferenceName        `json:"ref"` // string
	Rev       string                              `json:"rev"` // libgitPlumbing.Hash
	IsPreview bool                                `json:"isPreview"`
}

type ProjectCommitStatus struct {
	Phase CommitPhase `json:"phase"`
}

type CommitPhase int32

const (
	CommitPhase_Received     CommitPhase = 0
	CommitPhase_Building     CommitPhase = 1
	CommitPhase_BuildFailed  CommitPhase = 2
	CommitPhase_Deploying    CommitPhase = 3
	CommitPhase_DeployFailed CommitPhase = 4
	CommitPhase_Deployed     CommitPhase = 5
)

var CommitPhase_name = map[int32]string{
	0: "Received",
	1: "Building",
	2: "BuildFailed",
	3: "Deploying",
	4: "DeployFailed",
	5: "Deployed",
}

var CommitPhase_value = map[string]int32{
	"Received":     0,
	"Building":     1,
	"BuildFailed":  2,
	"Deploying":    3,
	"DeployFailed": 4,
	"Deployed":     5,
}

func (x CommitPhase) String() string {
	return proto.EnumName(CommitPhase_name, int32(x))
}

func (x CommitPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *CommitPhase) UnmarshalJSON(bs []byte) error {
	var str string
	if err := json.Unmarshal(bs, &str); err != nil {
		return err
	}
	val, ok := CommitPhase_value[str]
	if !ok {
		// non-fatal, for now?
		val = 0
	}
	*x = CommitPhase(val)
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
