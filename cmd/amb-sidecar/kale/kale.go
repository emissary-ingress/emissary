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

	k8sTypesCoreV1 "k8s.io/api/core/v1"

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
		w, err := k8s.NewWatcher(info)
		if err != nil {
			return err
		}

		err = w.WatchQuery(k8s.Query{Kind: "projects.getambassador.io"}, safeWatch(k.reconcileProjects))

		if err != nil {
			return err
		}

		err = w.WatchQuery(k8s.Query{
			Kind:          "pod",
			LabelSelector: "kale",
		}, safeWatch(func(w *k8s.Watcher) {
			var pods []*k8sTypesCoreV1.Pod
			if err := mapstructure.Convert(w.List("pod"), &pods); err != nil {
				panic(err)
			}
			k.reconcilePods(pods)
		}))

		if err != nil {
			return err
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

	handler := http.StripPrefix("/edge_stack_ui/edge_stack", http.HandlerFunc(safeHandleFunc(k.dispatch))).ServeHTTP
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/projects", "kale projects api", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/githook/", "kale githook", handler)
	httpHandler.AddEndpoint("/edge_stack_ui/edge_stack/api/logs/", "kale logs api", handler)

}

// This contains the global state for the controller/webhook. We
// assume there is only one copy of the controller running in the
// cluster, so this is global to the entire cluster.

type kale struct {
	cfg      types.Config
	Projects map[string]Project
	Pods     map[string]map[string]*k8sTypesCoreV1.Pod
	done     map[string]bool
}

func NewKale() *kale {
	return &kale{
		Projects: make(map[string]Project),
		Pods:     make(map[string]map[string]*k8sTypesCoreV1.Pod),
		done:     make(map[string]bool),
	}
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
	}
	k.Projects = projects
}

type Project struct {
	Metadata struct {
		Name      string
		Namespace string
		Uid       string
	} `json:"metadata"`
	Spec struct {
		Host        string `json:"host"`
		Prefix      string `json:"prefix"`
		GithubRepo  string `json:"githubRepo"`
		GithubToken string `json:"-"`
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
			return httpResult{200, "pong"}
		case "push":
			return k.handlePush(r, strings.Join(parts[2:], "/"))
		default:
			return httpResult{500, fmt.Sprintf("don't know how to handle %s events", eventType)}
		}
	case "projects":
		return httpResult{200, k.projectsJSON()}
	case "logs":
		// todo: this only does build logs, need to add deploy logs
		return httpResult{200, buildLogs(parts[2], parts[3], parts[4])}
	}
	return httpResult{400, "bad request"}
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
		return httpResult{404, fmt.Sprintf("no such project %s", key)}
	}

	dec := json.NewDecoder(r.Body)
	// todo: better id
	build := fmt.Sprintf("%d", time.Now().Unix())
	env := BuildEnv{Build: build, Project: proj}
	dec.Decode(&env.Push)
	log.Printf("PUSH: %q", env)
	out, err := apply(evalTemplate(BUILD, env))
	if err != nil {
		log.Printf("ERROR: %s\n%s", err.Error(), out)
		postStatus(env.StatusUrl(),
			GitHubStatus{
				State:       "error",
				TargetUrl:   fmt.Sprintf("http://%s/edge_stack/", proj.Spec.Host),
				Description: err.Error(),
				Context:     "aes",
			},
			proj.Spec.GithubToken)
	} else {
		postStatus(env.StatusUrl(),
			GitHubStatus{
				State:       "pending",
				TargetUrl:   proj.LogUrl(build),
				Description: "build started",
				Context:     "aes",
			},
			proj.Spec.GithubToken)
	}
	return httpResult{200, out}
}

type BuildEnv struct {
	Build   string
	Push    Push
	Project Project
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

func (e BuildEnv) StatusUrl() string {
	return strings.ReplaceAll(e.Push.Repository.StatusesUrl, "{sha}", e.Push.Head.Id)
}

// Note: If the kaniko destination is set to the full service name
// (registry.ambassador.svc.cluster.local), then we can't seem to push
// to the no matter how we tweak the settings. I assume this is due to
// some special handling of .local domains somewhere.
//
// todo: the ambassador namespace is hardcoded below in the registry
//       we to which we push

var BUILD = `
---
apiVersion: v1
kind: Pod
metadata:
  name: {{.Project.Metadata.Name}}-build-{{.Build}}
  namespace: {{.Project.Metadata.Namespace}}
  annotations:
    statusesUrl: "{{.StatusUrl}}"
  labels:
    kale: "0.0"
    project: "{{.Project.Metadata.Name}}"
    commit: "{{.Push.Head.Id}}"
    build: "{{.Build}}"
  ownerReferences:
    - apiVersion: getambassador.io/v2
      controller: true
      blockOwnerDeletion: true
      kind: Project
      name: {{.Project.Metadata.Name}}
      uid: {{.Project.Metadata.Uid}}
spec:
  containers:
  - name: kaniko
    image: gcr.io/kaniko-project/executor:v0.16.0
    args: ["--cache=true",
           "-v=debug",
           "--skip-tls-verify",
           "--skip-tls-verify-pull",
           "--skip-tls-verify-registry",
           "--dockerfile=Dockerfile",
           "--context={{.Push.Repository.GitUrl}}#{{.Push.Ref}}",
           "--destination=registry.ambassador/{{.Push.Head.Id}}"]
  restartPolicy: Never
`

func (k *kale) reconcilePods(pods []*k8sTypesCoreV1.Pod) {
	cutoff := time.Now().Add(-5 * 60 * time.Second)
	// todo: The system/business boundary needs some work here.
	// Specifically the logic for chunking up the snapshot into
	// the units we want to work on and processing each chunk is
	// mixed up here. We want to separate that out so that the
	// system can provide a guarantee that if processing one chunk
	// fails, it won't interfere with processing other chunks.
	for _, pod := range pods {
		phase := pod.Status.Phase
		qname := pod.GetName() + "." + pod.GetNamespace()
		key := qname + "." + string(phase)
		_, ok := k.done[key]
		if ok {
			continue
		}
		log.Println("POD", qname, phase)

		projName := pod.GetLabels()["project"]
		namespace := pod.GetNamespace()

		projKey := fmt.Sprintf("%s/%s", namespace, projName)
		projPods, ok := k.Pods[projKey]
		if !ok {
			projPods = make(map[string]*k8sTypesCoreV1.Pod)
			k.Pods[projKey] = projPods
		}
		projPods[pod.GetName()] = pod

		// Check when the pod was created. If it's old enough, we don't bother with it.
		if pod.GetCreationTimestamp().Time.Before(cutoff) {
			k.done[key] = true
			continue
		}

		proj, ok := k.GetProject(namespace, projName)
		if !ok {
			log.Printf("no such project: %s", projName)
			continue
		}

		statusesUrl := pod.GetAnnotations()["statusesUrl"]
		logUrl := proj.LogUrl(pod.GetLabels()["build"])
		switch phase {
		case k8sTypesCoreV1.PodFailed:
			log.Printf(podLogs(pod.GetName()))
			postStatus(statusesUrl, GitHubStatus{
				State:       "failure",
				TargetUrl:   logUrl,
				Description: string(phase),
				Context:     "aes",
			},
				proj.Spec.GithubToken)
		case k8sTypesCoreV1.PodSucceeded:
			run := evalTemplate(RUN, RunEnv{Project: proj, Commit: pod.GetLabels()["commit"]})
			out, err := apply(run)
			if err != nil {
				msg := fmt.Sprintf("ERROR: %s\n%s", err.Error(), out)
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
		k.done[key] = true
	}
}

type RunEnv struct {
	Project Project
	Commit  string
}

var RUN = `
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {{.Project.Metadata.Name}}-{{.Commit}}
  namespace: {{.Project.Metadata.Namespace}}
  ownerReferences:
    - apiVersion: getambassador.io/v2
      controller: true
      blockOwnerDeletion: true
      kind: Project
      name: {{.Project.Metadata.Name}}
      uid: {{.Project.Metadata.Uid}}
  labels:
    kale: "0.0"
spec:
  prefix: /kale/previews/{{.Project.Spec.Prefix}}/{{.Commit}}/
  service: {{.Project.Spec.Prefix}}-{{.Commit}}
---
apiVersion: v1
kind: Service
metadata:
  name: {{.Project.Metadata.Name}}-{{.Commit}}
  namespace: {{.Project.Metadata.Namespace}}
  ownerReferences:
    - apiVersion: getambassador.io/v2
      controller: true
      blockOwnerDeletion: true
      kind: Project
      name: {{.Project.Metadata.Name}}
      uid: {{.Project.Metadata.Uid}}
  labels:
    kale: "0.0"
spec:
  selector:
    project: {{.Project.Metadata.Name}}
    commit: {{.Commit}}
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
---
apiVersion: v1
kind: Pod
metadata:
  name: {{.Project.Metadata.Name}}-{{.Commit}}
  namespace: {{.Project.Metadata.Namespace}}
  ownerReferences:
    - apiVersion: getambassador.io/v2
      controller: true
      blockOwnerDeletion: true
      kind: Project
      name: {{.Project.Metadata.Name}}
      uid: {{.Project.Metadata.Uid}}
  labels:
    kale: "0.0"
    project: {{.Project.Metadata.Name}}
    commit: {{.Commit}}
spec:
  containers:
  - name: app
    image: 127.0.0.1:31000/{{.Commit}}
`
