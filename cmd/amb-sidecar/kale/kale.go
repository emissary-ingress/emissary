package kale

import (
	// standard library
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	// 3rd party
	libgitPlumbing "gopkg.in/src-d/go-git.v4/plumbing"

	// 3rd/1st party: k8s types
	aproTypesV2 "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// 3rd party: k8s clients
	k8sClientDynamic "k8s.io/client-go/dynamic"

	// 3rd party: k8s misc
	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"

	// 1st party
	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/apro/cmd/amb-sidecar/group"
	"github.com/datawire/apro/cmd/amb-sidecar/k8s/leaderelection"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/mapstructure"
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
	GlobalLabelName  = "kale"                              // cfg.AmbassadorID
	ProjectLabelName = "projects.getambassador.io/project" // proj.GetName()+"."+proj.GetNamespace()
	CommitLabelName  = "projects.getambassador.io/commit"  // commit.GetName()+"."+commit.GetNamespace()
)

func Setup(group *group.Group, httpHandler lyftserver.DebugHTTPHandler, info *k8s.KubeInfo, dynamicClient k8sClientDynamic.Interface) {
	k := NewKale(dynamicClient)

	upstreamWorker := make(chan Snapshot)
	downstreamWorker := make(chan Snapshot)
	go coalesce(upstreamWorker, downstreamWorker)

	upstreamWebUI := make(chan Snapshot)
	downstreamWebUI := make(chan Snapshot)
	go coalesce(upstreamWebUI, downstreamWebUI)

	group.Go("kale_watcher", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
		k.cfg = cfg
		softCtx = dlog.WithLogger(softCtx, l)

		client, err := k8s.NewClient(info)
		if err != nil {
			// this is non fatal (mostly just to facilitate local dev); don't `return err`
			l.Errorln("not watching Project resources:", fmt.Errorf("k8s.NewClient: %w", err))
			return nil
		}

		w := client.Watcher()
		var wg WatchGroup

		handler := func(w *k8s.Watcher) {
			snapshot := UntypedSnapshot{
				Pods:     w.List("pods."),
				Projects: w.List("projects.getambassador.io"),
				Commits:  w.List("projectcommits.getambassador.io"),
			}
			// More so out of paranoia than in response to an actual issue: repeat the
			// .TypedAndFiltered() call for each stream, so that they share no pointers.
			upstreamWorker <- snapshot.TypedAndFiltered(softCtx, cfg.AmbassadorID)
			upstreamWebUI <- snapshot.TypedAndFiltered(softCtx, cfg.AmbassadorID)
		}

		labelSelector := GlobalLabelName + "=" + cfg.AmbassadorID
		queries := []k8s.Query{
			{Kind: "projects.getambassador.io"},
			{Kind: "projectcommits.getambassador.io"},
			{Kind: "pods.", LabelSelector: labelSelector},
		}

		for _, query := range queries {
			err := w.WatchQuery(query, wg.Wrap(softCtx, handler))
			if err != nil {
				// this is non fatal (mostly just to facilitate local dev); don't `return err`
				l.Errorf("not watching %q resources: %v",
					query.Kind,
					fmt.Errorf("WatchQuery(%#v, ...): %w", query, err))
				return nil
			}
		}

		w.Start()
		go func() {
			<-softCtx.Done()
			w.Stop()
		}()
		w.Wait()
		close(upstreamWorker)
		close(upstreamWebUI)
		return nil
	})

	group.Go("kale_worker", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
		softCtx = dlog.WithLogger(softCtx, l)

		err := leaderelection.RunAsSingleton(softCtx, cfg, info, "kale", 15*time.Second, func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case snapshot, ok := <-downstreamWorker:
					if !ok {
						return
					}
					k.reconcile(ctx, snapshot)
				}
			}
		})
		if err != nil {
			// make this non-fatal
			l.Errorln("failed to participate in kale leader election, kale is disabled:", err)
		}
		return nil
	})

	group.Go("kale_webui_update", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
		for {
			select {
			case <-softCtx.Done():
				return nil
			case snapshot, ok := <-downstreamWebUI:
				if !ok {
					return nil
				}
				k.updateInternalState(snapshot)
			}
		}
	})

	// todo: lock down all these endpoints with auth
	handler := http.StripPrefix("/edge_stack_ui/edge_stack", http.HandlerFunc(safeHandleFunc(k.dispatch))).ServeHTTP
	// todo: this is just temporary, we will consolidate these sprawling endpoints later
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/projects", "kale projects api", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/githook/", "kale githook", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/logs/", "kale logs api", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/slogs/", "kale server logs api", handler)
}

type UntypedSnapshot struct {
	Pods     []k8s.Resource
	Projects []k8s.Resource
	Commits  []k8s.Resource
}

type Snapshot struct {
	Pods     []*k8sTypesCoreV1.Pod
	Projects []*Project
	Commits  []*ProjectCommit
}

func (in UntypedSnapshot) TypedAndFiltered(ctx context.Context, ambassadorID string) Snapshot {
	log := dlog.GetLogger(ctx)
	var out Snapshot

	// For built-in resource types, it is appropriate to a panic
	// because it's a bug; the api-server only gives us valid
	// resources, so if we fail to parse them, it's a bug in how
	// we're parsing.  For the same reason, it's "safe" to do this
	// all at once, because we don't need to do individual
	// validation, because they're all valid.
	if err := mapstructure.Convert(in.Pods, &out.Pods); err != nil {
		panic(fmt.Errorf("Pods: %q", err))
	}

	// However, for our CRDs, because the api-server can't
	// validate that CRs are valid the way that it can for
	// built-in Resources, we have to safely deal with the
	// possibility that any individual resource is invalid, and
	// not let that affect the others.
	for _, inProj := range in.Projects {
		var outProj *Project
		if err := mapstructure.Convert(inProj, &outProj); err != nil {
			log.Println(fmt.Errorf("Project: %w", err))
			continue
		}
		out.Projects = append(out.Projects, outProj)
	}
	for _, inCommit := range in.Commits {
		var outCommit *ProjectCommit
		if err := mapstructure.Convert(inCommit, &outCommit); err != nil {
			log.Println(fmt.Errorf("Commit: %w", err))
			continue
		}
		out.Commits = append(out.Commits, outCommit)
	}

	return out
}

func coalesce(upstream <-chan Snapshot, downstream chan<- Snapshot) {
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
		projectsGetter: dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "projects"}),
		commitsGetter:  dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v2", Resource: "projectcommits"}),
		Projects:       make(map[string]*Project),
		Pods:           make(PodMap),
	}
}

type kale struct {
	cfg types.Config

	projectsGetter k8sClientDynamic.NamespaceableResourceInterface
	commitsGetter  k8sClientDynamic.NamespaceableResourceInterface

	mu       sync.RWMutex
	Projects map[string]*Project
	Pods     PodMap
	Commits  map[string][]*ProjectCommit
}

// map["projectNamespace/projectName"]map["podName"]*Pod
type PodMap map[string]map[string]*k8sTypesCoreV1.Pod

func (pm PodMap) addPod(pod *k8sTypesCoreV1.Pod) {
	projName := pod.GetLabels()["project"]
	if projName != "" {
		projKey := fmt.Sprintf("%s/%s", pod.GetNamespace(), projName)
		projPods, ok := pm[projKey]
		if !ok {
			projPods = make(map[string]*k8sTypesCoreV1.Pod)
			pm[projKey] = projPods
		}
		// todo: exclude terminating pods so that we don't double delete them
		projPods[pod.GetName()] = pod
	}
}

func (k *kale) reconcile(ctx context.Context, snapshot Snapshot) {
	k.reconcileGitHub(snapshot.Projects)
	k.reconcileCluster(ctx, snapshot)
}

func (k *kale) updateInternalState(snapshot Snapshot) {
	// map["projectNamespace/projectName"]*Project
	projects := make(map[string]*Project)
	for _, proj := range snapshot.Projects {
		projects[proj.Key()] = proj
	}

	// map["projectNamespace/projectName"]map["podName"]*Pod
	pods := make(PodMap)
	for _, pod := range snapshot.Pods {
		pods.addPod(pod)
	}

	// map["projectNamespace/projectName"][]*ProjectCommit
	commits := make(map[string][]*ProjectCommit)
	for _, commit := range snapshot.Commits {
		key := commit.GetNamespace() + "/" + commit.Spec.Project.Name
		commits[key] = append(commits[key], commit)
	}

	k.mu.Lock()
	k.Projects = projects
	k.Pods = pods
	k.Commits = commits
	k.mu.Unlock()
}

func (k *kale) reconcileGitHub(projects []*Project) {
	for _, pr := range projects {
		postHook(pr.Spec.GithubRepo,
			fmt.Sprintf("https://%s/edge_stack/api/githook/%s", pr.Spec.Host, pr.Key()),
			pr.Spec.GithubToken)
	}
}

// This is our dispatcher for everything under /api/. This looks at
// the URL and based on it figures out an appropriate handler to
// call. All the real business logic for the web API is in the methods
// this calls.
func (k *kale) dispatch(r *http.Request) httpResult {
	parts := strings.Split(r.URL.Path[1:], "/")
	if parts[0] != "api" {
		panic("this shouldn't happen")
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
		selectors := []string{
			GlobalLabelName + "==" + k.cfg.AmbassadorID,
			CommitLabelName + "==" + commitQName,
		}
		if logType == "slogs" {
			selectors = append(selectors, "!build")
		} else {
			selectors = append(selectors, "build")
		}
		return httpResult{
			stream: func(w http.ResponseWriter) {
				streamLogs(w, r, namespace, strings.Join(selectors, ","))
			},
		}
	}
	return httpResult{status: 400, body: "bad request"}
}

// Returns the a JSON string with all the data for the root of the
// UI. This is a map of all the projects plus nested data as
// appropriate.
func (k *kale) projectsJSON() string {
	k.mu.RLock()
	defer k.mu.RUnlock()

	var keys []string
	for key, _ := range k.Projects {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	results := make([]interface{}, 0)

	for _, key := range keys {
		proj := k.Projects[key]
		pods, ok := k.Pods[key]
		if !ok {
			pods = make(map[string]*k8sTypesCoreV1.Pod)
		}
		commits, ok := k.Commits[key]
		if !ok {
			commits = make([]*ProjectCommit, 0)
		}

		m := make(map[string]interface{})
		m["project"] = proj
		m["pods"] = pods
		m["commits"] = commits

		results = append(results, m)
	}

	bytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(bytes) + "\n"
}

// Handle Push events from the github API.
func (k *kale) handlePush(r *http.Request, key string) httpResult {
	log := dlog.GetLogger(r.Context())

	k.mu.RLock()
	proj, ok := k.Projects[key]
	k.mu.RUnlock()
	if !ok {
		return httpResult{status: 404, body: fmt.Sprintf("no such project %s", key)}
	}

	var push Push
	if err := json.NewDecoder(r.Body).Decode(&push); err != nil {
		log.Printf("WEBHOOK PARSE ERROR: %v", err)
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
		// Bump the project's .Status, to trigger a rectify via
		// Kubernetes.  We do this instead of just poking the right
		// bits in memory because we might not be the elected leader.
		proj.Status.LastPush = time.Now()
		uProj := unstructureProject(proj)
		_, err := k.projectsGetter.Namespace(proj.Metadata.Namespace).UpdateStatus(uProj, k8sTypesMetaV1.UpdateOptions{})
		if err != nil {
			log.Println("update project status:", err)
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

func (k *kale) startBuild(proj *Project, commit *ProjectCommit) (string, error) {
	// Note: If the kaniko destination is set to the full service name
	// (registry.ambassador.svc.cluster.local), then we can't seem to push
	// to the no matter how we tweak the settings. I assume this is due to
	// some special handling of .local domains somewhere.
	//
	// todo: the ambassador namespace is hardcoded below in the registry
	//       we to which we push

	manifests := []interface{}{
		&k8sTypesCoreV1.Pod{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
			},
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      commit.GetName() + "-build",
				Namespace: commit.GetNamespace(),
				Labels: map[string]string{
					GlobalLabelName: k.cfg.AmbassadorID,
					CommitLabelName: commit.GetName() + "." + commit.GetNamespace(),
					"project":       commit.Spec.Project.Name,
					"build":         commit.GetName(),
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
			Spec: k8sTypesCoreV1.PodSpec{
				Containers: []k8sTypesCoreV1.Container{
					{
						Name:  "kaniko",
						Image: "gcr.io/kaniko-project/executor:v0.16.0",
						Args: []string{
							"--cache=true",
							"--skip-tls-verify",
							"--skip-tls-verify-pull",
							"--skip-tls-verify-registry",
							"--dockerfile=Dockerfile",
							"--context=git://github.com/" + proj.Spec.GithubRepo + ".git#" + commit.Spec.Ref.String(),
							"--destination=registry.ambassador/" + commit.Spec.Rev,
						},
					},
				},
				RestartPolicy: k8sTypesCoreV1.RestartPolicyNever,
			},
		},
	}

	out, err := applyObjs(manifests)
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, out)
	}

	return string(out), nil
}

func (k *kale) IsDesired(commits []*ProjectCommit, pod *k8sTypesCoreV1.Pod) bool {
	for _, commit := range commits {
		if pod.GetLabels()[CommitLabelName] == commit.GetName()+"."+commit.GetNamespace() {
			return true
		}
	}
	return false
}

func (k *kale) reconcileCluster(ctx context.Context, snapshot Snapshot) {
	log := dlog.GetLogger(ctx)

	// reconcile commits
	for _, proj := range snapshot.Projects {
		commitManifests, err := k.calculateCommits(proj)
		if err != nil {
			continue
		}
		selectors := []string{
			GlobalLabelName + "==" + k.cfg.AmbassadorID,
			ProjectLabelName + "==" + proj.Metadata.Name + "." + proj.Metadata.Namespace,
		}
		err = applyAndPrune(
			strings.Join(selectors, ","),
			[]k8sSchema.GroupVersionKind{
				{Group: "getambassador.io", Version: "v2", Kind: "ProjectCommit"},
			},
			commitManifests)
		if err != nil {
			log.Errorf("updating ProjectCommits for Project %q.%q: %v",
				proj.Metadata.Name, proj.Metadata.Namespace,
				err)
		}
	}

	// reconcile pods
	for _, commit := range snapshot.Commits {
		var project *Project
		for _, proj := range snapshot.Projects {
			if proj.Metadata.Namespace == commit.GetNamespace() &&
				proj.Metadata.Name == commit.Spec.Project.Name {
				project = proj
			}
		}
		var commitBuilders []*k8sTypesCoreV1.Pod
		var commitRunners []*k8sTypesCoreV1.Pod
		for _, pod := range snapshot.Pods {
			if pod.GetLabels()[CommitLabelName] == commit.GetName()+"."+commit.GetNamespace() {
				if pod.GetLabels()["build"] != "" {
					commitBuilders = append(commitBuilders, pod)
				} else {
					commitRunners = append(commitRunners, pod)
				}
			}
		}

		err := safeInvoke1(func() error { return k.reconcileCommit(ctx, project, commit, commitBuilders, commitRunners) })
		if err != nil {
			log.Printf("ERROR: %v", err)
		}
	}
	for _, pod := range snapshot.Pods {
		if !k.IsDesired(snapshot.Commits, pod) {
			err := deleteResource("pod", pod.GetName(), pod.GetNamespace())
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
		}
	}
}

func (k *kale) reconcileCommit(ctx context.Context, proj *Project, commit *ProjectCommit, builders, runners []*k8sTypesCoreV1.Pod) error {
	log := dlog.GetLogger(ctx)

	var commitPhase CommitPhase
	// Decide what the phase of the commit should be, based on available evidence
	if len(runners) == 0 {
		if len(builders) != 1 {
			commitPhase = CommitPhase_Received
		} else {
			switch builders[0].Status.Phase {
			case k8sTypesCoreV1.PodFailed:
				commitPhase = CommitPhase_BuildFailed
			case k8sTypesCoreV1.PodSucceeded:
				commitPhase = CommitPhase_Deploying
			default:
				commitPhase = CommitPhase_Building
			}
		}
	} else {
		if len(runners) != 1 {
			commitPhase = CommitPhase_Deploying
		} else {
			switch runners[0].Status.Phase {
			case k8sTypesCoreV1.PodFailed, k8sTypesCoreV1.PodSucceeded:
				commitPhase = CommitPhase_DeployFailed
			case k8sTypesCoreV1.PodRunning:
				commitPhase = CommitPhase_Deployed
			default:
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
		postStatus(ctx, fmt.Sprintf("https://api.github.com/repos/%s/statuses/%s", proj.Spec.GithubRepo, commit.Spec.Rev),
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

	}

	switch len(builders) {
	case 0:
		if len(runners) == 0 {
			_, err := k.startBuild(proj, commit)
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
		}
	case 1:
		// do nothing
	default:
		// TODO: more intelligently pick which pod gets to survive
		for _, pod := range builders[1:] {
			err := deleteResource("pod", pod.GetName(), pod.GetNamespace())
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
		}
	}

	if len(builders) > 0 {
		builder := builders[0]
		// TODO: validate that the builder looks how we expect

		if len(runners) == 0 { // don't bother with the builder if there's already a runner
			switch builder.Status.Phase {
			case k8sTypesCoreV1.PodSucceeded:
				out, err := k.startRun(proj, commit)
				if err != nil {
					log.Printf("ERROR: %v: %s", err, out)
				}
			}
		}
	}

	if len(runners) > 0 {
		// TODO: more intelligently pick which pod gets to survive
		for _, pod := range runners[1:] {
			err := deleteResource("pod", pod.GetName(), pod.GetNamespace())
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
		}

		runner := runners[0]
		// TODO: validate that the runner looks how we expect

		phase := runner.Status.Phase
		qname := runner.GetName() + "." + runner.GetNamespace()
		log.Println("RUNNER", qname, phase)
	}

	return nil
}

func (k *kale) startRun(proj *Project, commit *ProjectCommit) (string, error) {
	manifests := []interface{}{
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
					CommitLabelName: commit.GetName() + "." + commit.GetNamespace(),
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
					CommitLabelName: commit.GetName() + "." + commit.GetNamespace(),
				},
			},
			Spec: k8sTypesCoreV1.ServiceSpec{
				Selector: map[string]string{
					GlobalLabelName: k.cfg.AmbassadorID,
					CommitLabelName: commit.GetName() + "." + commit.GetNamespace(),
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
		&k8sTypesCoreV1.Pod{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Pod",
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
					CommitLabelName: commit.GetName() + "." + commit.GetNamespace(),
					"project":       commit.Spec.Project.Name,
				},
			},
			Spec: k8sTypesCoreV1.PodSpec{
				Containers: []k8sTypesCoreV1.Container{
					{
						Name:  "app",
						Image: "127.0.0.1:31000/" + commit.Spec.Rev,
					},
				},
			},
		},
	}

	out, err := applyObjs(manifests)
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, out)
	}

	return string(out), nil
}
