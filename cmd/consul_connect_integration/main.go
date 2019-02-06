package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/apro/lib/licensekeys"
)

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

type ConsulRootCert struct {
	Certificate              string
	IntermediateCertificates []string
}

type ConsulLeafCert struct {
	Certificate string
	PrivateKey  string
}

type Agent struct {
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
	consul *consulapi.Client

	ConsulRootCert *ConsulRootCert
	ConsulLeafCert *ConsulLeafCert

	RootCertChange chan ConsulRootCert
	LeafCertChange chan ConsulLeafCert
}

func NewAgent(ambassadorID string, secretNamespace string, secretName string, consul *consulapi.Client) *Agent {
	consulServiceName := "ambassador"
	if ambassadorID != "" {
		consulServiceName += "-" + ambassadorID
	}

	if secretName == "" {
		secretName = consulServiceName + "-consul-connect"
	}

	return &Agent{
		AmbassadorID:      consulServiceName,
		SecretNamespace:   secretNamespace,
		SecretName:        secretName,
		ConsulServiceName: consulServiceName,
		consul:            consul,
	}
}

const (
	// EnvAmbassadorID creates a secret for a specific instance of an Ambassador API Gateway. The TLS secret name will
	// be formatted as "$AMBASSADOR_ID-consul-connect."
	EnvAmbassadorID = "_AMBASSADOR_ID"

	// EnvConsulAPIHost is the IP address or DNS name of the Consul Agent's HTTP API server.
	EnvConsulAPIHost = "_CONSUL_HOST"

	// EnvConsulAPIPort is the Port number of the Consul Agent's HTTP API server.
	EnvConsulAPIPort = "_CONSUL_PORT"

	// EnvAmbassadorTLSSecretName is the full name of the Kubernetes Secret that contains the TLS certificate provided
	// by Consul. If this value is set then the value of AMBASSADOR_ID is ignored when the name of the TLS secret is
	// computed.
	EnvSecretName = "_AMBASSADOR_TLS_SECRET_NAME"

	// EnvSecretNamespace sets the namespace where the TLS secret is created.
	EnvSecretNamespace = "_AMBASSADOR_TLS_SECRET_NAMESPACE"
)

var secretTemplate = `---
kind: Secret
apiVersion: v1
metadata:
    name: "%s"
type: "kubernetes.io/tls"
data:
    tls.crt: "%s"
    tls.key: "%s"
`

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func main() {
	argparser := &cobra.Command{
		Use:     os.Args[0],
		Version: Version,
		Run:     Main,
	}
	keycheck := licensekeys.InitializeCommandFlags(argparser.PersistentFlags(), "consul-integration", Version)
	argparser.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		err := keycheck(cmd.PersistentFlags())
		if err == nil {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		time.Sleep(5 * 60 * time.Second)
		os.Exit(1)
	}
	err := argparser.Execute()
	if err != nil {
		os.Exit(2)
	}
}

func Main(flags *cobra.Command, args []string) {
	consulAPIHost := getEnvOrFallback(EnvConsulAPIHost, "127.0.0.1")
	consulAPIPort := getEnvOrFallback(EnvConsulAPIPort, "8500")
	consulAddress := fmt.Sprintf("%s:%s", consulAPIHost, consulAPIPort)

	// TODO: This really should log the integration version as well. But how?
	log.WithFields(log.Fields{
		"consul_host": consulAPIHost,
		"consul_port": consulAPIPort,
		"version":     Version,
	}).Info("Starting Consul Connect Integration")

	config := consulapi.DefaultConfig()
	config.Address = consulAddress

	consul, err := consulapi.NewClient(config)
	if err != nil {
		log.Fatalln(err)
	}

	agent := NewAgent(os.Getenv(EnvAmbassadorID), os.Getenv(EnvSecretNamespace), os.Getenv(EnvSecretName), consul)
	agent.RootCertChange = make(chan ConsulRootCert)
	agent.LeafCertChange = make(chan ConsulLeafCert)

	go agent.WatchConsulRootCertificateChanges()
	go agent.WatchConsulLeafCertificateChanges()

	agent.Run()
}

func (a *Agent) Run() {
	for {
		select {
		case cert := <-a.RootCertChange:
			a.ConsulRootCert = &cert
		case cert := <-a.LeafCertChange:
			a.ConsulLeafCert = &cert
		}

		if a.ConsulRootCert != nil && a.ConsulLeafCert != nil {
			log.WithFields(log.Fields{
				"namespace": a.SecretNamespace,
				"secret":    a.SecretName,
			}).Info("Updating TLS certificate secret")

			chain := createCertificateChain(
				a.ConsulRootCert.Certificate,
				a.ConsulLeafCert.Certificate,
				a.ConsulRootCert.IntermediateCertificates)

			secret := formatKubernetesSecretYAML(a.SecretName, chain, a.ConsulLeafCert.PrivateKey)
			err := applySecret(a.SecretNamespace, secret)
			if err != nil {
				log.Error(err)
			} else {
				log.WithFields(log.Fields{
					"namespace": a.SecretNamespace,
					"secret":    a.SecretName,
				}).Info("Updating TLS certificate secret")
			}
		}
	}
}

func (a *Agent) WatchConsulRootCertificateChanges() {
	currentIndex := uint64(0)

	for {
		log.WithFields(log.Fields{"current-index": currentIndex}).Info("Waiting for Root CA certificate to change")
		res, meta, err := a.consul.Agent().ConnectCARoots(&consulapi.QueryOptions{
			WaitIndex: currentIndex,
		})

		if err != nil {
			log.Fatalln(err)
		}

		if res == nil || meta == nil {
			time.Sleep(1 * time.Second)
		} else {
			for _, root := range res.Roots {

				// NOTE: Philip Lombardi - 2019-01
				// ===============================
				//
				// The Consul CA HTTP API docs say there should be intermediate certificates. The Go API does not seem
				// to expose the intermediate certificates at all however.
				//
				// API Docs: https://www.consul.io/docs/connect/ca.html
				//
				if root.Active {
					a.RootCertChange <- ConsulRootCert{
						Certificate:              root.RootCertPEM,
						IntermediateCertificates: []string{},
					}

					break
				}
			}

			currentIndex = meta.LastIndex
		}
	}
}

func (a *Agent) WatchConsulLeafCertificateChanges() {
	currentIndex := uint64(0)

	for {
		log.WithFields(log.Fields{
			"service":       a.ConsulServiceName,
			"current-index": currentIndex,
		}).Info("Fetching Leaf ")

		res, meta, err := a.consul.Agent().ConnectCALeaf(a.ConsulServiceName, &consulapi.QueryOptions{
			WaitIndex: currentIndex,
		})

		if err != nil {
			log.Fatalln(err)
		}

		if res == nil || meta == nil {
			time.Sleep(1 * time.Second)
		} else {
			a.LeafCertChange <- ConsulLeafCert{
				Certificate: res.CertPEM,
				PrivateKey:  res.PrivateKeyPEM,
			}

			currentIndex = meta.LastIndex
		}
	}
}

func getEnvOrFallback(name string, fallback string) string {
	if result := os.Getenv(name); result != "" {
		return result
	} else {
		return fallback
	}
}

func createCertificateChain(root string, leaf string, intermediaries []string) string {
	result := intermediaries
	result = append(result, root)
	result = append([]string{leaf}, result...)
	return strings.Join(result, "")
}

func formatKubernetesSecretYAML(name string, chain string, key string) string {
	chain64 := base64.StdEncoding.EncodeToString([]byte(chain))
	key64 := base64.StdEncoding.EncodeToString([]byte(key))

	return fmt.Sprintf(secretTemplate, name, chain64, key64)
}

func applySecret(namespace string, yaml string) error {
	args := []string{"apply", "-f", "-"}

	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}

	cmd := exec.Command("kubectl", args...)
	log.WithFields(log.Fields{"args": cmd.Args}).Debug("Computed kubectl command and arguments")

	var errBuffer bytes.Buffer
	cmd.Stderr = &errBuffer

	cmd.Stdin = bytes.NewBuffer([]byte(yaml))

	_, err := cmd.Output()
	fmt.Println(errBuffer.String())

	return err
}
