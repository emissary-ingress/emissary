package kale

import (
	// standard library
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	// 3rd party: k8s types
	k8sTypesAppsV1 "k8s.io/api/apps/v1"
	k8sTypesBatchV1 "k8s.io/api/batch/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"

	// 1st party
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/apro/lib/mapstructure"
)

type UntypedSnapshot struct {
	Controllers []k8s.Resource
	Projects    []k8s.Resource
	Commits     []k8s.Resource
	Jobs        []k8s.Resource
	Deployments []k8s.Resource
}

type Snapshot struct {
	Controllers map[k8sTypes.UID]*controllerAndChildren
	Projects    map[k8sTypes.UID]*projectAndChildren
	Commits     map[k8sTypes.UID]*commitAndChildren
	Jobs        map[k8sTypes.UID]*k8sTypesBatchV1.Job
	Deployments map[k8sTypes.UID]*k8sTypesAppsV1.Deployment
	Errors      []recordedError

	grouped *GroupedSnapshot `json:"-"`
}

func (in UntypedSnapshot) TypedAndIndexed(ctx context.Context) *Snapshot {
	var out Snapshot

	// For built-in resource types, it is appropriate to a panic
	// because it's a bug; the api-server only gives us valid
	// resources, so if we fail to parse them, it's a bug in how
	// we're parsing.  For the same reason, it's "safe" to do this
	// all at once, because we don't need to do individual
	// validation, because they're all valid.

	// jobs
	var outJobs []*k8sTypesBatchV1.Job
	if err := mapstructure.Convert(in.Jobs, &outJobs); err != nil {
		panicThisIsABug(fmt.Errorf("Jobs: %w", err))
	}
	out.Jobs = make(map[k8sTypes.UID]*k8sTypesBatchV1.Job, len(outJobs))
	for _, outJob := range outJobs {
		out.Jobs[outJob.GetUID()] = outJob
	}

	// deployments
	var outDeployments []*k8sTypesAppsV1.Deployment
	if err := mapstructure.Convert(in.Deployments, &outDeployments); err != nil {
		panicThisIsABug(fmt.Errorf("Deployments: %w", err))
	}
	out.Deployments = make(map[k8sTypes.UID]*k8sTypesAppsV1.Deployment, len(outDeployments))
	for _, outDeployment := range outDeployments {
		out.Deployments[outDeployment.GetUID()] = outDeployment
	}

	// However, for our CRDs, because the api-server can't
	// validate that CRs are valid the way that it can for
	// built-in Resources, we have to safely deal with the
	// possibility that any individual resource is invalid, and
	// not let that affect the others.

	// projects
	out.Projects = make(map[k8sTypes.UID]*projectAndChildren, len(in.Projects))
	for _, inProj := range in.Projects {
		var outProj *Project
		if err := mapstructure.Convert(inProj, &outProj); err != nil {
			reportRuntimeError(ctx, StepValidProject,
				fmt.Errorf("Project: %w", err))
			continue
		}
		out.Projects[outProj.GetUID()] = &projectAndChildren{Project: outProj}
	}

	// commits
	out.Commits = make(map[k8sTypes.UID]*commitAndChildren, len(in.Commits))
	for _, inCommit := range in.Commits {
		var outCommit *ProjectCommit
		if err := mapstructure.Convert(inCommit, &outCommit); err != nil {
			reportThisIsABug(ctx, fmt.Errorf("ProjectCommit: %w", err))
			continue
		}
		out.Commits[outCommit.GetUID()] = &commitAndChildren{ProjectCommit: outCommit}
	}

	// controllers
	out.Controllers = make(map[k8sTypes.UID]*controllerAndChildren, len(in.Controllers))
	for _, inController := range in.Controllers {
		var outController *ProjectController
		if err := mapstructure.Convert(inController, &outController); err != nil {
			reportThisIsABug(ctx, fmt.Errorf("ProjectController: %w", err))
			continue
		}
		out.Controllers[outController.GetUID()] = &controllerAndChildren{ProjectController: outController}
	}

	return &out
}

// All lists are sorted by UID (which essentially means ordering is
// arbitrary but stable).
type GroupedSnapshot struct {
	Controllers []*controllerAndChildren

	OrphanedCommits     []*commitAndChildren
	OrphanedJobs        []*k8sTypesBatchV1.Job
	OrphanedDeployments []*k8sTypesAppsV1.Deployment
	OrphanedErrors      []recordedError
}

type recordedError struct {
	Time       time.Time    `json:"time"`
	Message    string       `json:"message"`
	ProjectUID k8sTypes.UID `json:"project_uid,omitempty"`
	CommitUID  k8sTypes.UID `json:"commit_uid,omitempty"`
}

type controllerAndChildren struct {
	*ProjectController
	Children struct {
		Projects []*projectAndChildren `json:"projects"`
		Errors   []recordedError       `json:"errors"`
	} `json:"children"`
}

type projectAndChildren struct {
	*Project
	Parent   *controllerAndChildren `json:"-"`
	Children struct {
		Commits []*commitAndChildren `json:"commits"`
		Errors  []recordedError      `json:"errors"`
	} `json:"children"`
}

type commitAndChildren struct {
	*ProjectCommit
	Parent   *projectAndChildren `json:"-"`
	Children struct {
		Builders []*k8sTypesBatchV1.Job       `json:"builders"`
		Runners  []*k8sTypesAppsV1.Deployment `json:"runners"`
		Errors   []recordedError              `json:"errors"`
	} `json:"children"`
}

// Grouped (1) mutates the Snapshot such that the .Children and
// .Parent members are populated, and (2) returns a top-level
// GroupedSnapshot, that has pointers to the items in the original
// Snapshot.
func (in *Snapshot) Grouped() *GroupedSnapshot {
	if in.grouped != nil {
		return in.grouped
	}
	var out GroupedSnapshot

	for _, controllerUID := range sortedUIDKeys(in.Controllers) {
		out.Controllers = append(out.Controllers, in.Controllers[controllerUID])
	}
	if len(out.Controllers) == 0 {
		out.Controllers = append(out.Controllers, &controllerAndChildren{})
	}
	for _, projUID := range sortedUIDKeys(in.Projects) {
		controller := out.Controllers[0]
		proj := in.Projects[projUID]
		proj.Parent = controller
		controller.Children.Projects = append(controller.Children.Projects, proj)
	}
	for _, commitUID := range sortedUIDKeys(in.Commits) {
		commit := in.Commits[commitUID]
		projUID := k8sTypes.UID(commit.GetLabels()[ProjectLabelName])
		if proj, ok := in.Projects[projUID]; ok {
			commit.Parent = proj
			proj.Children.Commits = append(proj.Children.Commits, commit)
		} else {
			out.OrphanedCommits = append(out.OrphanedCommits, commit)
		}
	}
	for _, jobUID := range sortedUIDKeys(in.Jobs) {
		job := in.Jobs[jobUID]
		commitUID := k8sTypes.UID(job.GetLabels()[CommitLabelName])
		if commit, ok := in.Commits[commitUID]; ok {
			commit.Children.Builders = append(commit.Children.Builders, job)
		} else {
			out.OrphanedJobs = append(out.OrphanedJobs, job)
		}
	}
	for _, deploymentUID := range sortedUIDKeys(in.Deployments) {
		deployment := in.Deployments[deploymentUID]
		commitUID := k8sTypes.UID(deployment.GetLabels()[CommitLabelName])
		if commit, ok := in.Commits[commitUID]; ok {
			commit.Children.Runners = append(commit.Children.Runners, deployment)
		} else {
			out.OrphanedDeployments = append(out.OrphanedDeployments, deployment)
		}
	}
	sortErrors(in.Errors)
	for _, err := range in.Errors {
		switch {
		case err.CommitUID != "":
			if commit, ok := in.Commits[err.CommitUID]; ok {
				commit.Children.Errors = append(commit.Children.Errors, err)
			} else {
				out.OrphanedErrors = append(out.OrphanedErrors, err)
			}
		case err.ProjectUID != "":
			if project, ok := in.Projects[err.ProjectUID]; ok {
				project.Children.Errors = append(project.Children.Errors, err)
			} else {
				out.OrphanedErrors = append(out.OrphanedErrors, err)
			}
		default:
			controller := out.Controllers[0]
			controller.Children.Errors = append(controller.Children.Errors, err)
		}
	}

	in.grouped = &out
	return in.grouped
}

// sortedUIDKeys takes a map[k8sTypes.UID]ANYTHING, and returns a
// sorted list of the keys.  It is invalid to call it (it will panic)
// when the input is not a map or the key of the map isn't a
// k8sTypes.UID.
func sortedUIDKeys(m interface{}) []k8sTypes.UID {
	value := reflect.ValueOf(m)

	out := make([]k8sTypes.UID, 0, value.Len())

	iter := value.MapRange()
	for iter.Next() {
		out = append(out, iter.Key().Interface().(k8sTypes.UID))
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return out
}

func sortErrors(errs []recordedError) {
	sort.Slice(errs, func(i, j int) bool {
		return errs[i].Time.Before(errs[j].Time)
	})
}
