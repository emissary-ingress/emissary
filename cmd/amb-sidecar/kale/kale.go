package kale

import (
	// standard library
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	// 3rd party
	"github.com/google/uuid"
	libgitPlumbing "gopkg.in/src-d/go-git.v4/plumbing"

	// 3rd/1st party: k8s types
	aproTypesV2 "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	k8sTypesAppsV1 "k8s.io/api/apps/v1"
	k8sTypesBatchV1 "k8s.io/api/batch/v1"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"

	// 3rd party: k8s clients
	k8sClientDynamic "k8s.io/client-go/dynamic"

	// 3rd party: k8s misc
	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sValidation "k8s.io/apimachinery/pkg/util/validation"

	// 1st party
	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/metriton"
	"github.com/datawire/apro/cmd/amb-sidecar/group"
	"github.com/datawire/apro/cmd/amb-sidecar/k8s/leaderelection"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/mapstructure"
	aes_metriton "github.com/datawire/apro/lib/metriton"
	lyftserver "github.com/lyft/ratelimit/src/server"
)

// Future Exploration
//
// There are two primary areas of future work: customizability, and
// deeper github integration. Customizability would make this solution
// more appealing for use by larger teams with pre-existing CI. Deeper
// github integration would make this solution more appealing out of
// the box to smaller teams.
//
// Customizability: This is an all in one out of the box solution, but
// not very customizable. Future work could create plugin points to
// customize various pieces of what this controller does.
//
// Potential plugin points:
//
//  - provide a file in the github repo to customize aspects of deployment
//    + pod template(s), e.g. how to talk to database
//    + url prefix
//    + external build
//      - the in-cluster repo supports notifications, so an external
//        build could push images to the internal repo and we could
//        trigger the deploy process off the repo notification
//
// Deeper Github Integration: The Github API has a number of new rich
// capabilities, e.g. using the "checks" portion of the API, automated
// systems can supply PRs with feedback on individual lines of code,
// and provide actions for the user. These APIs are only available if
// you write a Github App. Doing so is more complicated than the
// github token techine this PoC uses. Github Apps can also be offered
// through the Github Marketplace, which could provide some
// interesting new onboarding options.
//
// It's likely we will need to expand this PoC incrementally into both
// areas based on user feedback.

// todo: security review
// todo: the watch and http handlers need a mutex
//       + a single giant mutex should do, can probably do it with a function transform
// todo: limit deploys to master + PRs
// todo: on success put url in a comment on the head commit of the PR
// todo: provide a proper UI
// todo: define schema/status for crd
// todo: experiment with minimal set of --tls-verify-blah arguments that kaniko needs
// todo: auto generate registry secret on install
// todo: this depends on panic.go which we copied out of apro, we
//       should put that into OSS, so we can depend on it without
//       pulling in all the rest of apro dependencies

// This program is both an http server *and* a kubernetes
// controller. The main function sets up listener watching for changes
// to kubernetes resources and a web server.

const (
	// We use human-unfriendly `UID`s instead of `name.namespace`s,
	// because label values are limited to 63 characters or less.
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
	GlobalLabelName  = "projects.getambassador.io/ambassador_id" // cfg.AmbassadorID
	ProjectLabelName = "projects.getambassador.io/project-uid"   // proj.GetUID()
	CommitLabelName  = "projects.getambassador.io/commit-uid"    // commit.GetUID()
	JobLabelName     = "job-name"                                // Don't change this; it's the label name used by Kubernetes itself
)

var globalKale *kale

const (
	StepSetup                      = "00-setup"
	StepLeader                     = "01-leader"
	StepValidProject               = "02-validproject"
	StepReconcileWebhook           = "03-reconcilewebook"
	StepReconcileProjectsToCommits = "04-reconcileprojectstocommits"
	StepReconcileCommitsToAction   = "05-reconcilecommitstoaction"
	//StepGitPull                    = "06-git-pull"
	//StepGitSanityCheck             = "07-git-sanity-check"
	StepBuild         = "08-build"
	StepDeploy        = "09-deploy"
	StepWebhookUpdate = "10-webhook-update"

	StepBackground = "XX-background"
	StepBug        = "XX-bug"
)

func telemetry(ctx context.Context, argData map[string]interface{}) {
	data := map[string]interface{}{
		"component":        "kale",
		"trace_replica_id": globalKale.telemetryReplicaID.String(),
		"trace_iteration":  CtxGetIteration(ctx),
	}
	if proj := CtxGetProject(ctx); proj != nil {
		data["trace_project_uid"] = proj.GetUID()
		data["trace_project_gen"] = proj.GetGeneration()
	}
	if commit := CtxGetCommit(ctx); commit != nil {
		data["trace_commit_uid"] = commit.GetUID()
	}
	for k, v := range argData {
		data[k] = v
	}
	if _, err := globalKale.telemetryReporter.Report(ctx, data); err != nil {
		dlog.GetLogger(ctx).Errorln("telemetry:", err)
	}
}

func telemetryErr(ctx context.Context, traceStep string, err error) {
	telemetry(ctx, map[string]interface{}{
		"trace_step": traceStep,
		"err":        fmt.Sprintf("%+v", err), // use %+v to include a stack trace if there is one
	})
}

func telemetryOK(ctx context.Context, traceStep string) {
	telemetry(ctx, map[string]interface{}{
		"trace_step": traceStep,
		"success":    true,
	})
}

func Setup(group *group.Group, httpHandler lyftserver.DebugHTTPHandler, info *k8s.KubeInfo, dynamicClient k8sClientDynamic.Interface) {
	k := NewKale(dynamicClient)
	globalKale = k

	upstreamWorker := make(chan UntypedSnapshot)
	downstreamWorker := make(chan UntypedSnapshot)
	go coalesce(upstreamWorker, downstreamWorker)

	upstreamWebUI := make(chan UntypedSnapshot)
	downstreamWebUI := make(chan UntypedSnapshot)
	go coalesce(upstreamWebUI, downstreamWebUI)

	group.Go("kale_watcher", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
		k.cfg = cfg
		softCtx = dlog.WithLogger(softCtx, l)

		client, err := k8s.NewClient(info)
		if err != nil {
			// this is non fatal (mostly just to facilitate local dev); don't `return err`
			reportRuntimeError(softCtx, StepSetup,
				fmt.Errorf("kale disabled: k8s.NewClient: %w", err))
			return nil
		}

		w := client.Watcher()
		var wg WatchGroup

		handler := func(w *k8s.Watcher) {
			snapshot := UntypedSnapshot{
				Projects:     w.List("projects.getambassador.io"),
				Commits:      w.List("projectcommits.getambassador.io"),
				Jobs:         w.List("jobs.batch"),
				StatefulSets: w.List("statefulsets.apps"),
			}
			upstreamWorker <- snapshot
			upstreamWebUI <- snapshot
		}

		labelSelector := GlobalLabelName + "=" + cfg.AmbassadorID
		queries := []k8s.Query{
			{Kind: "projects.getambassador.io"},
			{Kind: "projectcommits.getambassador.io"},
			{Kind: "jobs.batch", LabelSelector: labelSelector},
			{Kind: "statefulset.apps", LabelSelector: labelSelector},
		}

		for _, query := range queries {
			err := w.WatchQuery(query, wg.Wrap(softCtx, handler))
			if err != nil {
				// this is non fatal (mostly just to facilitate local dev); don't `return err`
				reportRuntimeError(softCtx, StepSetup,
					fmt.Errorf("kale disabled: WatchQuery(%#v, ...): %w", query, err))
				return nil
			}
		}

		if err := safeInvoke(w.Start); err != nil {
			// RBAC!
			reportRuntimeError(softCtx, StepSetup,
				fmt.Errorf("kale disabled: Start(): %w", err))
			return nil
		}
		go func() {
			<-softCtx.Done()
			w.Stop()
		}()
		telemetryOK(softCtx, StepSetup)
		w.Wait()
		close(upstreamWorker)
		close(upstreamWebUI)
		return nil
	})

	group.Go("kale_worker", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
		softCtx = dlog.WithLogger(softCtx, l)
		var telemetryIteration uint64

		err := leaderelection.RunAsSingleton(softCtx, cfg, info, "kale", 15*time.Second, func(ctx context.Context) {
			telemetryOK(ctx, StepLeader)
			for {
				select {
				case <-ctx.Done():
					return
				case _snapshot, ok := <-downstreamWorker:
					if !ok {
						return
					}
					ctx := CtxWithIteration(ctx, telemetryIteration)
					telemetryIteration++
					snapshot := _snapshot.TypedAndFiltered(ctx, cfg.AmbassadorID)
					err := safeInvoke(func() { k.reconcile(ctx, snapshot) })
					if err != nil {
						reportThisIsABug(ctx, err)
					}
					k.flushIterationErrors()
				}
			}
		})
		if err != nil {
			// Similar to Setup(), this is non-fatal
			reportRuntimeError(softCtx, StepLeader,
				fmt.Errorf("kale disabled: leader election: %w", err))
		}
		return nil
	})

	group.Go("kale_webui_update", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
		softCtx = dlog.WithLogger(softCtx, l)

		for {
			select {
			case <-softCtx.Done():
				return nil
			case _snapshot, ok := <-downstreamWebUI:
				if !ok {
					return nil
				}
				snapshot := _snapshot.TypedAndFiltered(softCtx, cfg.AmbassadorID)
				err := safeInvoke(func() { k.updateInternalState(softCtx, snapshot) })
				if err != nil {
					// recovered from a panic--this is a bug
					reportThisIsABug(softCtx, fmt.Errorf("recovered from panic: %w", err))
				}
			}
		}
	})

	// todo: lock down all these endpoints with auth
	handler := http.StripPrefix("/edge_stack_ui/edge_stack", ezHTTPHandler(k.dispatch)).ServeHTTP
	// todo: this is just temporary, we will consolidate these sprawling endpoints later
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/projects", "kale projects api", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/githook/", "kale githook", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/logs/", "kale logs api", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/slogs/", "kale server logs api", handler)
}

type UntypedSnapshot struct {
	Projects     []k8s.Resource
	Commits      []k8s.Resource
	Jobs         []k8s.Resource
	StatefulSets []k8s.Resource
}

type Snapshot struct {
	Projects     []*Project
	Commits      []*ProjectCommit
	Jobs         []*k8sTypesBatchV1.Job
	StatefulSets []*k8sTypesAppsV1.StatefulSet
}

func (in UntypedSnapshot) TypedAndFiltered(ctx context.Context, ambassadorID string) Snapshot {
	var out Snapshot

	// For built-in resource types, it is appropriate to a panic
	// because it's a bug; the api-server only gives us valid
	// resources, so if we fail to parse them, it's a bug in how
	// we're parsing.  For the same reason, it's "safe" to do this
	// all at once, because we don't need to do individual
	// validation, because they're all valid.
	if err := mapstructure.Convert(in.Jobs, &out.Jobs); err != nil {
		panicThisIsABug(fmt.Errorf("Jobs: %w", err))
	}
	if err := mapstructure.Convert(in.StatefulSets, &out.StatefulSets); err != nil {
		panicThisIsABug(fmt.Errorf("StatefulSets: %w", err))
	}

	// However, for our CRDs, because the api-server can't
	// validate that CRs are valid the way that it can for
	// built-in Resources, we have to safely deal with the
	// possibility that any individual resource is invalid, and
	// not let that affect the others.
	for _, inProj := range in.Projects {
		var outProj *Project
		if err := mapstructure.Convert(inProj, &outProj); err != nil {
			reportRuntimeError(ctx, StepValidProject,
				fmt.Errorf("Project: %w", err))
			continue
		}
		telemetryOK(CtxWithProject(ctx, outProj), StepValidProject)
		out.Projects = append(out.Projects, outProj)
	}
	for _, inCommit := range in.Commits {
		var outCommit *ProjectCommit
		if err := mapstructure.Convert(inCommit, &outCommit); err != nil {
			reportThisIsABug(ctx, fmt.Errorf("Commit: %w", err))
			continue
		}
		out.Commits = append(out.Commits, outCommit)
	}

	return out
}

func coalesce(upstream <-chan UntypedSnapshot, downstream chan<- UntypedSnapshot) {
do_read:
	item, ok := <-upstream
did_read:
	if !ok {
		close(downstream)
		return
	}
	select {
	case downstream <- item:
		goto do_read
	case item, ok = <-upstream:
		goto did_read
	}
}

// A kale contains the global state for the controller/webhook. We
// assume there is only one copy of the controller running in the
// cluster, so this is global to the entire cluster.
func NewKale(dynamicClient k8sClientDynamic.Interface) *kale {
	return &kale{
		telemetryReporter:  aes_metriton.Reporter,
		telemetryReplicaID: uuid.New(),

		projectsGetter: dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "projects"}),
		commitsGetter:  dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "projectcommits"}),
		Projects:       make(map[k8sTypes.UID]*projectAndChildren),
	}
}

type kale struct {
	cfg types.Config

	telemetryReporter  *metriton.Reporter
	telemetryReplicaID uuid.UUID

	projectsGetter k8sClientDynamic.NamespaceableResourceInterface
	commitsGetter  k8sClientDynamic.NamespaceableResourceInterface

	mu                  sync.RWMutex
	Projects            map[k8sTypes.UID]*projectAndChildren
	GlobalErrors        []recordedError
	ErrorsDirty         bool
	PersistentErrors    []recordedError
	PrevIterationErrors []recordedError

	NextIterationErrors []recordedError
}

func (k *kale) addPersistentError(err error, projectUID, commitUID k8sTypes.UID) {
	now := time.Now()
	k.mu.Lock()
	defer k.mu.Unlock()
	k.PersistentErrors = append(k.PersistentErrors, recordedError{
		Time:       now,
		Message:    fmt.Sprintf("%+v", err), // use %+v to include a stack trace if there is one
		ProjectUID: projectUID,
		CommitUID:  commitUID,
	})
	k.ErrorsDirty = true
}

func (k *kale) addIterationError(err error, projectUID, commitUID k8sTypes.UID) {
	now := time.Now()
	k.NextIterationErrors = append(k.NextIterationErrors, recordedError{
		Time:       now,
		Message:    fmt.Sprintf("%+v", err), // use %+v to include a stack trace if there is one
		ProjectUID: projectUID,
		CommitUID:  commitUID,
	})
}

func (k *kale) syncErrors() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.ErrorsDirty {
		return
	}
	// clear everything
	k.GlobalErrors = nil
	for _, project := range k.Projects {
		project.Children.Errors = nil
		for _, commit := range project.Children.Commits {
			commit.Children.Errors = nil
		}
	}
	// re-populate everything
	for _, err := range append(k.PersistentErrors, k.PrevIterationErrors...) {
		if project, projectOK := k.Projects[err.ProjectUID]; projectOK {
			var commit *commitAndChildren
			for _, straw := range project.Children.Commits {
				if straw.GetUID() == err.CommitUID {
					commit = straw
					break
				}
			}
			if commit != nil {
				commit.Children.Errors = append(commit.Children.Errors, err)
			} else {
				project.Children.Errors = append(project.Children.Errors, err)
			}
		} else {
			k.GlobalErrors = append(k.GlobalErrors, err)
		}
	}
	// sort everything
	sortErrors(k.GlobalErrors)
	for _, project := range k.Projects {
		sortErrors(project.Children.Errors)
		for _, commit := range project.Children.Commits {
			sortErrors(commit.Children.Errors)
		}
	}
}

func sortErrors(errs []recordedError) {
	sort.Slice(errs, func(i, j int) bool {
		return errs[i].Time.Before(errs[j].Time)
	})
}

func (k *kale) flushIterationErrors() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.PrevIterationErrors = k.NextIterationErrors
	k.NextIterationErrors = nil
	k.ErrorsDirty = true
}

func (k *kale) reconcile(ctx context.Context, snapshot Snapshot) {
	k.reconcileGitHub(ctx, snapshot.Projects)
	k.reconcileCluster(ctx, snapshot)
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
		Builders []*k8sTypesBatchV1.Job        `json:"builders"`
		Runners  []*k8sTypesAppsV1.StatefulSet `json:"runners"`
		Errors   []recordedError               `json:"errors"`
	} `json:"children"`
}

func (k *kale) updateInternalState(ctx context.Context, snapshot Snapshot) {
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
	for _, statefulset := range snapshot.StatefulSets {
		key := k8sTypes.UID(statefulset.GetLabels()[CommitLabelName])
		if _, ok := commits[key]; !ok {
			reportRuntimeError(ctx, StepBackground,
				fmt.Errorf("unable to pair StatefulSet %q.%q with ProjectCommit; ignoring",
					statefulset.GetName(), statefulset.GetNamespace()))
			continue
		}
		commits[key].Children.Runners = append(commits[key].Children.Runners, statefulset)
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

	k.mu.Lock()
	k.Projects = projects
	k.mu.Unlock()
}

func (k *kale) reconcileGitHub(ctx context.Context, projects []*Project) {
	for _, proj := range projects {
		ctx := CtxWithProject(ctx, proj)
		err := postHook(proj.Spec.GithubRepo,
			fmt.Sprintf("https://%s/edge_stack/api/githook/%s", proj.Spec.Host, proj.Key()),
			proj.Spec.GithubToken)
		if err != nil {
			reportRuntimeError(ctx, StepReconcileWebhook, err)
		} else {
			telemetryOK(ctx, StepReconcileWebhook)
		}
	}
}

// This is our dispatcher for everything under /api/. This looks at
// the URL and based on it figures out an appropriate handler to
// call. All the real business logic for the web API is in the methods
// this calls.
func (k *kale) dispatch(r *http.Request) httpResult {
	parts := strings.Split(r.URL.Path[1:], "/")
	if parts[0] != "api" {
		panicThisIsABug(errors.New("this shouldn't happen"))
	}
	switch parts[1] {
	case "githook":
		eventType := r.Header.Get("X-GitHub-Event")
		switch eventType {
		case "ping":
			return httpResult{status: 200, body: "pong"}
		case "push":
			return k.handlePush(r, strings.Join(parts[2:], "/"))
		default:
			return httpResult{status: 500, body: fmt.Sprintf("don't know how to handle %s events", eventType)}
		}
	case "projects":
		return httpResult{status: 200, body: k.projectsJSON()}
	case "logs", "slogs":
		logType := parts[1]
		commitQName := parts[2]
		sep := strings.LastIndexByte(commitQName, '.')
		if len(parts) > 3 || sep < 0 {
			return httpResult{status: http.StatusNotFound, body: "not found"}
		}
		namespace := commitQName[sep+1:]

		// resolve commitQName to a UID
		var commitUID k8sTypes.UID
		k.mu.RLock()
	OuterLoop:
		for _, proj := range k.Projects {
			if proj.GetNamespace() != namespace {
				continue
			}
			for _, commit := range proj.Children.Commits {
				if commit.GetName()+"."+commit.GetNamespace() == commitQName {
					commitUID = commit.GetUID()
					break OuterLoop
				}
			}
		}
		k.mu.RUnlock()
		if commitUID == "" {
			return httpResult{status: http.StatusNotFound, body: "not found"}
		}

		selectors := []string{
			GlobalLabelName + "==" + k.cfg.AmbassadorID,
			CommitLabelName + "==" + string(commitUID),
		}
		if logType == "slogs" {
			selectors = append(selectors, "statefulset.kubernetes.io/pod-name")
		} else {
			selectors = append(selectors, JobLabelName)
		}
		return httpResult{
			stream: func(w http.ResponseWriter) {
				err := streamLogs(w, r, namespace, strings.Join(selectors, ","))
				if err != nil {
					panicFlowControl(err)
				}
			},
		}
	}
	return httpResult{status: 400, body: "bad request"}
}

// Returns the a JSON string with all the data for the root of the
// UI. This is a map of all the projects plus nested data as
// appropriate.
func (k *kale) projectsJSON() string {
	k.syncErrors()
	k.mu.RLock()
	defer k.mu.RUnlock()

	var keys []string
	for key, _ := range k.Projects {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)

	var results struct {
		Projects []*projectAndChildren `json:"projects"`
		Errors   []recordedError       `json:"errors"`
	}
	results.Errors = k.GlobalErrors
	results.Projects = make([]*projectAndChildren, 0, len(k.Projects))
	for _, key := range keys {
		results.Projects = append(results.Projects, k.Projects[k8sTypes.UID(key)])
	}

	bytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		// Everything in results should be serializable to
		// JSON--this should never happen.
		panicThisIsABug(err)
	}

	return string(bytes) + "\n"
}

// Handle Push events from the github API.
func (k *kale) handlePush(r *http.Request, key string) httpResult {
	ctx := r.Context()

	k.mu.RLock()
	proj, ok := k.Projects[k8sTypes.UID(key)]
	k.mu.RUnlock()
	if !ok {
		reportRuntimeError(ctx, StepWebhookUpdate,
			errors.New("no such project"))
		return httpResult{status: 404, body: fmt.Sprintf("no such project %s", key)}
	}
	ctx = CtxWithProject(ctx, proj.Project)

	var push Push
	if err := json.NewDecoder(r.Body).Decode(&push); err != nil {
		reportRuntimeError(ctx, StepWebhookUpdate,
			fmt.Errorf("git webhook parse error: %w", err))
		return httpResult{status: 400, body: err.Error()}
	}

	// GitHub calls the hook asynchronously--it might not actually
	// be ready for us to do things based on the hook.  Poll
	// GitHub until we see that what the hook says has come to
	// pass... or we time out.
	gitReady := false
	apiReady := false
	deadline := time.Now().Add(2 * time.Minute)
	backoff := 1 * time.Second
	for (!gitReady || !apiReady) && time.Now().Before(deadline) {
		if !gitReady {
			ref, err := gitResolveRef("https://github.com/"+proj.Spec.GithubRepo, proj.Spec.GithubToken,
				libgitPlumbing.ReferenceName(push.Ref))
			if err != nil {
				continue
			}
			if ref.Hash().String() == push.After {
				gitReady = true
			}
		}
		if !apiReady {
			var prs []Pull
			if err := getJSON(fmt.Sprintf("https://api.github.com/repos/%s/pulls", proj.Spec.GithubRepo), proj.Spec.GithubToken, &prs); err != nil {
				continue
			}
			havePr := false
			for _, pr := range prs {
				if strings.EqualFold(pr.Head.Repo.FullName, proj.Spec.GithubRepo) &&
					"refs/heads/"+pr.Head.Ref == push.Ref {
					havePr = true
					if pr.Head.Sha == push.After {
						apiReady = true
					}
				}
			}
			if !havePr {
				apiReady = true
			}
		}
		time.Sleep(backoff)
		if backoff < 10*time.Second {
			backoff *= 2
		}
	}
	if gitReady && apiReady {
		// Bump the project's .Status, to trigger a reconcile via
		// Kubernetes.  We do this instead of just poking the right
		// bits in memory because we might not be the elected leader.
		proj.Status.LastPush = time.Now()
		uProj := unstructureProject(proj.Project)
		_, err := k.projectsGetter.Namespace(proj.GetNamespace()).UpdateStatus(uProj, k8sTypesMetaV1.UpdateOptions{})
		if err != nil {
			dlog.GetLogger(ctx).Println("update project status:", err)
			telemetryOK(ctx, StepWebhookUpdate)
		}
	}

	return httpResult{status: 200, body: ""}
}

type Push struct {
	Ref        string
	After      string
	Repository struct {
		GitUrl      string `json:"git_url"`
		StatusesUrl string `json:"statuses_url"`
	}
}

func (k *kale) calculateBuild(proj *Project, commit *ProjectCommit) []interface{} {
	// Note: If the kaniko destination is set to the full service name
	// (registry.ambassador.svc.cluster.local), then we can't seem to push
	// to the no matter how we tweak the settings. I assume this is due to
	// some special handling of .local domains somewhere.
	//
	// todo: the ambassador namespace is hardcoded below in the registry
	//       we to which we push

	job := &k8sTypesBatchV1.Job{
		TypeMeta: k8sTypesMetaV1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: k8sTypesMetaV1.ObjectMeta{
			Name:      commit.GetName() + "-build",
			Namespace: commit.GetNamespace(),
			Labels: map[string]string{
				GlobalLabelName: k.cfg.AmbassadorID,
				CommitLabelName: string(commit.GetUID()),
			},
			OwnerReferences: []k8sTypesMetaV1.OwnerReference{
				{
					Controller:         boolPtr(true),
					BlockOwnerDeletion: boolPtr(true),
					Kind:               commit.TypeMeta.Kind,
					APIVersion:         commit.TypeMeta.APIVersion,
					Name:               commit.GetName(),
					UID:                commit.GetUID(),
				},
			},
		},
		Spec: k8sTypesBatchV1.JobSpec{
			BackoffLimit: int32Ptr(1),
			Template: k8sTypesCoreV1.PodTemplateSpec{
				ObjectMeta: k8sTypesMetaV1.ObjectMeta{
					Labels: map[string]string{
						GlobalLabelName: k.cfg.AmbassadorID,
						CommitLabelName: string(commit.GetUID()),
					},
				},
				Spec: k8sTypesCoreV1.PodSpec{
					Containers: []k8sTypesCoreV1.Container{
						{
							Name:  "kaniko",
							Image: "quay.io/datawire/aes-project-builder@sha256:0b040450945683869343d9e5caaa04dac91c52c28722031dc133da18a3b84899",
							Args: []string{
								"--cache=true",
								"--skip-tls-verify",
								"--skip-tls-verify-pull",
								"--skip-tls-verify-registry",
								"--dockerfile=Dockerfile",
								"--destination=registry." + k.cfg.AmbassadorNamespace + "/" + commit.Spec.Rev,
							},
							Env: []k8sTypesCoreV1.EnvVar{
								{Name: "KALE_CREDS", Value: proj.Spec.GithubToken},
								{Name: "KALE_REPO", Value: proj.Spec.GithubRepo},
								{Name: "KALE_REF", Value: commit.Spec.Ref.String()},
								{Name: "KALE_REV", Value: commit.Spec.Rev},
							},
						},
					},
					RestartPolicy: k8sTypesCoreV1.RestartPolicyNever,
				},
			},
		},
	}

	if errs := k8sValidation.IsValidLabelValue(job.GetName()); len(errs) != 0 {
		// The Kubernetes docs say:
		//
		//    Leave `manualSelector` unset unless you are certain what
		//    you are doing. When false or unset, the system pick labels
		//    unique to this job and appends those labels to the pod
		//    template.
		//
		// ... yeah, except that the code in Kubernetes that does that
		// is just broken for Jobs with names >63 characters long[1].
		// Do they document that Jobs must have short names?  No,
		// because that's not a requirement; it's just a bug in the Job
		// controller.
		//
		// [1] Or rather, any name that isn't a valid label value.
		// Without thinking about it too hard, I think length is the
		// only way a valid name can not be a valid label value.
		//
		// So, that forces us to set `manualSelector` and do it
		// ourselves :/ We don't do this if we think the built-in
		// Kubernetes controller will do the right thing, because it has
		// access to the Job UID (and we don't), which is a robustness
		// win.
		job.Spec.ManualSelector = boolPtr(true)
		// We can't just set "controller-uid" to job.GetUID() (which
		// would be a safe subset of Kubernetes' built-in behavior),
		// because the Job's UID hasn't been populated yet!  And we
		// can't just generate a UUID to use, because we want this
		// function to be deterministic.  So cram in the information we
		// already have, and also add in a hash of the job name.
		//
		// I chose SHA-2/224 because I was going to choose SHA-2/256
		// because that's usually a safe choice, but I needed something
		// that hex-encodes to <64 characters (SHA-2/256 is 64
		// characters exactly), and I didn't want to think about if
		// truncating a hash is safe.
		nameHash := fmt.Sprintf("%x", sha256.Sum224([]byte(job.GetName())))
		job.Spec.Selector = &k8sTypesMetaV1.LabelSelector{
			MatchLabels: map[string]string{
				GlobalLabelName: k.cfg.AmbassadorID,
				CommitLabelName: string(commit.GetUID()),
				JobLabelName:    nameHash,
			},
		}
		job.Spec.Template.ObjectMeta.Labels[JobLabelName] = nameHash
	}

	return []interface{}{job}
}

func (k *kale) reconcileCluster(ctx context.Context, snapshot Snapshot) {
	// reconcile commits
	for _, proj := range snapshot.Projects {
		ctx := CtxWithProject(ctx, proj)
		commitManifests, err := k.calculateCommits(proj)
		if err != nil {
			continue
		}
		selectors := []string{
			GlobalLabelName + "==" + k.cfg.AmbassadorID,
			ProjectLabelName + "==" + string(proj.GetUID()),
		}
		err = applyAndPrune(
			strings.Join(selectors, ","),
			[]k8sSchema.GroupVersionKind{
				{Group: "getambassador.io", Version: "v2", Kind: "ProjectCommit"},
			},
			commitManifests)
		if err != nil {
			reportRuntimeError(ctx, StepReconcileProjectsToCommits,
				fmt.Errorf("updating ProjectCommits: %w", err))
		} else {
			telemetryOK(ctx, StepReconcileProjectsToCommits)
		}
	}

	// reconcile things managed by commits
	for _, commit := range snapshot.Commits {
		ctx := CtxWithCommit(ctx, commit)
		projectUID := k8sTypes.UID(commit.GetLabels()[ProjectLabelName])
		var project *Project
		for _, proj := range snapshot.Projects {
			if proj.GetUID() == projectUID {
				project = proj
			}
		}
		if project == nil {
			reportRuntimeError(ctx, StepReconcileCommitsToAction,
				errors.New("unable to pair ProjectCommit with Project"))
			continue
		}
		ctx = CtxWithProject(ctx, project)
		var commitBuilders []*k8sTypesBatchV1.Job
		for _, job := range snapshot.Jobs {
			if k8sTypes.UID(job.GetLabels()[CommitLabelName]) == commit.GetUID() {
				commitBuilders = append(commitBuilders, job)
			}
		}
		var commitRunners []*k8sTypesAppsV1.StatefulSet
		for _, statefulset := range snapshot.StatefulSets {
			if k8sTypes.UID(statefulset.GetLabels()[CommitLabelName]) == commit.GetUID() {
				commitRunners = append(commitRunners, statefulset)
			}
		}

		var runtimeErr, bugErr error
		bugErr = safeInvoke(func() {
			runtimeErr = k.reconcileCommit(ctx, project, commit, commitBuilders, commitRunners)
		})
		if runtimeErr != nil {
			reportRuntimeError(ctx, StepReconcileCommitsToAction, runtimeErr)
		}
		if bugErr != nil {
			reportThisIsABug(ctx,
				fmt.Errorf("recovered from panic: %w", bugErr))
		}
		if runtimeErr == nil && bugErr == nil {
			telemetryOK(ctx, StepReconcileCommitsToAction)
		}
	}
}

func (k *kale) reconcileCommit(ctx context.Context, proj *Project, commit *ProjectCommit, builders []*k8sTypesBatchV1.Job, runners []*k8sTypesAppsV1.StatefulSet) error {
	ctx = dlog.WithLogger(ctx, dlog.GetLogger(ctx).WithField("commit", commit.GetName()+"."+commit.GetNamespace()))
	log := dlog.GetLogger(ctx)

	var commitPhase CommitPhase
	// Decide what the phase of the commit should be, based on available evidence
	if len(runners) == 0 {
		if len(builders) != 1 {
			commitPhase = CommitPhase_Received
		} else {
			if complete, _ := jobConditionMet(builders[0], k8sTypesBatchV1.JobComplete, k8sTypesCoreV1.ConditionTrue); complete {
				// advance to next phase
				commitPhase = CommitPhase_Deploying
			} else if failed, _ := jobConditionMet(builders[0], k8sTypesBatchV1.JobFailed, k8sTypesCoreV1.ConditionTrue); failed {
				telemetryErr(ctx, StepBuild,
					fmt.Errorf("builder Job failed %d times", builders[0].Status.Failed))
				commitPhase = CommitPhase_BuildFailed
			} else {
				// keep waiting for one of the above to become true
				commitPhase = CommitPhase_Building
			}
		}
	} else {
		if len(runners) != 1 {
			commitPhase = CommitPhase_Deploying
		} else {
			if runners[0].Status.ObservedGeneration == runners[0].ObjectMeta.Generation &&
				runners[0].Status.CurrentRevision == runners[0].Status.UpdateRevision &&
				runners[0].Status.ReadyReplicas >= *runners[0].Spec.Replicas {
				// advance to next phase
				commitPhase = CommitPhase_Deployed
			} else {
				// TODO: Maybe inspect the pods that belong to this StatefulSet, in
				// order to detect a "failed" state.
				commitPhase = CommitPhase_Deploying
			}
		}
	}
	// If the detected phase of the commit doesn't match what's in the commit.status, inform
	// both Kubernetes and GitHub of the change.
	if commitPhase != commit.Status.Phase {
		commit.Status.Phase = commitPhase
		uCommit := unstructureCommit(commit)
		_, err := k.commitsGetter.Namespace(commit.GetNamespace()).UpdateStatus(uCommit, k8sTypesMetaV1.UpdateOptions{})
		if err != nil {
			log.Println("update commit status:", err)
		}
		err = postStatus(ctx, fmt.Sprintf("https://api.github.com/repos/%s/statuses/%s", proj.Spec.GithubRepo, commit.Spec.Rev),
			GitHubStatus{
				State: map[CommitPhase]string{
					CommitPhase_Received:  "pending",
					CommitPhase_Building:  "pending",
					CommitPhase_Deploying: "pending",
					CommitPhase_Deployed:  "success",

					CommitPhase_BuildFailed:  "failure",
					CommitPhase_DeployFailed: "failure",
				}[commitPhase],
				TargetUrl: map[CommitPhase]string{
					CommitPhase_Received:     proj.BuildLogUrl(commit),
					CommitPhase_Building:     proj.BuildLogUrl(commit),
					CommitPhase_BuildFailed:  proj.BuildLogUrl(commit),
					CommitPhase_Deploying:    proj.ServerLogUrl(commit),
					CommitPhase_DeployFailed: proj.ServerLogUrl(commit),
					CommitPhase_Deployed:     proj.PreviewUrl(commit),
				}[commitPhase],
				Description: commitPhase.String(),
				Context:     "aes",
			},
			proj.Spec.GithubToken)
		if err != nil {
			log.Println("update commit status:", err)
		}
	}

	var manifests []interface{}
	manifests = append(manifests, k.calculateBuild(proj, commit)...)
	if commit.Status.Phase >= CommitPhase_Deploying {
		telemetryOK(ctx, StepBuild)
		manifests = append(manifests, k.calculateRun(proj, commit)...)
	}
	if commit.Status.Phase == CommitPhase_Deployed {
		telemetryOK(ctx, StepDeploy)
	}
	selectors := []string{
		GlobalLabelName + "==" + k.cfg.AmbassadorID,
		CommitLabelName + "==" + string(commit.GetUID()),
	}
	err := applyAndPrune(
		strings.Join(selectors, ","),
		[]k8sSchema.GroupVersionKind{
			{Group: "getambassador.io", Version: "v2", Kind: "Mapping"},
			{Group: "", Version: "v1", Kind: "Service"},
			{Group: "batch", Version: "v1", Kind: "Job"},
			{Group: "apps", Version: "v1", Kind: "StatefulSet"},
		},
		manifests)
	if err != nil {
		if strings.Contains(err.Error(), "Forbidden: updates to statefulset spec for fields other than 'replicas', 'template', and 'updateStrategy' are forbidden") {
			deleteResource("statefulset.v1.apps", commit.GetName(), commit.GetNamespace())
		} else if regexp.MustCompile("(?s)The Job .* is invalid.* field is immutable").MatchString(err.Error()) {
			deleteResource("job.v1.batch", commit.GetName()+"-build", commit.GetNamespace())
		} else {
			reportRuntimeError(ctx, StepReconcileCommitsToAction,
				fmt.Errorf("deploying ProjectCommit: %w",
					err))
		}
	}

	return nil
}

func (k *kale) calculateRun(proj *Project, commit *ProjectCommit) []interface{} {
	return []interface{}{
		&Mapping{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "getambassador.io/v2",
				Kind:       "Mapping",
			},
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      commit.GetName(),
				Namespace: commit.GetNamespace(),
				OwnerReferences: []k8sTypesMetaV1.OwnerReference{
					{
						Controller:         boolPtr(true),
						BlockOwnerDeletion: boolPtr(true),
						Kind:               commit.TypeMeta.Kind,
						APIVersion:         commit.TypeMeta.APIVersion,
						Name:               commit.GetName(),
						UID:                commit.GetUID(),
					},
				},
				Labels: map[string]string{
					GlobalLabelName: k.cfg.AmbassadorID,
					CommitLabelName: string(commit.GetUID()),
				},
			},
			Spec: MappingSpec{
				AmbassadorID: aproTypesV2.AmbassadorID{k.cfg.AmbassadorID},
				// todo: figure out what is going on with /edge_stack/previews
				// not being routable
				Prefix:  "/.previews/" + proj.Spec.Prefix + "/" + commit.Spec.Rev + "/",
				Service: commit.GetName(),
			},
		},
		&k8sTypesCoreV1.Service{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      commit.GetName(),
				Namespace: commit.GetNamespace(),
				OwnerReferences: []k8sTypesMetaV1.OwnerReference{
					{
						Controller:         boolPtr(true),
						BlockOwnerDeletion: boolPtr(true),
						Kind:               commit.TypeMeta.Kind,
						APIVersion:         commit.TypeMeta.APIVersion,
						Name:               commit.GetName(),
						UID:                commit.GetUID(),
					},
				},
				Labels: map[string]string{
					GlobalLabelName: k.cfg.AmbassadorID,
					CommitLabelName: string(commit.GetUID()),
				},
			},
			Spec: k8sTypesCoreV1.ServiceSpec{
				Selector: map[string]string{
					GlobalLabelName: k.cfg.AmbassadorID,
					CommitLabelName: string(commit.GetUID()),
				},
				Ports: []k8sTypesCoreV1.ServicePort{
					{
						Protocol:   k8sTypesCoreV1.ProtocolTCP,
						Port:       80,
						TargetPort: intstr.FromInt(8080),
					},
				},
			},
		},
		// Use a StatefulSet (as opposed to a Deployment) so that the Pod name is
		// friendlier.
		&k8sTypesAppsV1.StatefulSet{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "StatefulSet",
			},
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      commit.GetName(),
				Namespace: commit.GetNamespace(),
				OwnerReferences: []k8sTypesMetaV1.OwnerReference{
					{
						Controller:         boolPtr(true),
						BlockOwnerDeletion: boolPtr(true),
						Kind:               commit.TypeMeta.Kind,
						APIVersion:         commit.TypeMeta.APIVersion,
						Name:               commit.GetName(),
						UID:                commit.GetUID(),
					},
				},
				Labels: map[string]string{
					GlobalLabelName: k.cfg.AmbassadorID,
					CommitLabelName: string(commit.GetUID()),
				},
			},
			Spec: k8sTypesAppsV1.StatefulSetSpec{
				Selector: &k8sTypesMetaV1.LabelSelector{
					MatchLabels: map[string]string{
						GlobalLabelName: k.cfg.AmbassadorID,
						CommitLabelName: string(commit.GetUID()),
					},
				},
				ServiceName: commit.GetName(),
				Template: k8sTypesCoreV1.PodTemplateSpec{
					ObjectMeta: k8sTypesMetaV1.ObjectMeta{
						Labels: map[string]string{
							GlobalLabelName: k.cfg.AmbassadorID,
							CommitLabelName: string(commit.GetUID()),
						},
					},
					Spec: k8sTypesCoreV1.PodSpec{
						Containers: []k8sTypesCoreV1.Container{
							{
								Name:  "app",
								Image: "127.0.0.1:31000/" + commit.Spec.Rev,
								Env: []k8sTypesCoreV1.EnvVar{
									{Name: "AMB_PROJECT_PREVIEW", Value: strconv.FormatBool(commit.Spec.IsPreview)},
									{Name: "AMB_PROJECT_REPO", Value: proj.Spec.GithubRepo},
									{Name: "AMB_PROJECT_REF", Value: commit.Spec.Ref.String()},
									{Name: "AMB_PROJECT_REV", Value: commit.Spec.Rev},
								},
							},
						},
					},
				},
			},
		},
	}
}
