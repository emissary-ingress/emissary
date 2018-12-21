package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jcuga/golongpoll"
)

// PatternInfo represents one Envoy header regex_match
type PatternInfo struct {
	Name       string `json:"name"`
	RegexMatch string `json:"regex_match"`
}

// InterceptInfo tracks one intercept operation
type InterceptInfo struct {
	Name        string
	Patterns    []PatternInfo
	Port        int
	LastQueryAt time.Time
}

func (intercept InterceptInfo) String() string {
	return fmt.Sprintf("%s -> %d (%s)", intercept.Patterns, intercept.Port, intercept.Name)
}

// DeploymentInfo tracks everything the proxy knows about one deployment
type DeploymentInfo struct {
	Intercepts  []*InterceptInfo
	LastQueryAt time.Time
}

// ProxyState holds the overall state of the proxy
type ProxyState struct {
	mutex       sync.Mutex
	FreePorts   []int
	Deployments map[string]*DeploymentInfo
	manager     *golongpoll.LongpollManager
}

func newProxyState(manager *golongpoll.LongpollManager) *ProxyState {
	const (
		portOffset = 9000
		numPorts   = 16
	)
	res := ProxyState{
		FreePorts:   make([]int, numPorts),
		Deployments: make(map[string]*DeploymentInfo),
		manager:     manager,
	}
	for idx := range res.FreePorts {
		res.FreePorts[idx] = portOffset + idx
	}
	return &res
}

// Dump the current state of the proxy
func (state *ProxyState) handleState(w http.ResponseWriter, r *http.Request) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	result, err := json.Marshal(state)
	if err != nil {
		panic(err)
	}
	w.Write([]byte(result))
}

func (state *ProxyState) publish(deployment string) error {
	dInfo, known := state.Deployments[deployment]
	if !known {
		return fmt.Errorf("Unknown deployment: %s", deployment)
	}
	return state.manager.Publish(deployment, dInfo.Intercepts)
}

// Track that a deployment exists, handle long poll to get routes
func (state *ProxyState) handleRoutes(w http.ResponseWriter, r *http.Request) {
	state.mutex.Lock()
	locked := true
	defer func() {
		if locked {
			state.mutex.Unlock()
		}
	}()

	deployment := r.URL.Query().Get("category")
	if len(deployment) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing required URL param: category"))
		return
	}
	dInfo := state.Deployments[deployment]
	if dInfo == nil {
		dInfo = &DeploymentInfo{
			Intercepts:  make([]*InterceptInfo, 0),
			LastQueryAt: time.Now(),
		}
		state.Deployments[deployment] = dInfo // FIXME need to garbage collect
		err := state.publish(deployment)
		if err != nil {
			panic(err)
		}
	} else {
		dInfo.LastQueryAt = time.Now()
	}
	state.mutex.Unlock()
	locked = false
	state.manager.SubscriptionHandler(w, r)
}

// Add an intercept to a deployment, return a port number
func (state *ProxyState) startIntercept(deployment, name string, patterns []PatternInfo) (int, error) {
	// Allocate a port
	if len(state.FreePorts) == 0 {
		return 0, errors.New("No ports available")
	}
	port := state.FreePorts[0]
	state.FreePorts = state.FreePorts[1:]

	// Add an intercept entry
	intercept := &InterceptInfo{
		Name:        name,
		Patterns:    patterns,
		Port:        port,
		LastQueryAt: time.Now(),
	}
	dInfo := state.Deployments[deployment]
	if dInfo == nil {
		return 0, fmt.Errorf("Unknown deployment: %s", deployment)
	}
	dInfo.Intercepts = append(dInfo.Intercepts, intercept)

	// Post an event to update the deployment's pods
	err := state.publish(deployment)
	if err != nil {
		return 0, err
	}

	return port, nil
}

// Add an intercept to a deployment, return a port number
func (state *ProxyState) renewIntercept(deployment string, port int) error {
	dInfo := state.Deployments[deployment]
	if dInfo == nil {
		return fmt.Errorf("Unknown deployment: %s", deployment)
	}

	for _, intercept := range dInfo.Intercepts {
		if intercept.Port == port {
			intercept.LastQueryAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("unclaimed port: deployment=%s, port=%d", deployment, port)
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}

// Remove an intercept from a deployment by port number
func (state *ProxyState) stopIntercept(deployment string, port int) error {
	dInfo := state.Deployments[deployment]
	if dInfo == nil {
		return fmt.Errorf("Unknown deployment: %s", deployment)
	}

	// Filter out the intercept with the specified port
	newIntercepts := make([]*InterceptInfo, 0, max(0, len(dInfo.Intercepts)-1))
	for _, intercept := range dInfo.Intercepts {
		if intercept.Port != port {
			newIntercepts = append(newIntercepts, intercept)
		}
	}

	// Fail if the port was not found
	if len(dInfo.Intercepts) == len(newIntercepts) {
		return fmt.Errorf("Intercept not found for deployment %s port %d", deployment, port)
	}

	// Remove intercept and return port to the free pool
	dInfo.Intercepts = newIntercepts
	state.FreePorts = append(state.FreePorts, port)

	// Post an event to update the deployment's pods
	return state.publish(deployment)
}

// Handle list, create, and delete of an intercept for a deployment
func (state *ProxyState) handleIntercept(w http.ResponseWriter, r *http.Request) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	path := r.URL.Path
	comps := strings.Split(path, "/")
	if len(comps) != 3 {
		http.NotFound(w, r)
		return
	}
	deployment := comps[len(comps)-1]
	dInfo := state.Deployments[deployment]
	if dInfo == nil {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		result, err := json.Marshal(dInfo.Intercepts)
		if err != nil {
			panic(err)
		}
		w.Write([]byte(result))
	case http.MethodPost:
		d := json.NewDecoder(r.Body)
		type InInterceptInfo struct {
			Name     string
			Patterns []PatternInfo
			Port     int
		}
		var inIntercept InInterceptInfo
		err := d.Decode(&inIntercept)
		if err != nil {
			http.Error(w, "Unable to parse intercept info", 400)
			return
		}
		var port int
		if inIntercept.Port == 0 {
			port, err = state.startIntercept(deployment, inIntercept.Name, inIntercept.Patterns)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
		} else {
			err = state.renewIntercept(deployment, inIntercept.Port)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			port = inIntercept.Port
		}
		result, err := json.Marshal(port)
		if err != nil {
			panic(err)
		}
		w.Write([]byte(result))
	case http.MethodDelete:
		d := json.NewDecoder(r.Body)
		var port int
		err := d.Decode(&port)
		if err != nil {
			http.Error(w, "Unable to parse port number", 400)
			return
		}
		err = state.stopIntercept(deployment, port)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Write([]byte("success"))
	}
}

// cleanup expired proxy requests
func (state *ProxyState) cleanup() {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	log.Printf("cleanup: started")
	for deployment, dinfo := range state.Deployments {
		var remaining []*InterceptInfo
		var freePorts []int
		for _, intercept := range dinfo.Intercepts {
			// only keep intercepts older than 10 seconds
			if time.Since(intercept.LastQueryAt) < 10*time.Second {
				remaining = append(remaining, intercept)
				fmt.Printf("keeping intercept %s:%d", deployment, intercept.Port)
			} else {
				fmt.Printf("expiring intercept %s:%d", deployment, intercept.Port)
				freePorts = append(freePorts, intercept.Port)
			}
		}
		if len(freePorts) > 0 {
			dinfo.Intercepts = remaining
			state.FreePorts = append(state.FreePorts, freePorts...)
			// Post an event to update the deployment's pods
			err := state.publish(deployment)
			if err != nil {
				log.Printf("cleanup: %v", err)
			}
		}
	}
	log.Printf("cleanup: finished")
}

func main() {
	manager, err := golongpoll.StartLongpoll(golongpoll.Options{
		LoggingEnabled:                 true,
		MaxLongpollTimeoutSeconds:      120,
		MaxEventBufferSize:             1,
		EventTimeToLiveSeconds:         golongpoll.FOREVER,
		DeleteEventAfterFirstRetrieval: false,
	})
	if err != nil {
		log.Fatalf("Failed to create manager: %q", err)
	}
	state := newProxyState(manager)

	go func() {
		for {
			time.Sleep(1 * time.Second)
			state.cleanup()
		}
	}()

	http.HandleFunc("/state", state.handleState)
	http.HandleFunc("/routes", state.handleRoutes)
	http.HandleFunc("/intercept/", state.handleIntercept)

	fmt.Println("Starting server...")
	http.ListenAndServe("0.0.0.0:8081", nil)
}
