package kale

import (
	// standard library
	"context"
	"fmt"
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
	Projects    []k8s.Resource
	Commits     []k8s.Resource
	Jobs        []k8s.Resource
	Deployments []k8s.Resource
}

type Snapshot struct {
	Projects    map[k8sTypes.UID]*Project
	Commits     map[k8sTypes.UID]*ProjectCommit
	Jobs        map[k8sTypes.UID]*k8sTypesBatchV1.Job
	Deployments map[k8sTypes.UID]*k8sTypesAppsV1.Deployment
}

func (in UntypedSnapshot) TypedAndIndexed(ctx context.Context) Snapshot {
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
	out.Projects = make(map[k8sTypes.UID]*Project, len(in.Projects))
	for _, inProj := range in.Projects {
		var outProj *Project
		if err := mapstructure.Convert(inProj, &outProj); err != nil {
			reportRuntimeError(ctx, StepValidProject,
				fmt.Errorf("Project: %w", err))
			continue
		}
		out.Projects[outProj.GetUID()] = outProj
	}

	// commits
	out.Commits = make(map[k8sTypes.UID]*ProjectCommit, len(in.Commits))
	for _, inCommit := range in.Commits {
		var outCommit *ProjectCommit
		if err := mapstructure.Convert(inCommit, &outCommit); err != nil {
			reportThisIsABug(ctx, fmt.Errorf("Commit: %w", err))
			continue
		}
		out.Commits[outCommit.GetUID()] = outCommit
	}

	return out
}

type recordedError struct {
	Time       time.Time    `json:"time"`
	Message    string       `json:"message"`
	ProjectUID k8sTypes.UID `json:"project_uid,omitempty"`
	CommitUID  k8sTypes.UID `json:"commit_uid,omitempty"`
}

type projectAndChildren struct {
	*Project
	Children struct {
		Commits []*commitAndChildren `json:"commits"`
		Errors  []recordedError      `json:"errors"`
	} `json:"children"`
}

type commitAndChildren struct {
	*ProjectCommit
	Children struct {
		Builders []*k8sTypesBatchV1.Job       `json:"builders"`
		Runners  []*k8sTypesAppsV1.Deployment `json:"runners"`
		Errors   []recordedError              `json:"errors"`
	} `json:"children"`
}

func (snapshot Snapshot) Grouped(ctx context.Context) map[k8sTypes.UID]*projectAndChildren {
	// map[commitUID]*commitAndChildren
	commits := make(map[k8sTypes.UID]*commitAndChildren)
	for _, commit := range snapshot.Commits {
		key := commit.GetUID()
		if _, ok := commits[key]; !ok {
			commits[key] = new(commitAndChildren)
		}
		commits[key].ProjectCommit = commit
	}
	for _, job := range snapshot.Jobs {
		key := k8sTypes.UID(job.GetLabels()[CommitLabelName])
		if _, ok := commits[key]; !ok {
			reportRuntimeError(ctx, StepBackground,
				fmt.Errorf("unable to pair Job %q.%q with ProjectCommit; ignoring",
					job.GetName(), job.GetNamespace()))
			continue
		}
		commits[key].Children.Builders = append(commits[key].Children.Builders, job)
	}
	for _, deployment := range snapshot.Deployments {
		key := k8sTypes.UID(deployment.GetLabels()[CommitLabelName])
		if _, ok := commits[key]; !ok {
			reportRuntimeError(ctx, StepBackground,
				fmt.Errorf("unable to pair Deployment %q.%q with ProjectCommit; ignoring",
					deployment.GetName(), deployment.GetNamespace()))
			continue
		}
		commits[key].Children.Runners = append(commits[key].Children.Runners, deployment)
	}

	// map[projectUID]*projectAndChildren
	projects := make(map[k8sTypes.UID]*projectAndChildren)
	for _, proj := range snapshot.Projects {
		key := proj.GetUID()
		if _, ok := projects[key]; !ok {
			projects[key] = new(projectAndChildren)
		}
		projects[key].Project = proj
	}
	for _, commit := range commits {
		key := k8sTypes.UID(commit.GetLabels()[ProjectLabelName])
		if _, ok := projects[key]; !ok {
			reportRuntimeError(ctx, StepBackground,
				fmt.Errorf("unable to pair ProjectCommit %q.%q with Project; ignoring",
					commit.GetName(), commit.GetNamespace()))
			continue
		}
		projects[key].Children.Commits = append(projects[key].Children.Commits, commit)
	}

	return projects
}
