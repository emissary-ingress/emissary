package kale

import (
	"bytes"
	"encoding/json"
	"fmt"
	htemplate "html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"text/template"

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
			w.WriteHeader(result.status)
			w.Write([]byte(result.body))
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

// Post a status to the github API
func postStatus(url string, status Status, token string) {
	resp, body := postJSON(url, status, token)

	if resp.Status[0] != '2' {
		panic(fmt.Errorf("error posting status: %s\n%s", resp.Status, string(body)))
	} else {
		log.Printf("posted status, got %s: %s, %q", resp.Status, url, status)
	}
}

type Status struct {
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

// Does a kubectl apply on the passed in yaml.
func apply(yaml string) (string, error) {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(yaml)
	out := strings.Builder{}
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	return out.String(), err
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
