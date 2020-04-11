package kale

import (
	// standard library
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	// 3rd party
	"github.com/pkg/errors"
	libgit "gopkg.in/src-d/go-git.v4"
	libgitConfig "gopkg.in/src-d/go-git.v4/config"
	libgitPlumbing "gopkg.in/src-d/go-git.v4/plumbing"
	libgitPlumbingStorer "gopkg.in/src-d/go-git.v4/plumbing/storer"
	libgitHTTP "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	libgitStorageMemory "gopkg.in/src-d/go-git.v4/storage/memory"

	// 3rd party: k8s types
	k8sTypesBatchV1 "k8s.io/api/batch/v1"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypesUnstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sTypes "k8s.io/apimachinery/pkg/types"

	// 3rd party: k8s misc
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
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

// Turn an ordinary http handler function into a safe one that will
// automatically turn panics into a useful 500.
func ezHTTPHandler(handler func(*http.Request) httpResult) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var result httpResult
		err := safeInvoke(func() {
			result = handler(r)
		})
		if err != nil {
			reportThisIsABug(r.Context(), err)
			result = httpResult{
				status: http.StatusInternalServerError,
				body:   fmt.Sprintf("internal server error: this is a bug: %+v", err),
			}
		}

		if result.stream == nil {
			w.WriteHeader(result.status)
			io.WriteString(w, result.body)
		} else {
			err := safeInvoke(func() {
				streamingFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					result.stream(w)
				})).ServeHTTP(w, r)
			})
			if err != nil {
				reportRuntimeError(r.Context(), StepBackground, err)
			}
		}
	})
}

type httpResult struct {
	status int
	body   string
	stream func(w http.ResponseWriter)
}

func streamingFunc(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			_, ok := w.(http.Flusher)
			if !ok {
				// This is a bug--this should never be called with a
				// ResponseWriter that is not a Flusher.
				panicThisIsABug(errors.New("streaming unsupported"))
			}
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w, "\n") // just something to get readyState=1
			w.(http.Flusher).Flush()
			if r.Method == http.MethodHead {
				return
			}

			handler.ServeHTTP(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
}

// Post a json payload to a URL.
func postJSON(url string, payload interface{}, token string) (*http.Response, string, error) {
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return resp, string(body), nil
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
		return errors.Errorf("HTTP %d (%s)", resp.StatusCode, resp.Status)
	}

	return nil
}

// Post a status to the github API
func postStatus(ctx context.Context, url string, status GitHubStatus, token string) error {
	resp, body, err := postJSON(url, status, token)
	if err != nil {
		return errors.Wrap(err, "posting GitHub status")
	}
	if resp.StatusCode/100 != 2 {
		return errors.Errorf("posting GitHub status: HTTP %d (%s)\n%s",
			resp.StatusCode, resp.Status, body)
	}

	dlog.GetLogger(ctx).Debugf("posted GitHub status, got %s: %s, %q", resp.Status, url, status)
	return nil
}

type GitHubStatus struct {
	State       string `json:"state"`
	TargetUrl   string `json:"target_url"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

func postHook(repo string, callback string, token string) error {
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
	resp, body, err := postJSON(url, h, token)
	if err != nil {
		return errors.Wrap(err, "posting GitHub status")
	}
	if resp.StatusCode/100 == 2 {

		return nil
	}
	if resp.StatusCode == 422 && strings.Contains(body, "already exists") {
		return nil
	}

	return errors.Errorf("posting GitHub hook: HTTP %d (%s)\n%s",
		resp.StatusCode, resp.Status, body)
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

// The streamLogs helper sends logs from the kubernetes pods defined
// by the namespace and selector args down the supplied
// http.ResponseWriter using server side events.
func streamLogs(w http.ResponseWriter, r *http.Request, namespace, selector string) error {
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

	cmdCtx, cmdKill := context.WithCancel(r.Context())
	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)

	reader, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		id := ""
		data := line

		if parts := strings.SplitN(line, " ", 2); len(parts) == 2 {
			if _, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				id = parts[0]
				data = parts[1]
			}
		}

		var err error
		if id == "" {
			_, err = fmt.Fprintf(w, "data: %s\n\n", data)
		} else {
			_, err = fmt.Fprintf(w, "id: %s\ndata: %s\n\n", id, data)
		}
		if err != nil {
			cmdKill() // Stop the process.
			// Don't return--keep draining the process's output so it doesn't deadlock.
		}

		w.(http.Flusher).Flush()
	}
	err = scanner.Err()
	if err != nil {
		reportRuntimeError(r.Context(), "streamLogs:scanner", err)
	}

	err = cmd.Wait()
	if cmdCtx.Err() != nil {
		// If we bailed early because the client hung up (signalled either by
		// `r.Context()` being canceled, or by writes failing and us calling
		// `cmdKill()`; both of which set `cmdCtx.Err()`), then that's not really
		// an error.
		err = nil
	}
	if ee, isEE := err.(*exec.ExitError); isEE && ee.ExitCode() >= 0 {
		// The exit codes from `kubectl logs` don't appear to be meaningful;
		// discard the error, we'll need to rely on grepping stdout/stderr to
		// detect runtime errors from `kubectl logs`.  Note that we don't discard
		// the error if kubectl was terminated by a signal.
		err = nil
	}
	if err == nil {
		io.WriteString(w, "event: close\ndata:\n\n")
	}
	return err
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
		if gvk.Group == "" {
			gvk.Group = "core"
		}
		args = append(args, "--prune-whitelist="+gvk.Group+"/"+gvk.Version+"/"+gvk.Kind)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = strings.NewReader(yamlStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Errorf("%v\n%s", err, out)
	}
	return nil
}

func deleteResource(kind, name, namespace string) error {
	out, err := exec.Command("kubectl", "delete", "--namespace="+namespace, kind+"/"+name).CombinedOutput()
	if err != nil {
		return errors.Errorf("%v\n%s", err, out)
	}
	return nil
}

func boolPtr(v bool) *bool {
	return &v
}

func int32Ptr(v int32) *int32 {
	return &v
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
				Name:      proj.GetName() + "-" + ref.Hash().String(), // todo: better id
				Namespace: proj.GetNamespace(),
				OwnerReferences: []k8sTypesMetaV1.OwnerReference{
					{
						APIVersion:         "getambassador.io/v2",
						Controller:         boolPtr(true),
						BlockOwnerDeletion: boolPtr(true),
						Kind:               "Project",
						Name:               proj.GetName(),
						UID:                proj.GetUID(),
					},
				},
				Labels: map[string]string{
					GlobalLabelName:  k.cfg.AmbassadorID,
					ProjectLabelName: string(proj.GetUID()),
				},
			},
			Spec: ProjectCommitSpec{
				Project: k8sTypesCoreV1.LocalObjectReference{
					Name: proj.GetName(),
				},
				// Use the resolved ref.Name() instead of the original
				// refName, in order to resolve symbolic references; users
				// would rather see "master" instead of "HEAD".
				Ref:       ref.Name(),
				Rev:       ref.Hash().String(),
				IsPreview: refName != libgitPlumbing.HEAD,
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
			err := safeInvoke(func() { listener(w) })
			if err != nil {
				reportThisIsABug(ctx, err)
			}
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
			"metadata":   unstructureMetadata(&project.ObjectMeta),
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

// unstructureController returns a *k8sTypesUnstructured.Unstructured
// representation of an *ambassadorTypesV2.ProjectController.  There are 2
// reasons why we might want this:
//
//  1. For use with a k8sClientDynamic.Interface
//  2. For use as a k8sRuntime.Object
func unstructureController(controller *ProjectController) *k8sTypesUnstructured.Unstructured {
	return &k8sTypesUnstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "getambassador.io/v2",
			"kind":       "ProjectController",
			"metadata":   unstructureMetadata(&controller.ObjectMeta),
			"spec":       controller.Spec,
			"status":     controller.Status,
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
		panicThisIsABug(err)
	}

	if err := json.Unmarshal(bs, &metadata); err != nil {
		// 'bs' is valid JSON, we just generated it.  This
		// should never happen.
		panicThisIsABug(err)
	}

	return metadata
}

// jobConditionMet returns whether a Job's `.status.conditions` meets
// the given condition; in the same manor as `kubectl wait
// --for=CONDITION`.
//
// This is based on
// k8s.io/kubectl/pkg/cmd/wait.ConditionalWait.checkCondition()
// https://github.com/kubernetes/kubectl/blob/kubernetes-1.16.0/pkg/cmd/wait/wait.go#L418-L440
func jobConditionMet(obj *k8sTypesBatchV1.Job, condType k8sTypesBatchV1.JobConditionType, condStatus k8sTypesCoreV1.ConditionStatus) (bool, error) {
	for _, cond := range obj.Status.Conditions {
		if cond.Type != condType {
			continue
		}
		return cond.Status == condStatus, nil
	}
	return false, nil
}

func _logErr(ctx context.Context, err error) {
	var eventTarget interface {
		k8sRuntime.Object
		GetNamespace() string
	}
	var commitUID, projectUID k8sTypes.UID
	if commit := CtxGetCommit(ctx); commit != nil {
		err = errors.Wrapf(err, "ProjectCommit %s.%s",
			commit.GetName(), commit.GetNamespace())
		commitUID = commit.GetUID()
		eventTarget = unstructureCommit(commit)
	}
	if project := CtxGetProject(ctx); project != nil {
		err = errors.Wrapf(err, "Project %s.%s",
			project.GetName(), project.GetNamespace())
		projectUID = project.GetUID()
		if eventTarget == nil {
			eventTarget = unstructureProject(project)
		}
	}

	dlog.GetLogger(ctx).Errorf("%+v", err)

	// Because I've thought it before, and I know I'm going to think it again: Yes, it
	// might be the case that 'project' is set but 'iteration' isn't.  This can happen
	// if we'd like to report an error from the GitHub webhook.
	if CtxGetIteration(ctx) != nil {
		isNew := globalKale.addIterationError(err, projectUID, commitUID)
		if !isNew {
			eventTarget = nil
		}
	}
	if eventTarget == nil {
		globalKale.mu.RLock()
		if globalKale.webSnapshot != nil && len(globalKale.webSnapshot.Controllers) == 1 {
			for _, controller := range globalKale.webSnapshot.Controllers {
				eventTarget = unstructureController(controller.ProjectController)
			}
		}
		globalKale.mu.RUnlock()
	}
	if eventTarget != nil {
		globalKale.eventLogger.Namespace(eventTarget.GetNamespace()).Eventf(
			eventTarget,                     // InvolvedObject
			k8sTypesCoreV1.EventTypeWarning, // EventType
			"Err",      // Reason
			"%+v", err, // Message
		)
	} else {
		// It's important that we don't discard these, because
		// they're probably RBAC errors.
		t := k8sTypesMetaV1.Time{Time: time.Now()}
		globalKale.addWebError(&k8sTypesCoreV1.Event{
			Reason:         "err",
			Message:        fmt.Sprintf("%+v", err),
			FirstTimestamp: t,
			LastTimestamp:  t,
			Count:          1,
			Type:           k8sTypesCoreV1.EventTypeWarning,
		})
	}
}

func reportThisIsABug(ctx context.Context, err error) {
	err = errors.Wrap(err, "this is a bug: error")
	_logErr(ctx, err)
	telemetryErr(ctx, StepBug, err)
}

func reportRuntimeError(ctx context.Context, step string, err error) {
	err = errors.Wrap(err, "runtime error")
	_logErr(ctx, err)
	telemetryErr(ctx, step, err)
}

func panicThisIsABugContext(ctx context.Context, err error) {
	err = errors.Wrap(err, "this is a bug: panicking")
	_logErr(ctx, err)
	telemetryErr(ctx, StepBug, err)
	panic(err)
}

func panicThisIsABug(err error) {
	err = errors.Wrap(err, "this is a bug: panicking")
	panic(err)
}

func panicFlowControl(err error) {
	panic(err)
}

type iterationContextKey struct{}

func CtxWithIteration(ctx context.Context, itr uint64) context.Context {
	return context.WithValue(ctx, iterationContextKey{}, itr)
}

func CtxGetIteration(ctx context.Context) *uint64 {
	itrInterface := ctx.Value(iterationContextKey{})
	if itrInterface == nil {
		return nil
	}
	itr := itrInterface.(uint64)
	return &itr
}

type projectContextKey struct{}

func CtxWithProject(ctx context.Context, proj *Project) context.Context {
	return context.WithValue(ctx, projectContextKey{}, proj)
}

func CtxGetProject(ctx context.Context) *Project {
	projInterface := ctx.Value(projectContextKey{})
	if projInterface == nil {
		return nil
	}
	return projInterface.(*Project)
}

type commitContextKey struct{}

func CtxWithCommit(ctx context.Context, commit *ProjectCommit) context.Context {
	return context.WithValue(ctx, commitContextKey{}, commit)
}

func CtxGetCommit(ctx context.Context) *ProjectCommit {
	commitInterface := ctx.Value(commitContextKey{})
	if commitInterface == nil {
		return nil
	}
	return commitInterface.(*ProjectCommit)
}
