package kale

import (
	// standard library
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	// 3rd party
	"github.com/google/uuid"
	"github.com/pkg/errors"

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
	"github.com/datawire/apro/cmd/amb-sidecar/k8s/events"
	"github.com/datawire/apro/cmd/amb-sidecar/k8s/leaderelection"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/cmd/amb-sidecar/webui"
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
	RevisionLabelName   = "projects.getambassador.io/revision-uid"  // revision.GetUID()
	JobLabelName        = "job-name"                                // Don't change this; it's the label name used by Kubernetes itself
	ServicePodLabelName = "projects.getambassador.io/service"       // "true"
)

var globalKale *kale

const (
	StepSetup                        = "00-setup"
	StepLeader                       = "01-leader"
	StepValidProject                 = "02-validproject"
	StepReconcileWebhook             = "03-reconcilewebook"
	StepReconcileController          = "04-reconcilecontroller"
	StepReconcileProjectsToRevisions = "05-reconcileprojectstorevisions"
	StepReconcileRevisionsToAction   = "06-reconcilerevisionstoaction"
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
	if revision := CtxGetRevision(ctx); revision != nil {
		data["trace_revision_uid"] = revision.GetUID()
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

func Setup(group *group.Group, httpHandler lyftserver.DebugHTTPHandler, info *k8s.KubeInfo, dynamicClient k8sClientDynamic.Interface, pubkey *rsa.PublicKey, eventLogger *events.EventLogger) {
	k := NewKale(dynamicClient, eventLogger)
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
				errors.Wrap(err, "kale disabled: k8s.NewClient"))
			return nil
		}

		w := client.Watcher()
		var wg WatchGroup

		handler := func(w *k8s.Watcher) {
			snapshot := UntypedSnapshot{
				Controllers: w.List("projectcontrollers.getambassador.io"),
				Projects:    w.List("projects.getambassador.io"),
				Revisions:   w.List("projectrevisions.getambassador.io"),
				Jobs:        w.List("jobs.batch"),
				Deployments: w.List("deployments.apps"),
				Pods:        w.List("pods."),
				Events:      w.List("events."),
			}
			if len(snapshot.Controllers) == 0 {
				return
			} else {
				webui.SetFeatureFlag("butterscotch")
			}
			upstreamWorker <- snapshot
			upstreamWebUI <- snapshot
		}

		labelSelector := GlobalLabelName + "=" + cfg.AmbassadorID
		queries := []k8s.Query{
			{Kind: "projectcontrollers.getambassador.io", LabelSelector: labelSelector},
			{Kind: "projects.getambassador.io"},
			{Kind: "projectrevisions.getambassador.io"},
			{Kind: "jobs.batch", LabelSelector: labelSelector},
			{Kind: "deployments.apps", LabelSelector: labelSelector},
			{Kind: "pods.", LabelSelector: labelSelector},

			// BUG(lukeshu): It seems that if we give a watcher multiple queries with the same type but
			// different field selectors, only 1 of their callbacks gets called.  That's a bug in
			// github.com/datawire/ambassador/pkg/k8s, but I don't have time to fix it now.
			{Kind: "events."},
			//{Kind: "events.", FieldSelector: "involvedObject.apiVersion=getmabassador.io/v2,involvedObject.kind=ProjectController"},
			//{Kind: "events.", FieldSelector: "involvedObject.apiVersion=getmabassador.io/v2,involvedObject.kind=Project"},
			//{Kind: "events.", FieldSelector: "involvedObject.apiVersion=getmabassador.io/v2,involvedObject.kind=ProjectRevision"},
			//{Kind: "events.", FieldSelector: "involvedObject.apiVersion=batch/v1,involvedObject.kind=Job"},
			//{Kind: "events.", FieldSelector: "involvedObject.apiVersion=apps/v1,involvedObject.kind=Deployment"},
			//{Kind: "events.", FieldSelector: "involvedObject.apiVersion=v1,involvedObject.kind=Pod"},
		}

		for _, query := range queries {
			err := w.WatchQuery(query, wg.Wrap(softCtx, handler))
			if err != nil {
				// this is non fatal (mostly just to facilitate local dev); don't `return err`
				reportRuntimeError(softCtx, StepSetup,
					errors.Wrapf(err, "kale disabled: WatchQuery(%#v, ...)", query))
				return nil
			}
		}

		if err := safeInvoke(w.Start); err != nil {
			// RBAC!
			reportRuntimeError(softCtx, StepSetup,
				errors.Wrap(err, "kale disabled: Start()"))
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

		timer := time.NewTimer(0)
		<-timer.C
		setTimeout := func(d time.Duration) {
			timer.Stop()
			select {
			case <-timer.C:
			default:
			}
			timer.Reset(d)
		}

		main := func(ctx context.Context, _snapshot UntypedSnapshot) {
			ctx = CtxWithIteration(ctx, telemetryIteration)
			telemetryIteration++
			snapshot := _snapshot.TypedAndIndexed(ctx)
			for _, proj := range snapshot.Projects {
				telemetryOK(CtxWithProject(ctx, proj.Project), StepValidProject)
			}
			for uid, oldResourceVersion := range k.knownChangedProjects {
				newInstance, stillExists := snapshot.Projects[uid]
				if !stillExists || newInstance.GetResourceVersion() != oldResourceVersion {
					delete(k.knownChangedProjects, uid)
				}
			}
			for uid, oldResourceVersion := range k.knownChangedRevisions {
				newInstance, stillExists := snapshot.Revisions[uid]
				if !stillExists || newInstance.GetResourceVersion() != oldResourceVersion {
					delete(k.knownChangedRevisions, uid)
				}
			}
			if len(k.knownChangedProjects) > 0 || len(k.knownChangedRevisions) > 0 {
				return
			}
			err := safeInvoke(func() { k.reconcile(ctx, snapshot) })
			if err != nil {
				reportThisIsABug(ctx, err)
			}
			k.flushIterationErrors()
			setTimeout(20 * time.Second)
		}

		err := leaderelection.RunAsSingleton(softCtx, cfg, info, "kale", 15*time.Second, func(ctx context.Context) {
			telemetryOK(ctx, StepLeader)
			for {
				var snapshot UntypedSnapshot
				var ok bool
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					main(ctx, snapshot)
				case snapshot, ok = <-downstreamWorker:
					if !ok {
						return
					}
					main(ctx, snapshot)
				}
			}
		})
		if err != nil {
			// Similar to Setup(), this is non-fatal
			reportRuntimeError(softCtx, StepLeader,
				errors.Wrap(err, "kale disabled: leader election"))
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
func NewKale(dynamicClient k8sClientDynamic.Interface, eventLogger *events.EventLogger) *kale {
	return &kale{
		eventLogger:        eventLogger,
		telemetryReporter:  aes_metriton.Reporter,
		telemetryReplicaID: uuid.New(),

		projectsGetter:  dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "projects"}),
		revisionsGetter: dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "projectrevisions"}),

		knownChangedProjects:  make(map[k8sTypes.UID]string),
		knownChangedRevisions: make(map[k8sTypes.UID]string),

		PrevIterationErrors: make(map[recordedError]struct{}),
		NextIterationErrors: make(map[recordedError]struct{}),
		webSnapshot:         &Snapshot{},
	}
}

type kale struct {
	cfg types.Config

	eventLogger        *events.EventLogger
	telemetryReporter  *metriton.Reporter
	telemetryReplicaID uuid.UUID

	projectsGetter  k8sClientDynamic.NamespaceableResourceInterface
	revisionsGetter k8sClientDynamic.NamespaceableResourceInterface

	knownChangedProjects  map[k8sTypes.UID]string
	knownChangedRevisions map[k8sTypes.UID]string

	mu          sync.RWMutex
	webSnapshot *Snapshot
	// Errors to include in webui snapshots, but aren't in the
	// regular snapshot.  (Probably errors complaining that we
	// don't have RBAC to generate the regular snapshot).
	webErrors []*k8sTypesCoreV1.Event

	PrevIterationErrors map[recordedError]struct{}
	NextIterationErrors map[recordedError]struct{}
}

type recordedError struct {
	Message     string
	ProjectUID  k8sTypes.UID
	RevisionUID k8sTypes.UID
}

func (k *kale) addIterationError(err error, projectUID, revisionUID k8sTypes.UID) bool {
	key := recordedError{
		Message:     fmt.Sprintf("%+v", err), // use %+v to include a stack trace if there is one
		ProjectUID:  projectUID,
		RevisionUID: revisionUID,
	}
	k.NextIterationErrors[key] = struct{}{}
	_, inPrev := k.PrevIterationErrors[key]
	isNew := !inPrev
	return isNew
}

// addWebError adds an error to show in the webui, that isn't in a
// regular snapshot (Probably errors complaining that we don't have
// RBAC to generate the regular snapshot).
func (k *kale) addWebError(event *k8sTypesCoreV1.Event) {
	k.mu.Lock()
	k.webErrors = append(k.webErrors, event)
	k.webSnapshot.grouped = nil
	k.mu.Unlock()
}

func (k *kale) updateWebGroupedSnapshot() *GroupedSnapshot {
	k.mu.Lock() // need a write-lock to call .Grouped()
	defer k.mu.Unlock()
	ret := k.webSnapshot.Grouped()
	if len(k.webErrors) > 0 {
		ret.Controllers[0].Children.Errors = append(k.webErrors, ret.Controllers[0].Children.Errors...)
	}
	return ret
}

func (k *kale) flushIterationErrors() {
	// We could just do
	//
	//    prev = next
	//    next = make()
	//
	// But instead let's clear the old prev instead of making a
	// new map.  It's easy enough, and avoids putting extra
	// pressure on the GC.  (Yes, I did actually observe that it
	// makes a substantial impact on memory use).

	// swap
	k.PrevIterationErrors, k.NextIterationErrors = k.NextIterationErrors, k.PrevIterationErrors
	// clear
	for key := range k.NextIterationErrors {
		delete(k.NextIterationErrors, key)
	}
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
			continue
		}
		if proj.Status.Phase < ProjectPhase_WebhookCreated {
			proj.Status.Phase = ProjectPhase_WebhookCreated
			uProj := unstructureProject(proj.Project)
			_, err := k.projectsGetter.Namespace(proj.GetNamespace()).UpdateStatus(uProj, k8sTypesMetaV1.UpdateOptions{})
			if err != nil {
				reportRuntimeError(ctx, StepReconcileWebhook, err)
				continue
			}
			k.knownChangedProjects[proj.GetUID()] = proj.GetResourceVersion()
		}
		telemetryOK(ctx, StepReconcileWebhook)
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
		return k.handleWebhook(r, strings.Join(parts[1:], "/"))
	case "kale-snapshot":
		return httpResult{status: 200, body: k.projectsJSON()}
	case "logs":
		logType := parts[1]
		revisionQName := parts[2]
		sep := strings.LastIndexByte(revisionQName, '.')
		if len(parts) > 3 || sep < 0 || !(logType == "build" || logType == "deploy") {
			return httpResult{status: http.StatusNotFound, body: "not found"}
		}
		namespace := revisionQName[sep+1:]

		revision := k.GetRevision(revisionQName)
		if revision == nil {
			return httpResult{status: http.StatusNotFound, body: "not found"}
		}

		selectors := []string{
			GlobalLabelName + "==" + k.cfg.AmbassadorID,
			RevisionLabelName + "==" + string(revision.GetUID()),
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

func (k *kale) GetRevision(qname string) *revisionAndChildren {
	k.mu.RLock()
	defer k.mu.RUnlock()
	for _, revision := range k.webSnapshot.Revisions {
		if revision.GetName()+"."+revision.GetNamespace() == qname {
			return revision
		}
	}
	return nil
}

// Handle push or pull_request events from the github API.
func (k *kale) handleWebhook(r *http.Request, key string) httpResult {
	ctx := r.Context()

	_proj := k.GetProject(key)
	if _proj == nil {
		err := errors.Errorf("Git webhook called for non-existent project: %q", key)
		reportRuntimeError(ctx, StepWebhookUpdate, err)
		return httpResult{status: http.StatusNotFound, body: err.Error()}
	}
	ctx = CtxWithProject(ctx, _proj.Project)
	proj := *_proj.Project // get the value (not a pointer) so we can mutate it locally

	projDirty := false
	if proj.Status.Phase < ProjectPhase_WebhookConfirmed {
		proj.Status.Phase = ProjectPhase_WebhookConfirmed
		projDirty = true
	}

	eventType := r.Header.Get("X-GitHub-Event")
	switch eventType {
	case "ping":
		// do nothing
	case "push", "pull_request":
		proj.Status.LastWebhook = time.Now()
		projDirty = true
	default:
		return httpResult{status: http.StatusBadRequest, body: fmt.Sprintf("don't know how to handle %q events", eventType)}
	}

	telemetryOK(ctx, StepWebhookUpdate)

	if projDirty {
		uProj := unstructureProject(&proj)
		_, err := k.projectsGetter.Namespace(proj.GetNamespace()).UpdateStatus(uProj, k8sTypesMetaV1.UpdateOptions{})
		if err != nil {
			reportRuntimeError(ctx, StepWebhookUpdate, errors.Wrap(err, "update project status"))
		}
	}

	return httpResult{status: http.StatusOK, body: "ok"}
}

type Push struct {
	Ref        string
	After      string
	Repository struct {
		GitUrl      string `json:"git_url"`
		StatusesUrl string `json:"statuses_url"`
	}
}

func (k *kale) calculateBuild(proj *Project, revision *ProjectRevision) []interface{} {
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
			Name:      revision.GetName() + "-build",
			Namespace: revision.GetNamespace(),
			Labels: map[string]string{
				GlobalLabelName:   k.cfg.AmbassadorID,
				RevisionLabelName: string(revision.GetUID()),
			},
			OwnerReferences: []k8sTypesMetaV1.OwnerReference{
				{
					Controller:         boolPtr(true),
					BlockOwnerDeletion: boolPtr(true),
					Kind:               revision.TypeMeta.Kind,
					APIVersion:         revision.TypeMeta.APIVersion,
					Name:               revision.GetName(),
					UID:                revision.GetUID(),
				},
			},
		},
		Spec: k8sTypesBatchV1.JobSpec{
			BackoffLimit: int32Ptr(0),
			Template: k8sTypesCoreV1.PodTemplateSpec{
				ObjectMeta: k8sTypesMetaV1.ObjectMeta{
					Labels: map[string]string{
						GlobalLabelName:   k.cfg.AmbassadorID,
						RevisionLabelName: string(revision.GetUID()),
					},
				},
				Spec: k8sTypesCoreV1.PodSpec{
					Containers: []k8sTypesCoreV1.Container{
						{
							Name:  "kaniko",
							Image: "quay.io/datawire/aes-project-builder:release_v0.1.0",
							Args: []string{
								"--cache=true",
								"--skip-tls-verify",
								"--skip-tls-verify-pull",
								"--skip-tls-verify-registry",
								"--dockerfile=Dockerfile",
								"--destination=registry." + k.cfg.AmbassadorNamespace + "/" + proj.Namespace + "/" + proj.Name + ":" + revision.Spec.Rev,
							},
							Env: []k8sTypesCoreV1.EnvVar{
								{Name: "KALE_CREDS", Value: proj.Spec.GithubToken},
								{Name: "KALE_REPO", Value: proj.Spec.GithubRepo},
								{Name: "KALE_REF", Value: revision.Spec.Ref.String()},
								{Name: "KALE_REV", Value: revision.Spec.Rev},
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
				GlobalLabelName:   k.cfg.AmbassadorID,
				RevisionLabelName: string(revision.GetUID()),
				JobLabelName:      nameHash,
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
				errors.Wrap(err, "initializing ProjectController"))
		}
	}

	// reconcile revisions
	for _, proj := range snapshot.Projects {
		ctx := CtxWithProject(ctx, proj.Project)
		revisionManifests, err := k.calculateRevisions(proj.Project)
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
				{Group: "getambassador.io", Version: "v2", Kind: "ProjectRevision"},
			},
			revisionManifests)
		if err != nil {
			reportRuntimeError(ctx, StepReconcileProjectsToRevisions,
				errors.Wrap(err, "updating ProjectRevisions"))
		} else {
			telemetryOK(ctx, StepReconcileProjectsToRevisions)
		}
	}

	// reconcile things managed by revisions
	//
	// Because we're limiting the number of concurrent Jobs, it's
	// important that we iterate over Revisions in a stable (but
	// possibly arbitrary) order.  So, sort them by UID.
	runningJobs := 0
	for _, revisionUID := range sortedUIDKeys(snapshot.Revisions) {
		revision := snapshot.Revisions[revisionUID]
		if len(revision.Children.Builders) > 0 {
			succeeded, _ := jobConditionMet(revision.Children.Builders[0].Job, k8sTypesBatchV1.JobComplete, k8sTypesCoreV1.ConditionTrue)
			failed, _ := jobConditionMet(revision.Children.Builders[0].Job, k8sTypesBatchV1.JobFailed, k8sTypesCoreV1.ConditionTrue)
			if !(succeeded || failed) {
				runningJobs++
			}
		}
	}
	for _, revisionUID := range sortedUIDKeys(snapshot.Revisions) {
		revision := snapshot.Revisions[revisionUID]
		ctx := CtxWithRevision(ctx, revision.ProjectRevision)
		if revision.Parent == nil {
			//reportRuntimeError(ctx, StepReconcileRevisionsToAction,
			//	errors.New("unable to pair ProjectRevision with Project"))
			continue
		}
		ctx = CtxWithProject(ctx, revision.Parent.Project)

		var runtimeErr, bugErr error
		bugErr = safeInvoke(func() {
			runtimeErr = k.reconcileRevision(ctx, revision, &runningJobs)
		})
		if runtimeErr != nil {
			reportRuntimeError(ctx, StepReconcileRevisionsToAction, runtimeErr)
		}
		if bugErr != nil {
			reportThisIsABug(ctx,
				errors.Wrap(bugErr, "recovered from panic"))
		}
		if runtimeErr == nil && bugErr == nil {
			telemetryOK(ctx, StepReconcileRevisionsToAction)
		}
	}
}

func (k *kale) reconcileRevision(ctx context.Context, _revision *revisionAndChildren, runningJobs *int) error {
	proj := _revision.Parent.Project
	revision := _revision.ProjectRevision
	builders := _revision.Children.Builders
	runners := _revision.Children.Runners

	ctx = dlog.WithLogger(ctx, dlog.GetLogger(ctx).WithField("revision", revision.GetName()+"."+revision.GetNamespace()))

	var revisionPhase RevisionPhase
	// Decide what the phase of the revision should be, based on available evidence.
	//
	//     | runners | builders | phase                                                                   |
	//     |---------+----------+-------------------------------------------------------------------------|
	//     |       0 |        0 | Received or BuildQueued (depending on if there are job slots available) |
	//     |       0 |        1 | Building or BuildFailed or Deploying (depending on job.status)          |
	//     |       0 |       >1 | Building (should not happen)                                            |
	//     |       1 |        n | Deploying or DeployFailed or Deployed (depending on deployment.status)  |
	//     |      >1 |        n | Deploying (should not happen)                                           |
	//
	// Or, inverted:
	//
	//     | phase                | builders | runners  |
	//     |----------------------+----------+----------|
	//     | Received/BuildQueued | 0        | 0        |
	//     | Building             | 1+       | 0        |
	//     | BuildFailed          | 1 (fail) | 0        |
	//     | Deploying            | 1 (succ) | 0        |
	//     | Deploying            | n        | 1+       |
	//     | DeployFailed         | n        | 1 (fail) |
	//     | Deployed             | n        | 1 (succ) |
	//
	// It may help to think of "Deploying" as a synonym for "Build Completed Successfully".
	switch len(runners) {
	case 0:
		switch len(builders) {
		case 0:
			revisionPhase = RevisionPhase_Received
			if *runningJobs >= _revision.Parent.Parent.GetMaximumConcurrentBuilds() {
				revisionPhase = RevisionPhase_BuildQueued
			}
		case 1:
			if complete, _ := jobConditionMet(builders[0].Job, k8sTypesBatchV1.JobComplete, k8sTypesCoreV1.ConditionTrue); complete {
				// advance to next phase
				revisionPhase = RevisionPhase_Deploying
			} else if failed, _ := jobConditionMet(builders[0].Job, k8sTypesBatchV1.JobFailed, k8sTypesCoreV1.ConditionTrue); failed {
				telemetryErr(ctx, StepBuild,
					errors.Errorf("builder Job failed %d times", builders[0].Status.Failed))
				revisionPhase = RevisionPhase_BuildFailed
			} else {
				// keep waiting for one of the above to become true
				revisionPhase = RevisionPhase_Building
			}
		default: // >= 2
			// Something weird is going on here, don't let it advance.
			// This should fix itself after we do an applyAndPrune().
			revisionPhase = RevisionPhase_Building
		}
	case 1:
		dep := runners[0]
		if dep.Status.ObservedGeneration == dep.ObjectMeta.Generation &&
			(dep.Spec.Replicas == nil || dep.Status.UpdatedReplicas >= *dep.Spec.Replicas) &&
			dep.Status.UpdatedReplicas == dep.Status.Replicas &&
			dep.Status.AvailableReplicas == dep.Status.Replicas {
			// advance to next phase
			revisionPhase = RevisionPhase_Deployed
		} else {
			revisionPhase = RevisionPhase_Deploying
			for _, p := range dep.Children.Pods {
				if p.InCrashLoopBackOff() {
					revisionPhase = RevisionPhase_DeployFailed
					break
				}
			}
		}
	default: // >= 2
		// Something weird is going on here, don't let it
		// reach a terminal Deployed/DeployFailed phase.
		// This should fix itself after we do an applyAndPrune().
		revisionPhase = RevisionPhase_Deploying
	}
	// If the detected phase of the revision doesn't match what's in the revision.status, inform
	// both Kubernetes and GitHub of the change.
	if revisionPhase != revision.Status.Phase {
		revision.Status.Phase = revisionPhase
		uRevision := unstructureRevision(revision)
		_, err := k.revisionsGetter.Namespace(revision.GetNamespace()).UpdateStatus(uRevision, k8sTypesMetaV1.UpdateOptions{})
		if err != nil {
			reportRuntimeError(ctx, StepBackground, errors.Wrap(err, "update revision status in Kubernetes"))
		} else {
			k.knownChangedRevisions[revision.GetUID()] = revision.GetResourceVersion()
		}
		err = postStatus(ctx, fmt.Sprintf("https://api.github.com/repos/%s/statuses/%s", proj.Spec.GithubRepo, revision.Spec.Rev),
			GitHubStatus{
				State: map[RevisionPhase]string{
					RevisionPhase_Received:  "pending",
					RevisionPhase_Building:  "pending",
					RevisionPhase_Deploying: "pending",
					RevisionPhase_Deployed:  "success",

					RevisionPhase_BuildFailed:  "failure",
					RevisionPhase_DeployFailed: "failure",
				}[revisionPhase],
				TargetUrl: map[RevisionPhase]string{
					RevisionPhase_Received:     proj.BuildLogUrl(revision),
					RevisionPhase_Building:     proj.BuildLogUrl(revision),
					RevisionPhase_BuildFailed:  proj.BuildLogUrl(revision),
					RevisionPhase_Deploying:    proj.ServerLogUrl(revision),
					RevisionPhase_DeployFailed: proj.ServerLogUrl(revision),
					RevisionPhase_Deployed:     proj.PreviewUrl(revision),
				}[revisionPhase],
				Description: revisionPhase.String(),
				Context:     "aes/" + revision.Spec.Ref.String(),
			},
			proj.Spec.GithubToken)
		if err != nil {
			reportRuntimeError(ctx, StepBackground, errors.Wrap(err, "update revision status in GitHub"))
		}
	}

	var manifests []interface{}
	if len(_revision.Children.Builders) > 0 || *runningJobs < _revision.Parent.Parent.GetMaximumConcurrentBuilds() {
		manifests = append(manifests, k.calculateBuild(proj, revision)...)
		if len(_revision.Children.Builders) == 0 {
			*runningJobs++
		}
	}
	if revision.Status.Phase >= RevisionPhase_Deploying {
		telemetryOK(ctx, StepBuild)
		manifests = append(manifests, k.calculateRun(proj, revision)...)
	}
	if revision.Status.Phase == RevisionPhase_Deployed {
		telemetryOK(ctx, StepDeploy)
	}
	selectors := []string{
		GlobalLabelName + "==" + k.cfg.AmbassadorID,
		RevisionLabelName + "==" + string(revision.GetUID()),
	}
	if len(manifests) == 0 {
		return nil
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
			deleteResource("deployment.v1.apps", revision.GetName(), revision.GetNamespace())
		} else if regexp.MustCompile("(?s)The Job .* is invalid.* field is immutable").MatchString(err.Error()) {
			deleteResource("job.v1.batch", revision.GetName()+"-build", revision.GetNamespace())
		} else {
			reportRuntimeError(ctx, StepReconcileRevisionsToAction,
				errors.Wrap(err, "deploying ProjectRevision"))
		}
	}

	return nil
}

func (k *kale) calculateRun(proj *Project, revision *ProjectRevision) []interface{} {
	prefix := proj.Spec.Prefix
	if revision.Spec.IsPreview {
		// todo: figure out what is going on with /edge_stack/previews
		// not being routable
		prefix = "/.previews" + proj.Spec.Prefix + revision.Spec.Rev + "/"
	}
	return []interface{}{
		&Mapping{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "getambassador.io/v2",
				Kind:       "Mapping",
			},
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      revision.GetName(),
				Namespace: revision.GetNamespace(),
				OwnerReferences: []k8sTypesMetaV1.OwnerReference{
					{
						Controller:         boolPtr(true),
						BlockOwnerDeletion: boolPtr(true),
						Kind:               revision.TypeMeta.Kind,
						APIVersion:         revision.TypeMeta.APIVersion,
						Name:               revision.GetName(),
						UID:                revision.GetUID(),
					},
				},
				Labels: map[string]string{
					GlobalLabelName:   k.cfg.AmbassadorID,
					RevisionLabelName: string(revision.GetUID()),
				},
			},
			Spec: MappingSpec{
				AmbassadorID: aproTypesV2.AmbassadorID{k.cfg.AmbassadorID},
				Prefix:       prefix,
				Service:      revision.GetName(),
			},
		},
		&k8sTypesCoreV1.Service{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      revision.GetName(),
				Namespace: revision.GetNamespace(),
				OwnerReferences: []k8sTypesMetaV1.OwnerReference{
					{
						Controller:         boolPtr(true),
						BlockOwnerDeletion: boolPtr(true),
						Kind:               revision.TypeMeta.Kind,
						APIVersion:         revision.TypeMeta.APIVersion,
						Name:               revision.GetName(),
						UID:                revision.GetUID(),
					},
				},
				Labels: map[string]string{
					GlobalLabelName:   k.cfg.AmbassadorID,
					RevisionLabelName: string(revision.GetUID()),
				},
			},
			Spec: k8sTypesCoreV1.ServiceSpec{
				Selector: map[string]string{
					GlobalLabelName:   k.cfg.AmbassadorID,
					RevisionLabelName: string(revision.GetUID()),
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
				Name:      revision.GetName(),
				Namespace: revision.GetNamespace(),
				OwnerReferences: []k8sTypesMetaV1.OwnerReference{
					{
						Controller:         boolPtr(true),
						BlockOwnerDeletion: boolPtr(true),
						Kind:               revision.TypeMeta.Kind,
						APIVersion:         revision.TypeMeta.APIVersion,
						Name:               revision.GetName(),
						UID:                revision.GetUID(),
					},
				},
				Labels: map[string]string{
					GlobalLabelName:   k.cfg.AmbassadorID,
					RevisionLabelName: string(revision.GetUID()),
				},
			},
			Spec: k8sTypesAppsV1.DeploymentSpec{
				Selector: &k8sTypesMetaV1.LabelSelector{
					MatchLabels: map[string]string{
						GlobalLabelName:   k.cfg.AmbassadorID,
						RevisionLabelName: string(revision.GetUID()),
					},
				},
				Template: k8sTypesCoreV1.PodTemplateSpec{
					ObjectMeta: k8sTypesMetaV1.ObjectMeta{
						Labels: map[string]string{
							GlobalLabelName:     k.cfg.AmbassadorID,
							RevisionLabelName:   string(revision.GetUID()),
							ServicePodLabelName: "true",
						},
					},
					Spec: k8sTypesCoreV1.PodSpec{
						Containers: []k8sTypesCoreV1.Container{
							{
								Name:  "app",
								Image: "127.0.0.1:31000/" + proj.Namespace + "/" + proj.Name + ":" + revision.Spec.Rev,
								Env: []k8sTypesCoreV1.EnvVar{
									{Name: "AMB_PROJECT_PREVIEW", Value: strconv.FormatBool(revision.Spec.IsPreview)},
									{Name: "AMB_PROJECT_REPO", Value: proj.Spec.GithubRepo},
									{Name: "AMB_PROJECT_REF", Value: revision.Spec.Ref.String()},
									{Name: "AMB_PROJECT_REV", Value: revision.Spec.Rev},
								},
							},
						},
					},
				},
			},
		},
	}
}
