package consulwatch

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"

	"github.com/datawire/ambassador/pkg/k8s"
	"github.com/datawire/ambassador/pkg/supervisor"
)

const (
	// envAmbassadorID creates a secret for a specific instance of an Ambassador API Gateway. The TLS secret name will
	// be formatted as "$AMBASSADOR_ID-consul-connect."
	envAmbassadorID = "_AMBASSADOR_ID"

	// envSecretName is the full name of the Kubernetes Secret that contains the TLS certificate provided
	// by Consul. If this value is set then the value of AMBASSADOR_ID is ignored when the name of the TLS secret is
	// computed.
	envSecretName = "_AMBASSADOR_TLS_SECRET_NAME"

	// envSecretNamespace sets the namespace where the TLS secret is created.
	envSecretNamespace = "_AMBASSADOR_TLS_SECRET_NAMESPACE"
)

const (
	secretTemplate = `---
kind: Secret
apiVersion: v1
metadata:
    name: "%s"
type: "kubernetes.io/tls"
data:
    tls.crt: "%s"
    tls.key: "%s"
`
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags)
}

type ConnectLeafWatcher struct {
	consul *api.Client
	plan   *watch.Plan
	logger *log.Logger
}

func NewConnectLeafWatcher(consul *api.Client, logger *log.Logger, service string) (*ConnectLeafWatcher, error) {
	if service == "" {
		err := errors.New("service name is empty")
		return nil, err
	}

	watcher := &ConnectLeafWatcher{consul: consul}

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

func (w *ConnectLeafWatcher) Watch(handler func(*Certificate, error)) {
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

func (w *ConnectLeafWatcher) Start() error {
	return w.plan.RunWithClientAndLogger(w.consul, w.logger)
}

func (w *ConnectLeafWatcher) Stop() {
	w.plan.Stop()
}

// ConnectCARootsWatcher watches the Consul Connect CA roots endpoint for changes and invokes a a handler function
// whenever it changes.
type ConnectCARootsWatcher struct {
	consul *api.Client
	plan   *watch.Plan
	logger *log.Logger
}

func NewConnectCARootsWatcher(consul *api.Client, logger *log.Logger) (*ConnectCARootsWatcher, error) {
	watcher := &ConnectCARootsWatcher{consul: consul}

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

func (w *ConnectCARootsWatcher) Watch(handler func(*CARoots, error)) {
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

func (w *ConnectCARootsWatcher) Start() error {
	return w.plan.RunWithClientAndLogger(w.consul, w.logger)
}

func (w *ConnectCARootsWatcher) Stop() {
	w.plan.Stop()
}

type agent struct {
	// AmbassadorID is the ID of the Ambassador instance.
	AmbassadorID string

	// The Agent registers a Consul Service when it starts and then fetches the leaf TLS certificate from the Consul
	// HTTP API with this name.
	ConsulServiceName string

	// SecretNamespace is the Namespace where the TLS secret is managed.
	SecretNamespace string

	// SecretName is the Name of the TLS secret managed by this agent.
	SecretName string

	// consulAPI is the client used to communicate with the Consul HTTP API server.
	consul *api.Client
}

func newAgent(ambassadorID string, secretNamespace string, secretName string, consul *api.Client) *agent {
	consulServiceName := "ambassador"
	if ambassadorID != "" {
		consulServiceName += "-" + ambassadorID
	}

	if secretName == "" {
		secretName = consulServiceName + "-consul-connect"
	}

	return &agent{
		AmbassadorID:      consulServiceName,
		SecretNamespace:   secretNamespace,
		SecretName:        secretName,
		ConsulServiceName: consulServiceName,
		consul:            consul,
	}
}

// certChain is a certificates chain that will be translated to a k8s certificate
type certChain struct {
	process *supervisor.Process
	CA      *CARoots
	Leaf    *Certificate
}

func newCertChain(process *supervisor.Process) certChain {
	return certChain{
		process: process,
	}
}

func (c *certChain) WriteTo(secretNamespace, secretName string) error {
	if c.CA == nil || c.Leaf == nil {
		return nil // we need both CA & Leaf for creating/updating the secret
	}

	temp := c.CA.Roots[c.CA.ActiveRootID]
	caRoot := &temp

	// create the certificate chain
	chain := c.Leaf.PEM + caRoot.PEM

	// format the kubernetes secret
	chain64 := base64.StdEncoding.EncodeToString([]byte(chain))
	key64 := base64.StdEncoding.EncodeToString([]byte(c.Leaf.PrivateKeyPEM))
	secret := fmt.Sprintf(secretTemplate, secretName, chain64, key64)

	logger.Printf("Creating/updating TLS certificate secret: namespace=%s, secret=%s", secretNamespace, secretName)
	kubeinfo := k8s.NewKubeInfo("", "", secretNamespace)
	args, err := kubeinfo.GetKubectlArray("apply", "-f", "-")
	if err != nil {
		return err
	}
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return err
	}
	apply := c.process.Command(kubectl, args...)
	apply.Stdin = strings.NewReader(secret)
	err = apply.Start()
	if err != nil {
		return err
	}
	err = c.process.DoClean(apply.Wait, apply.Process.Kill)
	if err != nil {
		return err
	}
	return nil
}

// ConnectWatcher is a watcher for Consul Connect certificates
type ConnectWatcher struct {
	process *supervisor.Process
	agent   *agent
	consul  *api.Client

	consulWorker *supervisor.Worker

	caRootWorker       *supervisor.Worker
	caRootCertificates chan *CARoots
	caRootWatcher      *ConnectCARootsWatcher

	leafCertificates chan *Certificate
	leafWatcher      *ConnectLeafWatcher
	leafWorker       *supervisor.Worker
}

// NewConnectWatcher creates a new watcher for Consul Connect certificates
func NewConnectWatcher(p *supervisor.Process, consul *api.Client) *ConnectWatcher {
	// TODO(alvaro): this shold be obtained from some custom resource
	agent := newAgent(os.Getenv(envAmbassadorID), os.Getenv(envSecretNamespace), os.Getenv(envSecretName), consul)

	return &ConnectWatcher{
		process:            p,
		consul:             consul,
		agent:              agent,
		caRootCertificates: make(chan *CARoots),
		leafCertificates:   make(chan *Certificate),
	}
}

// Watch retrieves the TLS certificate issued by the Consul CA and stores it as a Kubernetes
// secret that Ambassador will use to authenticate with upstream services.
func (w *ConnectWatcher) Watch() error {
	var err error

	log.Printf("Watching Root CA for %s\n", w.agent.ConsulServiceName)
	w.caRootWatcher, err = NewConnectCARootsWatcher(w.consul, logger)
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
	w.leafWatcher, err = NewConnectLeafWatcher(w.consul, logger, w.agent.ConsulServiceName)
	if err != nil {
		return err
	}
	w.leafWatcher.Watch(func(certificate *Certificate, e error) {
		if e != nil {
			w.process.Logf("Error watching certificates: %v\n", err)
		}
		w.leafCertificates <- certificate
	})

	w.consulWorker = w.process.Go(func(p *supervisor.Process) error {
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

	w.consulWorker.Wait()
}
