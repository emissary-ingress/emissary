package kale

import (
	// standard library
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	htemplate "html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"

	// 3rd party
	libgit "gopkg.in/src-d/go-git.v4"
	libgitConfig "gopkg.in/src-d/go-git.v4/config"
	libgitPlumbing "gopkg.in/src-d/go-git.v4/plumbing"
	libgitPlumbingStorer "gopkg.in/src-d/go-git.v4/plumbing/storer"
	libgitHTTP "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	libgitStorageMemory "gopkg.in/src-d/go-git.v4/storage/memory"

	// 3rd party: k8s types
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypesUnstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	// 3rd party: k8s misc
	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	// 1st party
	"github.com/datawire/ambassador/pkg/dlog"
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
func safeWatch(ctx context.Context, listener func(w *k8s.Watcher)) func(*k8s.Watcher) {
	return func(w *k8s.Watcher) {
		err := safeInvoke(func() {
			listener(w)
		})
		if err != nil {
			dlog.GetLogger(ctx).Printf("watch error: %+v", err)
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
func getJSON(url string, authToken string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d (%s)", resp.StatusCode, resp.Status)
	}

	return nil
}

// Post a status to the github API
func postStatus(ctx context.Context, url string, status GitHubStatus, token string) {
	resp, body := postJSON(url, status, token)

	if resp.Status[0] != '2' {
		panic(fmt.Errorf("error posting status: %s\n%s", resp.Status, string(body)))
	} else {
		dlog.GetLogger(ctx).Printf("posted status, got %s: %s, %q", resp.Status, url, status)
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
	log := dlog.GetLogger(r.Context())

	since := r.Header.Get("Last-Event-ID")

	args := []string{
		"kubectl",
		"--namespace=" + namespace,
		"logs",
		"--timestamps",
		"--tail=10000",
		"--follow",
		"--selector=" + selector,
	}
	if since != "" {
		args = append(args, "--since-time", since)
	}

	cmd := exec.Command(args[0], args[1:]...)

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

func applyAndPrune(labelSelector string, types []k8sSchema.GroupVersionKind, objs []interface{}) error {
	var yamlStr string
	for _, obj := range objs {
		bs, err := yaml.Marshal(obj)
		if err != nil {
			return err
		}
		yamlStr += "---\n" + string(bs)
	}

	args := []string{"kubectl", "apply",
		"--filename=-",
		"--prune",
		"--selector=" + labelSelector,
	}
	for _, gvk := range types {
		args = append(args, "--prune-whitelist="+gvk.Group+"/"+gvk.Version+"/"+gvk.Kind)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = strings.NewReader(yamlStr)
	out := strings.Builder{}
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("%w\n%s", err, out.String())
	}
	return nil
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

type Pull struct {
	Number int `json:"number"`
	Head   struct {
		Ref  string `json:"ref"`
		Sha  string `json:"sha"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
}

// calculateCommits does a `git ls-remote`, gets the listing of open GitHub
// pull-requests, and cross-references the two in order to decide
// which things we want to deploy.
func (k *kale) calculateCommits(proj *Project) ([]interface{}, error) {
	repo := proj.Spec.GithubRepo
	token := proj.Spec.GithubToken

	// Ask the server for a collection of references
	refs, err := gitLsRemote(fmt.Sprintf("https://github.com/%s", repo), token)
	if err != nil {
		return nil, err
	}

	// Ask the server for a list of open PRs
	var openPulls []Pull
	if err := getJSON(fmt.Sprintf("https://api.github.com/repos/%s/pulls", repo), token, &openPulls); err != nil {
		return nil, err
	}

	// Which refnames to deploy
	var deployRefNames []libgitPlumbing.ReferenceName
	// Always deploy HEAD (which is a symbolic ref, usually to
	// "refs/heads/master").
	deployRefNames = append(deployRefNames,
		libgitPlumbing.HEAD)
	// And also deploy any open first-party PRs.
	for _, pull := range openPulls {
		if strings.EqualFold(pull.Head.Repo.FullName, repo) {
			deployRefNames = append(deployRefNames,
				libgitPlumbing.ReferenceName(fmt.Sprintf("refs/pull/%d/head", pull.Number)))
		}
	}

	// Resolve all of those refNames, and generate ProjectCommit objects for them.
	var commits []interface{}
	for _, refName := range deployRefNames {
		// Use libgitPlumbing.ReferenceName() instead of refs.Reference() (or even having
		// gitLsRemote return a simple slice of refs) in order to resolve refs recursively.
		// This is important, because HEAD is always a symbolic reference.
		ref, err := libgitPlumbingStorer.ResolveReference(refs, refName)
		if err != nil {
			continue
		}
		commits = append(commits, &ProjectCommit{
			TypeMeta: k8sTypesMetaV1.TypeMeta{
				APIVersion: "getambassador.io/v2",
				Kind:       "ProjectCommit",
			},
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      proj.Metadata.Name + "-" + ref.Hash().String(), // todo: better id
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
					GlobalLabelName:  k.cfg.AmbassadorID,
					ProjectLabelName: proj.Metadata.Name + "." + proj.Metadata.Namespace,
				},
			},
			Spec: ProjectCommitSpec{
				Project: k8sTypesCoreV1.LocalObjectReference{
					Name: proj.Metadata.Name,
				},
				// Use the resolved ref.Name() instead of the original
				// refName, in order to resolve symbolic references; users
				// would rather see "master" instead of "HEAD".
				Ref: ref.Name(),
				Rev: ref.Hash().String(),
			},
		})
	}
	return commits, nil
}

func gitLsRemote(repoURL, authToken string) (libgitPlumbingStorer.ReferenceStorer, error) {
	remote := libgit.NewRemote(nil, &libgitConfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})

	refs, err := remote.List(&libgit.ListOptions{
		Auth: &libgitHTTP.BasicAuth{Username: authToken},
	})
	if err != nil {
		return nil, err
	}

	// Instead of returning 'refs' as a simple slice, pack the
	// result in to a ReferenceStorer, so that we can easily
	// resolve recursive refs by using storer.ResolveReference().
	storage := libgitStorageMemory.NewStorage()
	for _, ref := range refs {
		if err := storage.SetReference(ref); err != nil {
			return nil, err
		}
	}
	return storage, nil
}

func gitResolveRef(repoURL, authToken string, refname libgitPlumbing.ReferenceName) (*libgitPlumbing.Reference, error) {
	storage, err := gitLsRemote(repoURL, authToken)
	if err != nil {
		return nil, err
	}
	return libgitPlumbingStorer.ResolveReference(storage, refname)
}

// WatchGroup is used to wait for multiple Watcher queries to all be
// ready before invoking a listener.
type WatchGroup struct {
	count int
	mu    sync.Mutex
}

func (wg *WatchGroup) Wrap(ctx context.Context, listener func(*k8s.Watcher)) func(*k8s.Watcher) {
	listener = safeWatch(ctx, listener)
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
func unstructureProject(project *Project) *k8sTypesUnstructured.Unstructured {
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

// unstructureCommit returns a *k8sTypesUnstructured.Unstructured
// representation of an *ambassadorTypesV2.ProjectCommit.  There are 2
// reasons why we might want this:
//
//  1. For use with a k8sClientDynamic.Interface
//  2. For use as a k8sRuntime.Object
func unstructureCommit(commit *ProjectCommit) *k8sTypesUnstructured.Unstructured {
	return &k8sTypesUnstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "getambassador.io/v2",
			"kind":       "ProjectCommit",
			"metadata":   unstructureMetadata(&commit.ObjectMeta),
			"spec":       commit.Spec,
			"status":     commit.Status,
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
