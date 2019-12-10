package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/datawire/ambassador/pkg/consulwatch"
)

var logger = logrus.New()

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
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)
}

func main() {
	argparser := &cobra.Command{
		Use:     os.Args[0],
		Version: Version,
		Run:     Main,
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

	stdLogger := log.New(logger.WriterLevel(logrus.DebugLevel), "", 0)

	logger.WithFields(logrus.Fields{
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

	// TODO: this can probably be removed in the future or modified somehow
	agent := NewAgent(os.Getenv(EnvAmbassadorID), os.Getenv(EnvSecretNamespace), os.Getenv(EnvSecretName), consul)

	caRootWatcher, err := consulwatch.NewConnectCARootsWatcher(consul, stdLogger)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Watching CA leaf for %s\n", agent.ConsulServiceName)
	leafWatcher, err := consulwatch.NewConnectLeafWatcher(consul, stdLogger, agent.ConsulServiceName)

	caRootChanged := make(chan *consulwatch.CARoots)
	leafChanged := make(chan *consulwatch.Certificate)

	caRootWatcher.Watch(func(roots *consulwatch.CARoots, e error) {
		if e != nil {
			logger.Errorf("Error watching root CA: %v\n", err)
		}

		caRootChanged <- roots
	})
	leafWatcher.Watch(func(certificate *consulwatch.Certificate, e error) {
		if e != nil {
			logger.Errorf("Error watching certificates: %v\n", err)
		}

		leafChanged <- certificate
	})

	// TODO: this is probably wrong, but whatever
	go func() {
		if err := caRootWatcher.Start(); err != nil {
			logger.Fatalln(err)
		}
	}()

	go func() {
		if err := leafWatcher.Start(); err != nil {
			logger.Fatalln(err)
		}
	}()

	var caRoot *consulwatch.CARoot
	var leafCert *consulwatch.Certificate

	for {
		select {
		case cert := <-caRootChanged:
			temp := cert.Roots[cert.ActiveRootID]
			caRoot = &temp
		case cert := <-leafChanged:
			leafCert = cert
		}

		if caRoot != nil && leafCert != nil {
			chain := createCertificateChain(caRoot.PEM, leafCert.PEM)
			secret := formatKubernetesSecretYAML(agent.SecretName, chain, leafCert.PrivateKeyPEM)

			err := applySecret(agent.SecretNamespace, secret)
			if err != nil {
				logger.Error(err)
				continue
			}

			logger.WithFields(logrus.Fields{"namespace": agent.SecretNamespace, "secret": agent.SecretName}).
				Info("Updating TLS certificate secret")
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

func createCertificateChain(rootPEM string, leafPEM string) string {
	return leafPEM + rootPEM
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
	logger.WithFields(logrus.Fields{"args": cmd.Args}).Debug("Computed kubectl command and arguments")

	var errBuffer bytes.Buffer
	cmd.Stderr = &errBuffer

	cmd.Stdin = bytes.NewBuffer([]byte(yaml))

	_, err := cmd.Output()
	fmt.Println(errBuffer.String())

	return err
}
