package consulwatch

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"

	"github.com/datawire/ambassador/pkg/supervisor"
)

const (
	// defConsulServiceName is the default Consul service name
	defConsulServiceName = "ambassador"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}


type ConsulWatchSpec struct {
	Id            string `json:"id"`
	ConsulAddress string `json:"consul-address"`
	Datacenter    string `json:"datacenter"`
	ServiceName   string `json:"service-name"`
	Secret        string `json:"secret"`
}

func (c ConsulWatchSpec) WatchId() string {
	return fmt.Sprintf("%s|%s|%s", c.ConsulAddress, c.Datacenter, c.ServiceName)
}

type connectLeafWatcher struct {
	consul *api.Client
	plan   *watch.Plan
	logger *log.Logger
}

func newConnectLeafWatcher(consul *api.Client, logger *log.Logger, service string) (*connectLeafWatcher, error) {
	if service == "" {
		err := errors.New("service name is empty")
		return nil, err
	}

	watcher := &connectLeafWatcher{consul: consul}

	plan, err := watch.Parse(map[string]interface{}{"type": "connect_leaf", "service": service})
	if err != nil {
		return nil, err
	}

	if logger != nil {
		watcher.logger = logger
	} else {
		watcher.logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	watcher.plan = plan

	return watcher, nil
}

func (w *connectLeafWatcher) Watch(handler func(*Certificate, error)) {
	w.plan.HybridHandler = func(val watch.BlockingParamVal, raw interface{}) {
		if raw == nil {
			handler(nil, fmt.Errorf("unexpected empty/nil response from consul"))
			return
		}

		v, ok := raw.(*api.LeafCert)
		if !ok {
			handler(nil, fmt.Errorf("unexpected raw type. expected: %T, was: %T", &api.LeafCert{}, raw))
			return
		}

		certificate := &Certificate{
			PEM:           v.CertPEM,
			PrivateKeyPEM: v.PrivateKeyPEM,
			ValidBefore:   v.ValidBefore,
			ValidAfter:    v.ValidAfter,
			SerialNumber:  v.SerialNumber,
			Service:       v.Service,
			ServiceURI:    v.ServiceURI,
		}

		handler(certificate, nil)
	}
}

func (w *connectLeafWatcher) Start() error {
	return w.plan.RunWithClientAndLogger(w.consul, w.logger)
}

func (w *connectLeafWatcher) Stop() {
	w.plan.Stop()
}

// connectCARootsWatcher watches the Consul Connect CA roots endpoint for changes and invokes a a handler function
// whenever it changes.
type connectCARootsWatcher struct {
	consul *api.Client
	plan   *watch.Plan
	logger *log.Logger
}

func newConnectCARootsWatcher(consul *api.Client, logger *log.Logger) (*connectCARootsWatcher, error) {
	watcher := &connectCARootsWatcher{consul: consul}

	plan, err := watch.Parse(map[string]interface{}{"type": "connect_roots"})
	if err != nil {
		return nil, err
	}

	if logger != nil {
		watcher.logger = logger
	} else {
		watcher.logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	watcher.plan = plan

	return watcher, nil
}

func (w *connectCARootsWatcher) Watch(handler func(*CARoots, error)) {
	w.plan.HybridHandler = func(val watch.BlockingParamVal, raw interface{}) {
		if raw == nil {
			handler(nil, fmt.Errorf("unexpected empty/nil response from consul"))
			return
		}

		v, ok := raw.(*api.CARootList)
		if !ok {
			handler(nil, fmt.Errorf("unexpected raw type. expected: %T, was: %T", &api.CARootList{}, raw))
			return
		}

		rootsMap := make(map[string]CARoot)
		for _, root := range v.Roots {
			rootsMap[root.ID] = CARoot{
				ID:     root.ID,
				Name:   root.Name,
				PEM:    root.RootCertPEM,
				Active: root.Active,
			}
		}

		roots := &CARoots{
			ActiveRootID: v.ActiveRootID,
			TrustDomain:  v.TrustDomain,
			Roots:        rootsMap,
		}

		handler(roots, nil)
	}
}

func (w *connectCARootsWatcher) Start() error {
	return w.plan.RunWithClientAndLogger(w.consul, w.logger)
}

func (w *connectCARootsWatcher) Stop() {
	w.plan.Stop()
}

// ConnectWatcher is a watcher for Consul certificates
type ConnectWatcher struct {
	process *supervisor.Process
	agent   *Agent
	consul  *api.Client

	mainWorker *supervisor.Worker

	caRootWorker       *supervisor.Worker
	caRootCertificates chan *CARoots
	caRootWatcher      *connectCARootsWatcher

	leafCertificates chan *Certificate
	leafWatcher      *connectLeafWatcher
	leafWorker       *supervisor.Worker
}

// NewConnectWatcher creates a new watcher for Consul Connect certificates
func NewConnectWatcher(p *supervisor.Process, consul *api.Client, agent *Agent) *ConnectWatcher {
	return &ConnectWatcher{
		process:            p,
		consul:             consul,
		agent:              agent,
		caRootCertificates: make(chan *CARoots),
		leafCertificates:   make(chan *Certificate),
	}
}

// Watch watches the TLS certificate issued by the Consul CA and stores it as a Kubernetes
// secret that Ambassador will use to authenticate with upstream services.
// This methods does not block while waiting for updates from Consul.
func (w *ConnectWatcher) Watch() error {
	var err error

	log.Printf("Watching Root CA for %s\n", w.agent.ConsulServiceName)
	w.caRootWatcher, err = newConnectCARootsWatcher(w.consul, logger)
	if err != nil {
		return err
	}
	w.caRootWatcher.Watch(func(roots *CARoots, e error) {
		if e != nil {
			w.process.Logf("Error watching root CA: %v\n", err)
		}

		w.caRootCertificates <- roots
	})

	log.Printf("Watching CA leaf for %s\n", w.agent.ConsulServiceName)
	w.leafWatcher, err = newConnectLeafWatcher(w.consul, logger, w.agent.ConsulServiceName)
	if err != nil {
		return err
	}
	w.leafWatcher.Watch(func(certificate *Certificate, e error) {
		if e != nil {
			w.process.Logf("Error watching certificates: %v\n", err)
		}
		w.leafCertificates <- certificate
	})

	w.mainWorker = w.process.Go(func(p *supervisor.Process) error {
		p.Log("Starting Consul certificates watcher...")
		chain := newCertChain(p)

		// wait for root CA and certificates, and update the
		// copy in Kubernetes when we get a new version
	loop:
		for {
			select {
			case cert, ok := <-w.caRootCertificates:
				if !ok {
					break loop // return when one of the input channels is closed
				}
				chain.CA = cert
			case cert, ok := <-w.leafCertificates:
				if !ok {
					break loop
				}
				chain.Leaf = cert
			case <-p.Shutdown():
				break loop
			}

			if err := chain.WriteTo(w.agent.SecretNamespace, w.agent.SecretName); err != nil {
				p.Log(err)
				continue
			}
		}

		p.Log("Quitting Consul certificates watcher")
		return nil
	})
	w.leafWorker = w.process.Go(func(p *supervisor.Process) error {
		p.Log("Starting Consul leaf certificates watcher...")
		if err := w.leafWatcher.Start(); err != nil {
			p.Logf("failed to start Consul leaf watcher %v", err)
			return err
		}
		return nil
	})
	w.caRootWorker = w.process.Go(func(p *supervisor.Process) error {
		p.Log("Starting Consul CA certificate watcher...")
		if err := w.caRootWatcher.Start(); err != nil {
			p.Logf("failed to start Consul CA certificate watcher %v", err)
			return err
		}
		return nil
	})

	return nil
}

// Close stops watching Consul Connect
func (w *ConnectWatcher) Close() {
	w.process.Logf("Stopping Consul Connect watchers...")
	w.caRootWatcher.Stop()
	w.caRootWorker.Wait()

	w.leafWatcher.Stop()
	w.leafWorker.Wait()

	w.caRootWatcher.Stop()
	close(w.caRootCertificates)

	w.leafWatcher.Stop()
	close(w.leafCertificates)

	w.mainWorker.Wait()
}

// getNamespaceAndName gets the namespace and the name of a resource
// For example, "default/my-secret" -> ("default", "my-secret")
//              "my-secret"         -> ("", "my-secret")
func getNamespaceAndName(s string) (string, string) {
	r := strings.SplitN(s, "/", 2)
	switch len(r) {
	case 0:
		return "", ""
	case 1:
		return "", r[0]
	default:
		return r[0], r[1]
	}
}

