package kale

import (
	// standard library
	"context"
	"fmt"
	"reflect"
	"sort"

	// 3rd party: k8s types
	k8sTypesAppsV1 "k8s.io/api/apps/v1"
	k8sTypesBatchV1 "k8s.io/api/batch/v1"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"

	// 3rd party: k8s misc
	k8sLabels "k8s.io/apimachinery/pkg/labels"

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
	Pods        []k8s.Resource
	Events      []k8s.Resource
}

type Snapshot struct {
	Controllers map[k8sTypes.UID]*controllerAndChildren
	Projects    map[k8sTypes.UID]*projectAndChildren
	Commits     map[k8sTypes.UID]*commitAndChildren
	Jobs        map[k8sTypes.UID]*jobAndChildren
	Deployments map[k8sTypes.UID]*deploymentAndChildren
	Pods        map[k8sTypes.UID]*podAndChildren
	Events      map[k8sTypes.UID]*k8sTypesCoreV1.Event

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
	out.Jobs = make(map[k8sTypes.UID]*jobAndChildren, len(outJobs))
	for _, outJob := range outJobs {
		out.Jobs[outJob.GetUID()] = &jobAndChildren{Job: outJob}
	}

	// deployments
	var outDeployments []*k8sTypesAppsV1.Deployment
	if err := mapstructure.Convert(in.Deployments, &outDeployments); err != nil {
		panicThisIsABug(fmt.Errorf("Deployments: %w", err))
	}
	out.Deployments = make(map[k8sTypes.UID]*deploymentAndChildren, len(outDeployments))
	for _, outDeployment := range outDeployments {
		out.Deployments[outDeployment.GetUID()] = &deploymentAndChildren{Deployment: outDeployment}
	}

	// pods
	var outPods []*k8sTypesCoreV1.Pod
	if err := mapstructure.Convert(in.Pods, &outPods); err != nil {
		panicThisIsABug(fmt.Errorf("Pods: %w", err))
	}
	out.Pods = make(map[k8sTypes.UID]*podAndChildren, len(outPods))
	for _, outPod := range outPods {
		out.Pods[outPod.GetUID()] = &podAndChildren{Pod: outPod}
	}

	// events
	var outEvents []*k8sTypesCoreV1.Event
	if err := mapstructure.Convert(in.Events, &outEvents); err != nil {
		panicThisIsABug(fmt.Errorf("Events: %w", err))
	}
	out.Events = make(map[k8sTypes.UID]*k8sTypesCoreV1.Event, len(outEvents))
	for _, outEvent := range outEvents {
		out.Events[outEvent.GetUID()] = outEvent
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
	OrphanedJobs        []*jobAndChildren
	OrphanedDeployments []*deploymentAndChildren
	OrphanedPods        []*podAndChildren
}

type controllerAndChildren struct {
	*ProjectController
	Children struct {
		Projects []*projectAndChildren   `json:"projects"`
		Errors   []*k8sTypesCoreV1.Event `json:"errors"`
	} `json:"children"`
}

type projectAndChildren struct {
	*Project
	Parent   *controllerAndChildren `json:"-"`
	Children struct {
		Commits []*commitAndChildren    `json:"commits"`
		Errors  []*k8sTypesCoreV1.Event `json:"errors"`
	} `json:"children"`
}

type commitAndChildren struct {
	*ProjectCommit
	Parent   *projectAndChildren `json:"-"`
	Children struct {
		Builders []*jobAndChildren        `json:"builders"`
		Runners  []*deploymentAndChildren `json:"runners"`
		Errors   []*k8sTypesCoreV1.Event  `json:"errors"`
	} `json:"children"`
}

type jobAndChildren struct {
	*k8sTypesBatchV1.Job
	Parent   *commitAndChildren `json:"-"`
	Children struct {
		Pods   []*jobPodAndChildren    `json:"pods"`
		Events []*k8sTypesCoreV1.Event `json:"events"`
	} `json:"children"`
}

type deploymentAndChildren struct {
	*k8sTypesAppsV1.Deployment
	Parent   *commitAndChildren `json:"-"`
	Children struct {
		Pods   []*deploymentPodAndChildren `json:"pods"`
		Events []*k8sTypesCoreV1.Event     `json:"events"`
	} `json:"children"`
}

type podAndChildren struct {
	*k8sTypesCoreV1.Pod
	Children struct {
		Events []*k8sTypesCoreV1.Event `json:"events"`
	} `json:"children"`
}

type jobPodAndChildren struct {
	*podAndChildren
	Parent *jobAndChildren `json:"-"`
}

type deploymentPodAndChildren struct {
	*podAndChildren
	Parent *deploymentAndChildren `json:"-"`
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
	for _, podUID := range sortedUIDKeys(in.Pods) {
		pod := in.Pods[podUID]
		if pod.GetLabels()[JobLabelName] != "" {
			for _, jobUID := range sortedUIDKeys(in.Jobs) {
				job := in.Jobs[jobUID]
				selector, err := k8sTypesMetaV1.LabelSelectorAsSelector(job.Spec.Selector)
				if err != nil {
					continue
				}
				if selector.Empty() {
					continue
				}
				if selector.Matches(k8sLabels.Set(pod.GetLabels())) {
					pod := &jobPodAndChildren{podAndChildren: pod}
					pod.Parent = job
					job.Children.Pods = append(job.Children.Pods, pod)
					break
				}
			}
		} else {
			for _, deploymentUID := range sortedUIDKeys(in.Deployments) {
				deployment := in.Deployments[deploymentUID]
				selector, err := k8sTypesMetaV1.LabelSelectorAsSelector(deployment.Spec.Selector)
				if err != nil {
					continue
				}
				if selector.Empty() {
					continue
				}
				if selector.Matches(k8sLabels.Set(pod.GetLabels())) {
					pod := &deploymentPodAndChildren{podAndChildren: pod}
					pod.Parent = deployment
					deployment.Children.Pods = append(deployment.Children.Pods, pod)
					break
				}
			}
		}
	}
	for _, eventUID := range sortedUIDKeys(in.Events) {
		event := in.Events[eventUID]
		if event.InvolvedObject.APIVersion == "getambassador.io/v2" && event.Type != k8sTypesCoreV1.EventTypeWarning {
			// The field is .Children.Errors, not .Children.Events
			continue
		}
		// Don't worry about orphaned Events--we expect a lot
		// of them, just drop them on the floor.
		switch event.InvolvedObject.Kind {
		case "ProjectController":
			if controller, ok := in.Controllers[event.InvolvedObject.UID]; ok {
				controller.Children.Errors = append(controller.Children.Errors, event)
			}
		case "Project":
			if project, ok := in.Projects[event.InvolvedObject.UID]; ok {
				project.Children.Errors = append(project.Children.Errors, event)
			}
		case "ProjectCommit":
			if commit, ok := in.Commits[event.InvolvedObject.UID]; ok {
				commit.Children.Errors = append(commit.Children.Errors, event)
			}
		case "Job":
			if job, ok := in.Jobs[event.InvolvedObject.UID]; ok {
				job.Children.Events = append(job.Children.Events, event)
			}
		case "Deployment":
			if deployment, ok := in.Deployments[event.InvolvedObject.UID]; ok {
				deployment.Children.Events = append(deployment.Children.Events, event)
			}
		case "Pod":
			if pod, ok := in.Pods[event.InvolvedObject.UID]; ok {
				pod.Children.Events = append(pod.Children.Events, event)
			}
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
