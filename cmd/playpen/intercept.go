package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/datawire/teleproxy/pkg/supervisor"
	"github.com/pkg/errors"
)

// InterceptInfo tracks one intercept operation
type InterceptInfo struct {
	Name       string // Name of the intercept (user/logging)
	Deployment string // Name of the deployment being intercepted
	Patterns   map[string]string
	TargetHost string
	TargetPort int
}

// Acquire an intercept from the traffic manager
func (ii *InterceptInfo) Acquire(p *supervisor.Process, tm *TrafficManager) (int, error) {
	reqPatterns := make([]map[string]string, 0, len(ii.Patterns))
	for header, regex := range ii.Patterns {
		pattern := map[string]string{"name": header, "regex_match": regex}
		reqPatterns = append(reqPatterns, pattern)
	}
	request := map[string]interface{}{
		"name":     ii.Name,
		"patterns": reqPatterns,
	}
	reqData, err := json.Marshal(request)
	if err != nil {
		return 0, err
	}
	result, code, err := tm.request("POST", "intercept/"+ii.Deployment, reqData)
	if err != nil {
		return 0, errors.Wrap(err, "acquire intercept")
	}
	if code == 404 {
		return 0, fmt.Errorf("Deployment %q is not known to the traffic manager", ii.Deployment)
	}
	if !(200 <= code && code <= 299) {
		return 0, fmt.Errorf("acquire intercept: %s: %s", http.StatusText(code), result)
	}
	port, err := strconv.Atoi(result)
	if err != nil {
		return 0, errors.Wrapf(err, "bad port number from traffic manager: %q", result)
	}
	return port, nil
}

// Retain the given intercept. This likely needs to be called every
// five seconds or so.
func (ii *InterceptInfo) Retain(p *supervisor.Process, tm *TrafficManager, port int) error {
	data := []byte(fmt.Sprintf("{\"port\": %d}", port))
	result, code, err := tm.request("POST", "intercept/"+ii.Deployment, data)
	if err != nil {
		return errors.Wrap(err, "retain intercept")
	}
	if !(200 <= code && code <= 299) {
		return fmt.Errorf("retain intercept: %s: %s", http.StatusText(code), result)
	}
	return nil
}

// Release the given intercept.
func (ii *InterceptInfo) Release(p *supervisor.Process, tm *TrafficManager, port int) error {
	data := []byte(fmt.Sprintf("%d", port))
	result, code, err := tm.request("DELETE", "intercept/"+ii.Deployment, data)
	if err != nil {
		return errors.Wrap(err, "release intercept")
	}
	if !(200 <= code && code <= 299) {
		return fmt.Errorf("release intercept: %s: %s", http.StatusText(code), result)
	}
	return nil
}

// ListIntercepts lists active intercepts
func (d *Daemon) ListIntercepts(p *supervisor.Process, out *Emitter) error {
	for idx, cept := range d.intercepts {
		ii := cept.ii
		out.Printf("%4d. %s\n", idx, ii.Name)
		out.Printf("      Intercepting requests to %s when\n", ii.Deployment)
		for k, v := range ii.Patterns {
			out.Printf("      - %s: %s\n", k, v)
		}
		out.Printf("      and redirecting them to %s:%d\n", ii.TargetHost, ii.TargetPort)
	}
	if len(d.intercepts) == 0 {
		out.Println("No intercepts")
	}
	return nil
}

// AddIntercept adds one intercept
func (d *Daemon) AddIntercept(p *supervisor.Process, out *Emitter, ii *InterceptInfo) error {
	for _, cept := range d.intercepts {
		if cept.ii.Name == ii.Name {
			out.Printf("Intercept with name %q already exists\n", ii.Name)
			out.SendExit(1)
			return nil
		}
	}
	cept, err := MakeIntercept(p, d.trafficMgr, ii)
	if err != nil {
		out.Printf("Failed to establish intercept: %s\n", err)
		out.SendExit(1)
		return nil
	}
	d.intercepts = append(d.intercepts, cept)
	out.Printf("Added intercept %q\n", ii.Name)
	return nil
}

// RemoveIntercept removes one intercept by name
func (d *Daemon) RemoveIntercept(p *supervisor.Process, out *Emitter, name string) error {
	for idx, cept := range d.intercepts {
		if cept.ii.Name == name {
			d.intercepts = append(d.intercepts[:idx], d.intercepts[idx+1:]...)
			out.Printf("Removed intercept %q\n", name)
			if err := cept.Close(); err != nil {
				out.Printf("Error while removing intercept: %v\n", err)
				out.SendExit(1)
			}

			return nil
		}
	}
	out.Printf("Intercept named %q not found\n", name)
	out.SendExit(1)
	return nil
}

// Intercept is a Resource handle that represents a live intercept
type Intercept struct {
	ii   *InterceptInfo
	tm   *TrafficManager
	port int
	ResourceBase
}

// MakeIntercept acquires an intercept and returns a Resource handle
// for it
func MakeIntercept(p *supervisor.Process, tm *TrafficManager, ii *InterceptInfo) (cept *Intercept, err error) {
	port, err := ii.Acquire(p, tm)
	if err != nil {
		return
	}
	cept = &Intercept{ii: ii, tm: tm, port: port}
	cept.doCheck = cept.check
	cept.doQuit = cept.quit
	cept.setup(p.Supervisor(), ii.Name)
	return
}

func (cept *Intercept) check(p *supervisor.Process) error {
	return cept.ii.Retain(p, cept.tm, cept.port)
}

func (cept *Intercept) quit(p *supervisor.Process) error {
	cept.done = true
	return cept.ii.Release(p, cept.tm, cept.port)
}
