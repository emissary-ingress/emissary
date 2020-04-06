package kale

import (
	// standard library
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
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
	"github.com/datawire/apro/cmd/amb-sidecar/api"
	"github.com/datawire/apro/cmd/amb-sidecar/group"
	"github.com/datawire/apro/cmd/amb-sidecar/k8s/leaderelection"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
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
	GlobalLabelName     = "projects.getambassador.io/ambassador_id" // cfg.AmbassadorID
	ProjectLabelName    = "projects.getambassador.io/project-uid"   // proj.GetUID()
	CommitLabelName     = "projects.getambassador.io/commit-uid"    // commit.GetUID()
	JobLabelName        = "job-name"                                // Don't change this; it's the label name used by Kubernetes itself
	ServicePodLabelName = "projects.getambassador.io/service"       // "true"
)

var globalKale *kale

const (
	StepSetup                      = "00-setup"
	StepLeader                     = "01-leader"
	StepValidProject               = "02-validproject"
	StepReconcileWebhook           = "03-reconcilewebook"
	StepReconcileController        = "04-reconcilecontroller"
	StepReconcileProjectsToCommits = "05-reconcileprojectstocommits"
	StepReconcileCommitsToAction   = "06-reconcilecommitstoaction"
	//StepGitPull                    = "07-git-pull"
	//StepGitSanityCheck             = "08-git-sanity-check"
	StepBuild         = "09-build"
	StepDeploy        = "10-deploy"
	StepWebhookUpdate = "11-webhook-update"

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

func Setup(group *group.Group, httpHandler lyftserver.DebugHTTPHandler, info *k8s.KubeInfo, dynamicClient k8sClientDynamic.Interface, pubkey *rsa.PublicKey) {
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
				Controllers: w.List("projectcontrollers.getambassador.io"),
				Projects:    w.List("projects.getambassador.io"),
				Commits:     w.List("projectcommits.getambassador.io"),
				Jobs:        w.List("jobs.batch"),
				Deployments: w.List("deployments.apps"),
			}
			upstreamWorker <- snapshot
			upstreamWebUI <- snapshot
		}

		labelSelector := GlobalLabelName + "=" + cfg.AmbassadorID
		queries := []k8s.Query{
			{Kind: "projectcontrollers.getambassador.io", LabelSelector: labelSelector},
			{Kind: "projects.getambassador.io"},
			{Kind: "projectcommits.getambassador.io"},
			{Kind: "jobs.batch", LabelSelector: labelSelector},
			{Kind: "deployments.apps", LabelSelector: labelSelector},
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
					snapshot := _snapshot.TypedAndIndexed(ctx)
					for _, proj := range snapshot.Projects {
						telemetryOK(CtxWithProject(ctx, proj.Project), StepValidProject)
					}
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
				snapshot := _snapshot.TypedAndIndexed(softCtx)
				k.mu.Lock()
				k.webSnapshot = snapshot
				k.mu.Unlock()
			}
		}
	})

	unauthenticated := ezHTTPHandler(k.dispatch)
	authenticated := api.PermitCookieAuth(
		func(path string) bool {
			return strings.HasPrefix(path, "/logs/")
		},
		api.AuthenticatedHTTPHandler(unauthenticated, pubkey))
	handler := http.StripPrefix("/edge_stack_ui/edge_stack/api/projects", authenticated).ServeHTTP
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/projects/kale-snapshot", "kale projects api", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/projects/logs/", "kale logs api", handler)
	githook := http.StripPrefix("/edge_stack_ui/edge_stack/api/projects", unauthenticated).ServeHTTP
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/projects/githook/", "kale githook", githook)
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
	}
}

type kale struct {
	cfg types.Config

	telemetryReporter  *metriton.Reporter
	telemetryReplicaID uuid.UUID

	projectsGetter k8sClientDynamic.NamespaceableResourceInterface
	commitsGetter  k8sClientDynamic.NamespaceableResourceInterface

	mu                  sync.RWMutex
	webSnapshot         *Snapshot
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

func (k *kale) updateWebGroupedSnapshot() *GroupedSnapshot {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.ErrorsDirty || k.webSnapshot.grouped == nil {
		k.webSnapshot.Errors = append(k.PersistentErrors, k.PrevIterationErrors...)
		k.webSnapshot.grouped = nil
		k.ErrorsDirty = false
	}
	return k.webSnapshot.Grouped()
}

func (k *kale) flushIterationErrors() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.PrevIterationErrors = k.NextIterationErrors
	k.NextIterationErrors = nil
	k.ErrorsDirty = true
}

func (k *kale) reconcile(ctx context.Context, snapshot *Snapshot) {
	_ = snapshot.Grouped() // populate .Children
	k.reconcileGitHub(ctx, snapshot)
	k.reconcileCluster(ctx, snapshot)
}

func (k *kale) reconcileGitHub(ctx context.Context, snapshot *Snapshot) {
	for _, proj := range snapshot.Projects {
		ctx := CtxWithProject(ctx, proj.Project)
		err := postHook(proj.Spec.GithubRepo,
			fmt.Sprintf("https://%s/edge_stack/api/projects/githook/%s", proj.Spec.Host, proj.Key()),
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
	switch parts[0] {
	case "githook":
		eventType := r.Header.Get("X-GitHub-Event")
		switch eventType {
		case "ping":
			return httpResult{status: 200, body: "pong"}
		case "push":
			return k.handlePush(r, strings.Join(parts[1:], "/"))
		default:
			return httpResult{status: 500, body: fmt.Sprintf("don't know how to handle %q events", eventType)}
		}
	case "kale-snapshot":
		return httpResult{status: 200, body: k.projectsJSON()}
	case "logs":
		logType := parts[1]
		commitQName := parts[2]
		sep := strings.LastIndexByte(commitQName, '.')
		if len(parts) > 3 || sep < 0 || !(logType == "build" || logType == "deploy") {
			return httpResult{status: http.StatusNotFound, body: "not found"}
		}
		namespace := commitQName[sep+1:]

		commit := k.GetCommit(commitQName)
		if commit == nil {
			return httpResult{status: http.StatusNotFound, body: "not found"}
		}

		selectors := []string{
			GlobalLabelName + "==" + k.cfg.AmbassadorID,
			CommitLabelName + "==" + string(commit.GetUID()),
		}
		if logType == "deploy" {
			selectors = append(selectors, ServicePodLabelName)
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
	default:
		return httpResult{status: http.StatusNotFound, body: "not found"}
	}
}

// Returns the a JSON string with all the data for the root of the
// UI. This is a map of all the projects plus nested data as
// appropriate.
func (k *kale) projectsJSON() string {
	snapshot := k.updateWebGroupedSnapshot()
	k.mu.RLock()
	defer k.mu.RUnlock()

	bytes, err := json.MarshalIndent(snapshot.Controllers[0].Children, "", "  ")
	if err != nil {
		// Everything in results should be serializable to
		// JSON--this should never happen.
		panicThisIsABug(err)
	}

	return string(bytes) + "\n"
}

func (k *kale) GetProject(key string) *projectAndChildren {
	k.mu.RLock()
	defer k.mu.RUnlock()
	for _, proj := range k.webSnapshot.Projects {
		if key == proj.Key() {
			return proj
		}
	}
	return nil
}

func (k *kale) GetCommit(qname string) *commitAndChildren {
	k.mu.RLock()
	defer k.mu.RUnlock()
	for _, commit := range k.webSnapshot.Commits {
		if commit.GetName()+"."+commit.GetNamespace() == qname {
			return commit
		}
	}
	return nil
}

// Handle Push events from the github API.
func (k *kale) handlePush(r *http.Request, key string) httpResult {
	ctx := r.Context()

	proj := k.GetProject(key)
	if proj == nil {
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
			BackoffLimit: int32Ptr(0),
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
							Image: "quay.io/datawire/aes-project-builder@sha256:f107422a588a2925634722c341b7174ff263610d9557e123e4ef41291cbf6c5e",
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

func (k *kale) reconcileCluster(ctx context.Context, snapshot *Snapshot) {
	if len(snapshot.Controllers) != 1 {
		name := "projectcontroller"
		if k.cfg.AmbassadorID != "default" {
			name += "-" + k.cfg.AmbassadorID
		}
		err := applyAndPrune(
			GlobalLabelName+"=="+k.cfg.AmbassadorID,
			[]k8sSchema.GroupVersionKind{
				{Group: "getambassador.io", Version: "v2", Kind: "ProjectController"},
			},
			[]interface{}{
				&ProjectController{
					TypeMeta: k8sTypesMetaV1.TypeMeta{
						APIVersion: "getambassador.io/v2",
						Kind:       "ProjectController",
					},
					ObjectMeta: k8sTypesMetaV1.ObjectMeta{
						Name:      name,
						Namespace: k.cfg.AmbassadorNamespace,
						Labels: map[string]string{
							GlobalLabelName: k.cfg.AmbassadorID,
						},
					},
					Spec: ProjectControllerSpec{},
				},
			})
		if err != nil {
			reportRuntimeError(ctx, StepReconcileController,
				fmt.Errorf("initializing ProjectController: %q", err))
		}
	}

	// reconcile commits
	for _, proj := range snapshot.Projects {
		ctx := CtxWithProject(ctx, proj.Project)
		commitManifests, err := k.calculateCommits(proj.Project)
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
		ctx := CtxWithCommit(ctx, commit.ProjectCommit)
		if commit.Parent == nil {
			//reportRuntimeError(ctx, StepReconcileCommitsToAction,
			//	errors.New("unable to pair ProjectCommit with Project"))
			continue
		}
		ctx = CtxWithProject(ctx, commit.Parent.Project)

		var runtimeErr, bugErr error
		bugErr = safeInvoke(func() {
			runtimeErr = k.reconcileCommit(ctx, commit)
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

func (k *kale) reconcileCommit(ctx context.Context, _commit *commitAndChildren) error {
	proj := _commit.Parent.Project
	commit := _commit.ProjectCommit
	builders := _commit.Children.Builders
	runners := _commit.Children.Runners

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
			dep := runners[0]
			if dep.Status.ObservedGeneration == dep.ObjectMeta.Generation &&
				(dep.Spec.Replicas == nil || dep.Status.UpdatedReplicas >= *dep.Spec.Replicas) &&
				dep.Status.UpdatedReplicas == dep.Status.Replicas &&
				dep.Status.AvailableReplicas == dep.Status.Replicas {
				// advance to next phase
				commitPhase = CommitPhase_Deployed
			} else {
				// TODO: Maybe inspect the pods that belong to this Deployment, in
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
			{Group: "apps", Version: "v1", Kind: "Deployment"},
		},
		manifests)
	if err != nil {
		if strings.Contains(err.Error(), "Forbidden: updates to deployment spec for fields other than 'replicas', 'template', and 'updateStrategy' are forbidden") {
			deleteResource("deployment.v1.apps", commit.GetName(), commit.GetNamespace())
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
	prefix := proj.Spec.Prefix
	if commit.Spec.IsPreview {
		// todo: figure out what is going on with /edge_stack/previews
		// not being routable
		prefix = "/.previews" + proj.Spec.Prefix + commit.Spec.Rev + "/"
	}
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
				Prefix:       prefix,
				Service:      commit.GetName(),
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
		&k8sTypesAppsV1.Deployment{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
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
			Spec: k8sTypesAppsV1.DeploymentSpec{
				Selector: &k8sTypesMetaV1.LabelSelector{
					MatchLabels: map[string]string{
						GlobalLabelName: k.cfg.AmbassadorID,
						CommitLabelName: string(commit.GetUID()),
					},
				},
				Template: k8sTypesCoreV1.PodTemplateSpec{
					ObjectMeta: k8sTypesMetaV1.ObjectMeta{
						Labels: map[string]string{
							GlobalLabelName:     k.cfg.AmbassadorID,
							CommitLabelName:     string(commit.GetUID()),
							ServicePodLabelName: "true",
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
