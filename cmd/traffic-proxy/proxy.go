package traffic_manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/jcuga/golongpoll"
	// golongpoll doesn't have a go.mod, and something about that
	// confuses Go about whether it needs golongpoll's gouuid
	// dependency or not.  So, tell Go explicitly: We need gouuid.
	_ "github.com/nu7hatch/gouuid"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/supervisor"
	"github.com/datawire/apro/lib/licensekeys"
	"github.com/datawire/apro/lib/metriton"
)

// Helper types for the Watcher

type svcResource struct {
	Spec svcSpec
}

type svcSpec struct {
	ClusterIP string
	Ports     []svcPort
}

type svcPort struct {
	Name     string
	Port     int
	Protocol string
}

type Table struct {
	Name   string  `json:"name"`
	Routes []Route `json:"routes"`
}

func (t *Table) Add(route Route) {
	t.Routes = append(t.Routes, route)
}

type Route struct {
	Name   string `json:"name,omitempty"`
	Ip     string `json:"ip"`
	Proto  string `json:"proto"`
	Port   string `json:"port,omitempty"`
	Target string `json:"target"`
	Action string `json:"action,omitempty"`
}

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
	snapshot    *Table
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
		snapshot:    nil, // 503 until we have a snapshot
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
	w.Write(result)
}

func (state *ProxyState) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if state.snapshot == nil {
		http.Error(w, "snapshot unavailable", http.StatusServiceUnavailable)
		return
	}

	result, err := json.Marshal([]Table{*state.snapshot})
	if err != nil {
		panic(err)
	}
	w.Write(result)
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
		state.Deployments[deployment] = dInfo
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

	// deployment := strings.TrimRight(r.URL.Path, "/")
	deployment := r.URL.Path

	log.Printf("handleIntercept: deployment is %v", deployment)

	if deployment == "" {
		deployments := make([]string, len(state.Deployments))
		i := 0
		for deployment := range state.Deployments {
			deployments[i] = deployment
			i++
		}
		sort.Strings(deployments)
		result, err := json.Marshal(map[string]interface{}{"paths": deployments})
		if err != nil {
			panic(err)
		}
		w.Write(result)
		return
	}

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
		w.Write(result)
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
		w.Write(result)
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
func (state *ProxyState) cleanup(p *supervisor.Process) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	keepMsg := []string{}
	expireMsg := []string{}
	expireDeploys := []string{}
	for deployment, dinfo := range state.Deployments {
		// Expire deployments that haven't updated within the last 2 minutes
		msg := fmt.Sprintf("deploy/%s", deployment)
		if time.Since(dinfo.LastQueryAt) > 2*time.Minute {
			expireMsg = append(expireMsg, msg)
			expireDeploys = append(expireDeploys, deployment)
			continue
		}
		keepMsg = append(keepMsg, msg)

		var remaining []*InterceptInfo
		var freedPorts []int
		for _, intercept := range dinfo.Intercepts {
			msg := fmt.Sprintf("cept/%s:%d", deployment, intercept.Port)
			// only keep intercepts that updated within the last 10 seconds
			if time.Since(intercept.LastQueryAt) < 10*time.Second {
				remaining = append(remaining, intercept)
				keepMsg = append(keepMsg, msg)
			} else {
				freedPorts = append(freedPorts, intercept.Port)
				expireMsg = append(expireMsg, msg)
			}
		}
		if len(freedPorts) > 0 {
			dinfo.Intercepts = remaining
			state.FreePorts = append(state.FreePorts, freedPorts...)
			// Post an event to update the deployment's pods
			err := state.publish(deployment)
			if err != nil {
				p.Logf("error posting to %s: %v", deployment, err)
			}
		}
	}

	for _, deployment := range expireDeploys {
		delete(state.Deployments, deployment)
	}

	if len(expireMsg) > 0 {
		p.Logf("Keeping %q; Expiring %q", keepMsg, expireMsg)
	}
}

func updateTable(p *supervisor.Process, w *k8s.Watcher, state *ProxyState) {
	const ProxyRedirPort = "1234" // Client can always override this
	table := Table{Name: "kubernetes"}

	for _, svc := range w.List("services") {
		decoded := svcResource{}
		err := svc.Decode(&decoded)
		if err != nil {
			p.Logf("error decoding service: %v", err)
			continue
		}

		spec := decoded.Spec

		ports := ""
		for _, port := range spec.Ports {
			if ports == "" {
				ports = fmt.Sprintf("%d", port.Port)
			} else {
				ports = fmt.Sprintf("%s,%d", ports, port.Port)
			}
		}

		ip := spec.ClusterIP
		// for headless services the IP is None, we should properly handle
		// these by listening for endpoints and returning multiple A records
		if ip != "" && ip != "None" {
			qualName := svc.Name() + "." + svc.Namespace() + ".svc.cluster.local"
			table.Add(Route{
				Name:   qualName,
				Ip:     ip,
				Port:   ports,
				Proto:  "tcp",
				Target: ProxyRedirPort,
			})
		}
	}

	for _, pod := range w.List("pods") {
		qname := ""

		hostname, ok := pod.Spec()["hostname"]
		if ok && hostname != "" {
			qname += hostname.(string)
		}

		subdomain, ok := pod.Spec()["subdomain"]
		if ok && subdomain != "" {
			qname += "." + subdomain.(string)
		}

		if qname == "" {
			// Note: this is a departure from kubernetes, kubernetes will
			// simply not publish a dns name in this case.
			qname = pod.Name() + "." + pod.Namespace() + ".pod.cluster.local"
		} else {
			qname += ".svc.cluster.local"
		}

		ip, ok := pod.Status()["podIP"]
		if ok && ip != "" {
			table.Add(Route{
				Name:   qname,
				Ip:     ip.(string),
				Proto:  "tcp",
				Target: ProxyRedirPort,
			})
		}
	}
	state.mutex.Lock()
	state.snapshot = &table
	state.mutex.Unlock()
}

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func Main() {
	log.Printf("Traffic Manager version %s", Version)

	argparser := &cobra.Command{
		Use:          os.Args[0],
		Version:      Version,
		RunE:         Run,
		SilenceUsage: true,
	}

	cmdContext := &licensekeys.LicenseContext{}
	if err := cmdContext.AddFlagsTo(argparser); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	argparser.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		licenseClaims, err := cmdContext.GetClaims()

		if err == nil {
			err = licenseClaims.RequireFeature(licensekeys.FeatureTraffic)
		}
		if err == nil {
			go metriton.PhoneHome(licenseClaims, nil, "traffic-proxy", Version)
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err := argparser.Execute()
	if err != nil {
		os.Exit(2)
	}
}

func Run(flags *cobra.Command, args []string) error {
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

	http.HandleFunc("/state", state.handleState)
	http.HandleFunc("/routes", state.handleRoutes)
	http.HandleFunc("/snapshot", state.handleSnapshot)
	http.Handle("/intercept/", http.StripPrefix("/intercept/", http.HandlerFunc(state.handleIntercept)))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		result, err := json.Marshal(map[string]interface{}{"paths": []string{
			"/state",
			"/routes",
			"/snapshot",
			"/intercept/",
		}})
		if err != nil {
			panic(err)
		}
		w.Write(result)
	})

	sup := supervisor.WithContext(context.Background())
	sup.Supervise(&supervisor.Worker{
		Name: "longpoll",
		Work: func(p *supervisor.Process) error {
			p.Ready()
			<-p.Shutdown()
			manager.Shutdown()
			return nil
		},
	})
	sup.Supervise(&supervisor.Worker{
		Name: "cleanup",
		Work: func(p *supervisor.Process) error {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			p.Ready()
			for {
				select {
				case <-ticker.C:
					state.cleanup(p)
				case <-p.Shutdown():
					return nil
				}
			}
		},
	})
	sup.Supervise(&supervisor.Worker{
		Name: "watcher",
		Work: func(p *supervisor.Process) error {
			client, err := k8s.NewClient(nil)
			if err != nil {
				return err
			}
			watcher := client.Watcher()
			callback := func(w *k8s.Watcher) { updateTable(p, w, state) }
			if err := watcher.Watch("services", callback); err != nil {
				return err
			}
			if err := watcher.Watch("pods", callback); err != nil {
				return err
			}

			// The watcher panics on error, so...
			defer func() {
				if r := recover(); r != nil {
					p.Logf("Failed: %v", r)
				}
			}()
			watcher.Start()

			p.Ready()
			<-p.Shutdown()
			watcher.Stop()

			return nil
		},
	})
	sup.Supervise(&supervisor.Worker{
		Name: "server",
		Work: func(p *supervisor.Process) error {
			httpPort := os.Getenv("APRO_HTTP_PORT")
			if httpPort == "" {
				httpPort = "8081"
			}
			server := &http.Server{Addr: net.JoinHostPort("0.0.0.0", httpPort)}
			p.Ready()
			return p.DoClean(
				func() error {
					err := server.ListenAndServe()
					if err == http.ErrServerClosed {
						return nil
					}
					return err
				},
				func() error {
					return server.Shutdown(context.Background())
				})
		},
	})
	sup.Supervise(&supervisor.Worker{
		Name: "signal",
		Work: func(p *supervisor.Process) error {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			p.Ready()
			select {
			case sig := <-sigs:
				return errors.Errorf("Received signal %v", sig)
			case <-p.Shutdown():
				return nil
			}
		},
	})
	sup.Supervise(&supervisor.Worker{
		Name: "sshd",
		Work: func(p *supervisor.Process) error {
			cmd := p.Command("/usr/sbin/sshd", "-De")
			if err := cmd.Start(); err != nil {
				return err
			}
			p.Ready()
			return p.DoClean(cmd.Wait, cmd.Process.Kill)
		},
	})

	sup.Logger.Printf("Starting server...")
	runErrors := sup.Run()
	sup.Logger.Printf("")

	if len(runErrors) > 0 {
		sup.Logger.Printf("Traffic Manager has exited with %d error(s):", len(runErrors))
		for _, err := range runErrors {
			sup.Logger.Printf("- %v", err)
		}
	}

	return errors.New("Traffic Manager has exited")
}
