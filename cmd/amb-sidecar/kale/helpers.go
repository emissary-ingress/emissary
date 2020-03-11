package kale

import (
	// standard library
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	htemplate "html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"

	// 3rd party
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	ghttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"

	// 3rd party: k8s types
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypesUnstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	// 3rd party: k8s misc
	"sigs.k8s.io/yaml"

	// 1st party
	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/apro/lib/util"
)

// This file is a set of utility, convenience, and helper functions
// they assume it is ok to panic because they are being called
// underneath safeInvoke or equivalent.

// The safeInvoke function calls the code you pass to it and converts
// any panics into an error. This is useful for defining the boundary
// between "business logic" (aka code that handles a discrete
// interaction with a user or external system), and "system code", aka
// the rest of the code.
//
// We use this in two places:
//  1. handling http requests
//  2. responding to changes in kubernetes resources, aka the watch callback

func safeInvoke(code func()) (err error) {
	defer func() {
		err = util.PanicToError(recover())
	}()
	code()
	return
}

func safeInvoke1(fn func() error) (err error) {
	defer func() {
		if _err := util.PanicToError(recover()); _err != nil {
			err = _err
		}
	}()
	err = fn()
	return
}

// Turn an ordinary watch listener into one that will automatically
// turn panics into a useful log message.
func safeWatch(listener func(w *k8s.Watcher)) func(*k8s.Watcher) {
	return func(w *k8s.Watcher) {
		err := safeInvoke(func() {
			listener(w)
		})
		if err != nil {
			log.Printf("watch error: %+v", err)
		}
	}
}

// Turn an ordinary http handler function into a safe one that will
// automatically turn panics into a useful 500.
func safeHandleFunc(handler func(*http.Request) httpResult) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := safeInvoke(func() {
			result := handler(r)
			if result.stream == nil {
				w.WriteHeader(result.status)
				w.Write([]byte(result.body))
			} else {
				streamingFunc(func(w http.ResponseWriter, r *http.Request) {
					result.stream(w)
				})(w, r)
			}
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%+v\n", err)
		}
	}
}

type httpResult struct {
	status int
	body   string
	stream func(w http.ResponseWriter)
}

func streamingFunc(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			_, ok := w.(http.Flusher)
			if !ok {
				panic("streaming unsupported")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, "\n") // just something to get readyState=1
			w.(http.Flusher).Flush()
			if r.Method == http.MethodHead {
				return
			}

			handler(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// Post a json payload to a URL.
func postJSON(url string, payload interface{}, token string) (*http.Response, string) {
	buf, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return resp, string(body)
}

// Get a json payload from a URL.
func getJSON(url string, token string, target interface{}) *http.Response {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(target)
	if err != nil {
		panic(err)
	}
	return resp
}

// Post a status to the github API
func postStatus(url string, status GitHubStatus, token string) {
	resp, body := postJSON(url, status, token)

	if resp.Status[0] != '2' {
		panic(fmt.Errorf("error posting status: %s\n%s", resp.Status, string(body)))
	} else {
		log.Printf("posted status, got %s: %s, %q", resp.Status, url, status)
	}
}

type GitHubStatus struct {
	State       string `json:"state"`
	TargetUrl   string `json:"target_url"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

func postHook(repo string, callback string, token string) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/hooks", repo)
	h := hook{
		Name:   "web",
		Active: true,
		Events: []string{"push"},
		Config: hookConfig{
			Url:         callback,
			ContentType: "json",
		},
	}
	resp, body := postJSON(url, h, token)
	if resp.Status[0] == '2' {
		return
	}

	if resp.StatusCode == 422 && strings.Contains(body, "already exists") {
		return
	}

	panic(fmt.Errorf("%s: %s", resp.Status, body))
}

type hook struct {
	Name   string     `json:"name"`
	Active bool       `json:"active"`
	Events []string   `json:"events"`
	Config hookConfig `json:"config"`
}

type hookConfig struct {
	Url         string `json:"url"`
	ContentType string `json:"content_type"`
}

// Returns the pod logs for a build of the supplied commit.
func buildLogs(namespace, name, build string) string {
	selector := fmt.Sprintf("project=%s,build=%s", name, build)
	cmd := exec.Command("kubectl", "logs", "--tail=10000", "-n", namespace, "-l", selector)
	bytes, err := cmd.CombinedOutput()
	out := string(bytes)
	if err != nil {
		panic(fmt.Errorf("%w: %s", err, out))
	}
	return out
}

// Returns the pod logs for the supplied pod name.
func podLogs(name string) string {
	cmd := exec.Command("kubectl", "logs", name)
	bytes, err := cmd.CombinedOutput()
	out := string(bytes)
	if err != nil {
		panic(fmt.Errorf("%w: %s", err, out))
	}
	return out
}

// The streamLogs helper sends logs from the kubernetes pods defined
// by the namespace and selector args down the supplied
// http.ResponseWriter using server side events.
func streamLogs(w http.ResponseWriter, r *http.Request, namespace, selector string) {
	since := r.Header.Get("Last-Event-ID")
	args := []string{"logs", "--timestamps", "--tail=10000", "-f", "-n", namespace, "-l", selector}
	if since != "" {
		args = append(args, "--since-time", since)
	}
	cmd := exec.Command("kubectl", args...)

	rawReader, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	cmd.Stderr = cmd.Stdout
	reader := bufio.NewReader(rawReader)

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			line, err := reader.ReadString('\n')
			exit := false
			if err != nil {
				exit = true
				if err != io.EOF {
					log.Printf("warning: reading from kubectl logs: %v", err)
				}
			}

			if len(line) > 0 {
				parts := strings.SplitN(line, " ", 2)

				if len(parts) == 2 {
					_, tserr := time.Parse(time.RFC3339Nano, parts[0])
					if tserr != nil {
						_, err = fmt.Fprintf(w, "data: %s\n\n", line)
					} else {
						_, err = fmt.Fprintf(w, "id: %s\ndata: %s\n\n", parts[0], parts[1])
					}
				} else {
					_, err = fmt.Fprintf(w, "data: %s\n\n", line)
				}
				if err != nil {
					log.Printf("warning: writing to client: %v", err)
					cmd.Process.Kill()
					return
				}

				w.(http.Flusher).Flush()
			}

			if exit {
				fmt.Fprint(w, "event: close\ndata:\n\n")
				return
			}
		}
	}()

	err = cmd.Wait()
	if err != nil {
		log.Printf("warning: %v", err)
	}

	<-done
}

// Does a kubectl apply on the passed in yaml.
func applyStr(yaml string) (string, error) {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(yaml)
	out := strings.Builder{}
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	return out.String(), err
}

// Does a kubectl apply on the passed in yaml.
func applyObjs(objs []interface{}) (string, error) {
	var str string
	for _, obj := range objs {
		bs, err := yaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		str += "---\n" + string(bs)
	}
	return applyStr(str)
}

// Evaluates a golang template and returns the result.
func evalTemplate(text string, data interface{}) string {
	var out strings.Builder
	t := template.New("eval")
	t.Parse(text)
	err := t.Execute(&out, data)
	if err != nil {
		panic(err)
	}
	return out.String()
}

// Evaluates a golang template and returns the html-safe result.
func evalHtmlTemplate(text string, data interface{}) string {
	var out strings.Builder
	t := htemplate.New("eval")
	t.Parse(text)
	err := t.Execute(&out, data)
	if err != nil {
		panic(err)
	}
	return out.String()
}

func boolPtr(v bool) *bool {
	return &v
}

func deleteResource(kind, name, namespace string) error {
	out, err := exec.Command("kubectl", "delete", "--namespace="+namespace, kind+"/"+name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}
	return nil
}

type Deploy struct {
	Project Project
	Ref     *plumbing.Reference
	Pull    *Pull
}

func PrettyDeploys(deps []Deploy) string {
	var parts []string
	for _, d := range deps {
		parts = append(parts, fmt.Sprintf("%s => %s", d.Ref.Name().Short(), d.Ref.Hash()))
	}
	return strings.Join(parts, ", ")
}

type Pull struct {
	Number int
	Head   struct {
		Ref string
		Sha string
	}
	MergeSha string `json:"merge_commit_sha"`
}

func (d Deploy) IsBuilder(pod *k8sTypesCoreV1.Pod) bool {
	labels := pod.GetLabels()
	return (labels["project"] == d.Project.Metadata.Name &&
		pod.GetNamespace() == d.Project.Metadata.Namespace &&
		labels["build"] != "" &&
		labels["commit"] == d.Ref.Hash().String())
}

func (d Deploy) IsRunner(pod *k8sTypesCoreV1.Pod) bool {
	labels := pod.GetLabels()
	return (labels["project"] == d.Project.Metadata.Name &&
		pod.GetNamespace() == d.Project.Metadata.Namespace &&
		labels["build"] == "" &&
		labels["commit"] == d.Ref.Hash().String())
}

// GetDeploys does a `git ls-remote`, gets the listing of open GitHub
// pull-requests, and cross-references the two in order to decide
// which things we want to deploy.
func GetDeploys(project Project) []Deploy {
	repo := project.Spec.GithubRepo
	token := project.Spec.GithubToken
	refs := gitLsRemote(fmt.Sprintf("https://github.com/%s", repo), token, "refs/heads/*")
	var result []Pull
	resp := getJSON(fmt.Sprintf("https://api.github.com/repos/%s/pulls", repo), token, &result)
	if resp.StatusCode != 200 {
		panic(resp.Status)
	}

	pulls := make(map[string]Pull)

	for _, p := range result {
		pulls[p.Head.Sha] = p
	}

	var deploys []Deploy
	for _, ref := range refs {
		pull, ok := pulls[ref.Hash().String()]
		if ok {
			deploys = append(deploys, Deploy{project, ref, &pull})
		}
		if ref.Name().Short() == "master" {
			deploys = append(deploys, Deploy{project, ref, nil})
		}
	}

	return deploys
}

func gitLsRemote(repo, token string, specs ...string) []*plumbing.Reference {
	// Create the remote with repository URL
	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repo},
	})

	// We can then use every Remote functions to retrieve wanted information
	refs, err := rem.List(&git.ListOptions{
		Auth: &ghttp.BasicAuth{Username: token},
	})
	if err != nil {
		panic(err)
	}

	var result []*plumbing.Reference
	for _, ref := range refs {
		for _, spec := range specs {
			rs := config.RefSpec(fmt.Sprintf("%s:", spec))
			if rs.Match(ref.Name()) {
				result = append(result, ref)
			}
		}
	}
	return result
}

// WatchGroup is used to wait for multiple Watcher queries to all be
// ready before invoking a listener.
type WatchGroup struct {
	count int
	mu    sync.Mutex
}

func (wg *WatchGroup) Wrap(listener func(*k8s.Watcher)) func(*k8s.Watcher) {
	listener = safeWatch(listener)
	wg.count += 1
	invoked := false
	return func(w *k8s.Watcher) {
		wg.mu.Lock()
		defer wg.mu.Unlock()
		if !invoked {
			wg.count--
			invoked = true
		}
		if wg.count == 0 {
			listener(w)
		}
	}
}

// unstructureProject returns a *k8sTypesUnstructured.Unstructured
// representation of an *ambassadorTypesV2.Project.  There are 2 reasons
// why we might want this:
//
//  1. For use with a k8sClientDynamic.Interface
//  2. For use as a k8sRuntime.Object
func unstructureProject(project Project) *k8sTypesUnstructured.Unstructured {
	return &k8sTypesUnstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "getambassador.io/v2",
			"kind":       "Project",
			"metadata":   unstructureMetadata(&project.Metadata),
			"spec":       project.Spec,
			"status":     project.Status,
		},
	}
}

// unstructureMetadata marshals a *k8sTypesMetaV1.ObjectMeta for use
// in a `*k8sTypesUnstructured.Unstructured`.
//
// `*k8sTypesUnstructured.Unstructured` requires that the "metadata"
// field be a `map[string]interface{}`.  Going through JSON is the
// easiest way to get from a typed `*k8sTypesMetaV1.ObjectMeta` to an
// untyped `map[string]interface{}`.  Yes, it's gross and stupid.
func unstructureMetadata(in *k8sTypesMetaV1.ObjectMeta) map[string]interface{} {
	var metadata map[string]interface{}
	bs, err := json.Marshal(in)
	if err != nil {
		// 'in' is a valid object.  This should never happen.
		panic(err)
	}

	if err := json.Unmarshal(bs, &metadata); err != nil {
		// 'bs' is valid JSON, we just generated it.  This
		// should never happen.
		panic(err)
	}

	return metadata
}
