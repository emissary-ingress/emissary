package kale

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/apro/cmd/amb-sidecar/group"
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

func Setup(group *group.Group, httpHandler lyftserver.DebugHTTPHandler, info *k8s.KubeInfo) {
	k := NewKale()

	group.Go("kale", func(hardCtx, softCtx context.Context, cfg types.Config, l dlog.Logger) error {
		client, err := k8s.NewClient(info)
		if err != nil {
			// this is non fatal (mostly just to facilitate local dev); don't `return err`
			l.Errorln("not watching Project resources:", fmt.Errorf("k8s.NewClient: %w", err))
			return nil
		}

		w := client.Watcher()
		wg := &WatchGroup{}

		err = w.WatchQuery(k8s.Query{Kind: "projects.getambassador.io"},
			safeWatch(wg.Wrap(k.reconcileConsistently)))

		if err != nil {
			// this is non fatal (mostly just to facilitate local dev); don't `return err`
			l.Errorln("not watching Project resources:", fmt.Errorf("WatchQuery(projects): %w", err))
			return nil
		}

		err = w.WatchQuery(k8s.Query{Kind: "pod", LabelSelector: "kale"},
			safeWatch(wg.Wrap(k.reconcileConsistently)))

		if err != nil {
			// this is non fatal (mostly just to facilitate local dev); don't `return err`
			l.Errorln("not watching Project resources:", fmt.Errorf("WatchQuery(pods): %w", err))
			return nil
		}

		w.Start()
		select {
		case <-softCtx.Done():
			w.Stop()
			w.Wait()
		case <-hardCtx.Done():
			w.Stop()
		}
		return nil
	})

	// todo: lock down all these endpoints with auth
	handler := http.StripPrefix("/edge_stack_ui/edge_stack", http.HandlerFunc(safeHandleFunc(k.dispatch))).ServeHTTP
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/projects", "kale projects api", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/githook/", "kale githook", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/logs/", "kale logs api", handler)
	// todo: this is just temporary, we will consolidate these sprawling endpoints later
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/slogs/", "kale server logs api", handler)
}

// A kale contains the global state for the controller/webhook. We
// assume there is only one copy of the controller running in the
// cluster, so this is global to the entire cluster.
func NewKale() *kale {
	return &kale{
		Projects: make(map[string]Project),
		Pods:     make(PodMap),
		Deploys:  make(DeployMap),
	}
}

type kale struct {
	cfg      types.Config
	Projects map[string]Project
	Pods     PodMap
	Deploys  DeployMap
}

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

type DeployMap map[string][]Deploy

func (k *kale) reconcileConsistently(w *k8s.Watcher) {
	k.reconcileProjects(w)

	var pods []*k8sTypesCoreV1.Pod
	if err := mapstructure.Convert(w.List("pod"), &pods); err != nil {
		panic(err)
	}
	k.reconcilePods(pods)
}

func (k *kale) reconcileProjects(w *k8s.Watcher) {
	projects := make(map[string]Project)
	for _, rsrc := range w.List("projects.getambassador.io") {
		pr := Project{}
		err := rsrc.Decode(&pr)
		if err != nil {
			log.Printf(err.Error())
			continue
		}

		key := pr.Key()
		hookUrl := fmt.Sprintf("https://%s/edge_stack/api/githook/%s", pr.Spec.Host, key)
		postHook(pr.Spec.GithubRepo, hookUrl, pr.Spec.GithubToken)
		projects[key] = pr
		k.Deploys[key] = GetDeploys(pr)
	}
	k.Projects = projects
	// todo: we should have better data structures so this isn't so awkward
	for key, _ := range k.Pods {
		_, ok := k.Projects[key]
		if !ok {
			delete(k.Pods, key)
		}
	}
	for key, _ := range k.Deploys {
		_, ok := k.Projects[key]
		if !ok {
			delete(k.Deploys, key)
		}
	}
}

type Project struct {
	Metadata struct {
		Name      string       `json:"name"`
		Namespace string       `json:"namespace"`
		UID       k8sTypes.UID `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		Host        string `json:"host"`
		Prefix      string `json:"prefix"`
		GithubRepo  string `json:"githubRepo"`
		GithubToken string `json:"githubToken"` // todo: make this a secret ref
	} `json:"spec"`
}

func (p Project) Key() string {
	return p.Metadata.Namespace + "/" + p.Metadata.Name
}

func (k *kale) GetProject(namespace, name string) (Project, bool) {
	p, ok := k.Projects[namespace+"/"+name]
	return p, ok
}

func (p Project) LogUrl(build string) string {
	return fmt.Sprintf("https://%s/edge_stack/api/logs/%s/%s/%s", p.Spec.Host, p.Metadata.Namespace, p.Metadata.Name,
		build)
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
	case "slogs":
		fallthrough
	case "logs":
		namespace := parts[2]
		name := parts[3]
		build := parts[4]
		var selector string
		if parts[1] == "slogs" {
			// todo: we need to make our pod labels and
			// service selectors distinguish between build
			// and deploy pods
			selector = fmt.Sprintf("project=%s,commit=%s,!build", name, build)
		} else {
			selector = fmt.Sprintf("project=%s,build=%s", name, build)
		}
		return httpResult{
			stream: func(w http.ResponseWriter) {
				streamLogs(w, r, namespace, selector)
			},
		}
	}
	return httpResult{status: 400, body: "bad request"}
}

// Returns the a JSON string with all the data for the root of the
// UI. This is a map of all the projects plus nested data as
// appropriate.
func (k *kale) projectsJSON() string {
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

		m := make(map[string]interface{})
		m["project"] = proj
		m["pods"] = pods

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
	proj, ok := k.Projects[key]
	if !ok {
		return httpResult{status: 404, body: fmt.Sprintf("no such project %s", key)}
	}

	// todo: the webhook is totally asynchronous... we basically
	// need to work based on polling and treat the webhook as a
	// hint to increase the frequency of polling
	time.Sleep(5 * time.Second)

	old, ok := k.Deploys[key]
	if ok {
		log.Printf("DEPLOYS BEFORE: %s", PrettyDeploys(old))
	}
	deploys := GetDeploys(proj)
	k.Deploys[key] = deploys
	log.Printf("DEPLOYS AFTER: %s", PrettyDeploys(deploys))
	k.reconcile()

	return httpResult{status: 200, body: ""}
}

type Push struct {
	Ref  string
	Head struct {
		Id string
	} `json:"head_commit"`
	Repository struct {
		GitUrl      string `json:"git_url"`
		StatusesUrl string `json:"statuses_url"`
	}
}

func (k *kale) startBuild(proj Project, buildID, ref, commit string) (string, error) {
	// Note: If the kaniko destination is set to the full service name
	// (registry.ambassador.svc.cluster.local), then we can't seem to push
	// to the no matter how we tweak the settings. I assume this is due to
	// some special handling of .local domains somewhere.
	//
	// todo: the ambassador namespace is hardcoded below in the registry
	//       we to which we push

	pod := &k8sTypesCoreV1.Pod{
		TypeMeta: k8sTypesMetaV1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: k8sTypesMetaV1.ObjectMeta{
			Name:      proj.Metadata.Name + "-build-" + buildID,
			Namespace: proj.Metadata.Namespace,
			Annotations: map[string]string{
				"statusesUrl": "https://api.github.com/repos/" + proj.Spec.GithubRepo + "/statuses/" + commit,
			},
			Labels: map[string]string{
				"kale":    "0.0",
				"project": proj.Metadata.Name,
				"commit":  commit,
				"build":   buildID,
			},
			OwnerReferences: []k8sTypesMetaV1.OwnerReference{
				{
					APIVersion:         "getambassador.io/v2",
					Controller:         boolPtr(true),
					BlockOwnerDeletion: boolPtr(true),
					Kind:               "Project",
					Name:               proj.Metadata.Name,
					UID:                proj.Metadata.UID,
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
						"--context=git://github.com/" + proj.Spec.GithubRepo + ".git#" + ref,
						"--destination=registry.ambassador/" + commit,
					},
				},
			},
			RestartPolicy: k8sTypesCoreV1.RestartPolicyNever,
		},
	}

	manifests := []interface{}{pod}

	out, err := applyObjs(manifests)
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, out)
	}

	k.Pods.addPod(pod)

	return string(out), nil
}

func (k *kale) updatePods(pods []*k8sTypesCoreV1.Pod) {
	pm := make(PodMap)

	for _, pod := range pods {
		pm.addPod(pod)
	}

	k.Pods = pm
}

func (k *kale) reconcilePods(pods []*k8sTypesCoreV1.Pod) {
	k.updatePods(pods)
	k.reconcile()
}

func (k *kale) pods() []*k8sTypesCoreV1.Pod {
	var result []*k8sTypesCoreV1.Pod
	for _, pods := range k.Pods {
		for _, pod := range pods {
			result = append(result, pod)
		}
	}
	return result
}

func (k *kale) desiredDeploys() []Deploy {
	var result []Deploy
	for _, deps := range k.Deploys {
		for _, dep := range deps {
			result = append(result, dep)
		}
	}
	return result
}

func (k *kale) IsDesired(pod *k8sTypesCoreV1.Pod) bool {
	labels := pod.GetLabels()
	key := fmt.Sprintf("%s/%s", pod.GetNamespace(), labels["project"])
	commit := labels["commit"]
	deps, ok := k.Deploys[key]
	if ok {
		for _, dep := range deps {
			if dep.Ref.Hash().String() == commit {
				return true
			}
		}
	}
	return false
}

func (k *kale) reconcile() {
	deploys := k.desiredDeploys()
	pods := k.pods()
	for _, dep := range deploys {
		var deployBuilders []*k8sTypesCoreV1.Pod
		var deployRunners []*k8sTypesCoreV1.Pod
		for _, pod := range pods {
			if dep.IsBuilder(pod) {
				deployBuilders = append(deployBuilders, pod)
			} else if dep.IsRunner(pod) {
				deployRunners = append(deployRunners, pod)
			}
		}

		err := safeInvoke1(func() error { return k.reconcileDeploy(dep, deployBuilders, deployRunners) })
		if err != nil {
			log.Printf("ERROR: %v", err)
		}
	}
	for _, pod := range pods {
		if !k.IsDesired(pod) {
			err := deleteResource("pod", pod.GetName(), pod.GetNamespace())
			if err != nil {
				log.Printf("ERROR: %v", err)
			}
		}
	}
}

func (k *kale) reconcileDeploy(dep Deploy, builders, runners []*k8sTypesCoreV1.Pod) error {
	proj := dep.Project

	switch len(builders) {
	case 0:
		if len(runners) == 0 {
			//buildID := fmt.Sprintf("%d", time.Now().Unix()) // todo: better id
			buildID := dep.Ref.Hash().String()
			out, err := k.startBuild(proj, buildID, dep.Ref.Name().String(), dep.Ref.Hash().String())
			if err != nil {
				log.Printf("OUTPUT: %s", out)
				log.Printf("ERROR: %v", err)
				postStatus(fmt.Sprintf("https://api.github.com/repos/%s/statuses/%s", proj.Spec.GithubRepo, dep.Ref.Hash().String()),
					GitHubStatus{
						State:       "error",
						TargetUrl:   fmt.Sprintf("http://%s/edge_stack/", proj.Spec.Host),
						Description: err.Error(),
						Context:     "aes",
					},
					proj.Spec.GithubToken)
			} else {
				postStatus(fmt.Sprintf("https://api.github.com/repos/%s/statuses/%s", proj.Spec.GithubRepo, dep.Ref.Hash().String()),
					GitHubStatus{
						State:       "pending",
						TargetUrl:   proj.LogUrl(buildID),
						Description: "build started",
						Context:     "aes",
					},
					proj.Spec.GithubToken)
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

		phase := builder.Status.Phase
		qname := builder.GetName() + "." + builder.GetNamespace()
		log.Println("BUILDER", qname, phase)

		if len(runners) == 0 { // don't bother with the builder if there's already a runner
			statusesUrl := builder.GetAnnotations()["statusesUrl"]
			logUrl := proj.LogUrl(builder.GetLabels()["build"])
			switch phase {
			case k8sTypesCoreV1.PodFailed:
				log.Printf(podLogs(builder.GetName()))
				postStatus(statusesUrl, GitHubStatus{
					State:       "failure",
					TargetUrl:   logUrl,
					Description: string(phase),
					Context:     "aes",
				},
					proj.Spec.GithubToken)
			case k8sTypesCoreV1.PodSucceeded:
				_, err := k.startRun(proj, builder.GetLabels()["commit"])
				if err != nil {
					msg := fmt.Sprintf("ERROR: %v", err)
					log.Print(msg)
					if len(msg) > 140 {
						msg = msg[len(msg)-140:]
					}
					// todo: need a way to get log output to github
					postStatus(statusesUrl,
						GitHubStatus{
							State:       "error",
							TargetUrl:   "http://asdf",
							Description: msg,
							Context:     "aes",
						},
						proj.Spec.GithubToken)
					return err
				} else {
					// todo: fake url
					postStatus(statusesUrl,
						GitHubStatus{
							State:       "success",
							TargetUrl:   "http://asdf",
							Description: string(phase),
							Context:     "aes",
						},
						proj.Spec.GithubToken)
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

func (k *kale) startRun(proj Project, commit string) (string, error) {
	pod := &k8sTypesCoreV1.Pod{
		TypeMeta: k8sTypesMetaV1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: k8sTypesMetaV1.ObjectMeta{
			Name:      proj.Metadata.Name + "-" + commit,
			Namespace: proj.Metadata.Namespace,
			OwnerReferences: []k8sTypesMetaV1.OwnerReference{
				{
					APIVersion:         "getambassador.io/v2",
					Controller:         boolPtr(true),
					BlockOwnerDeletion: boolPtr(true),
					Kind:               "Project",
					Name:               proj.Metadata.Name,
					UID:                proj.Metadata.UID,
				},
			},
			Labels: map[string]string{
				"kale":    "0.0",
				"project": proj.Metadata.Name,
				"commit":  commit,
			},
		},
		Spec: k8sTypesCoreV1.PodSpec{
			Containers: []k8sTypesCoreV1.Container{
				{
					Name:  "app",
					Image: "127.0.0.1:31000/" + commit,
				},
			},
		},
	}
	manifests := []interface{}{
		map[string]interface{}{
			"apiVersion": "getambassador.io/v2",
			"kind":       "Mapping",
			"metadata": k8sTypesMetaV1.ObjectMeta{
				Name:      proj.Metadata.Name + "-" + commit,
				Namespace: proj.Metadata.Namespace,
				OwnerReferences: []k8sTypesMetaV1.OwnerReference{
					{
						APIVersion:         "getambassador.io/v2",
						Controller:         boolPtr(true),
						BlockOwnerDeletion: boolPtr(true),
						Kind:               "Project",
						Name:               proj.Metadata.Name,
						UID:                proj.Metadata.UID,
					},
				},
				Labels: map[string]string{
					"kale": "0.0",
				},
			},
			"spec": map[string]interface{}{
				"prefix":  "/.previews/" + proj.Spec.Prefix + "/" + commit + "/",
				"service": proj.Spec.Prefix + "-" + commit,
			},
		},
		&k8sTypesCoreV1.Service{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      proj.Metadata.Name + "-" + commit,
				Namespace: proj.Metadata.Namespace,
				OwnerReferences: []k8sTypesMetaV1.OwnerReference{
					{
						APIVersion:         "getambassador.io/v2",
						Controller:         boolPtr(true),
						BlockOwnerDeletion: boolPtr(true),
						Kind:               "Project",
						Name:               proj.Metadata.Name,
						UID:                proj.Metadata.UID,
					},
				},
				Labels: map[string]string{
					"kale": "0.0",
				},
			},
			Spec: k8sTypesCoreV1.ServiceSpec{
				Selector: map[string]string{
					"project": proj.Metadata.Name,
					"commit":  commit,
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
		pod,
	}

	out, err := applyObjs(manifests)
	if err != nil {
		return "", fmt.Errorf("%w\n%s", err, out)
	}

	k.Pods.addPod(pod)

	return string(out), nil
}
