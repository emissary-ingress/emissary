package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

var err error
var consul *consulapi.Client

type rootCert struct {
	CertPEM              string
	IntermediateCertsPEM []string
}

type leafCert struct {
	CertPEM       string
	PrivateKeyPEM string
}

const (
	// EnvAmbassadorID creates a secret for a specific instance of an Ambassador API Gateway. The TLS secret name will
	// be formatted as "$AMBASSADOR_ID-consul-connect."
	EnvAmbassadorID = "AMBASSADOR_ID"

	// EnvConsulAPIHost is the IP address or DNS name of the Consul Agent's HTTP API server.
	EnvConsulAPIHost = "CONSUL_HOST"

	// EnvConsulAPIPort is the Port number of the Consul Agent's HTTP API server.
	EnvConsulAPIPort = "CONSUL_PORT"

	// EnvAmbassadorTLSSecretName is the full name of the Kubernetes Secret that contains the TLS certificate provided
	// by Consul. If this value is set then the value of AMBASSADOR_ID is ignored when the name of the TLS secret is
	// computed.
	EnvSecretName = "AMBASSADOR_TLS_SECRET_NAME"

	// EnvSecretNamespace sets the namespace where the TLS secret is created.
	EnvSecretNamespace = "AMBASSADOR_TLS_SECRET_NAMESPACE"
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
	log.Info("Ambassador Consul Connect integration is starting...")

	consulAPIHost := getEnvOrFallback(EnvConsulAPIHost, "127.0.0.1")
	consulAPIPort := getEnvOrFallback(EnvConsulAPIPort, "8500")

	consulAddress := fmt.Sprintf("%s:%s", consulAPIHost, consulAPIPort)

	config := consulapi.DefaultConfig()
	config.Address = consulAddress
	log.WithFields(log.Fields{"address": consulAddress}).Info("Set Consul HTTP API Address")

	consul, err = consulapi.NewClient(config)
	if err != nil {
		log.Fatalln(err)
	}

	rootCertChannel := make(chan rootCert)
	leafCertChannel := make(chan leafCert)

	ambassadorID := os.Getenv(EnvAmbassadorID)
	if ambassadorID != "" {
		log.WithFields(log.Fields{EnvAmbassadorID: ambassadorID}).Info("Set Ambassador ID")
	}

	consulServiceName := createAmbassadorConsulServiceName(ambassadorID)
	if err := registerAmbassadorAsConsulService(consulServiceName, consul.Agent()); err != nil {
		log.Fatalln(err)
	}

	log.WithFields(log.Fields{"name": consulServiceName}).Info("Registered Consul service for Ambassador")

	kubeTLSSecretName := os.Getenv(EnvSecretName)
	if kubeTLSSecretName == "" {
		kubeTLSSecretName = fmt.Sprintf("%s-consul-connect", consulServiceName)
		log.WithFields(log.Fields{"name": kubeTLSSecretName}).Info("Computed secret name for Ambassador TLS certificate")
	} else {
		log.WithFields(log.Fields{"name": kubeTLSSecretName}).Info("Set secret name for Ambassador TLS certificate")
	}

	kubeTLSSecretNamespace := getEnvOrFallback(EnvSecretNamespace, "")

	if kubeTLSSecretNamespace != "" {
		log.WithFields(log.Fields{"namespace": kubeTLSSecretNamespace}).Info("Ambassador TLS secret will be in specified namespace")
	} else {
		log.Info("Ambassador TLS secret will be in same namespace as this Pod")
	}

	go subscribeToRootCertificateChanges(rootCertChannel, consul.Agent())
	go subscribeToServiceCertificateChanges(leafCertChannel, consulServiceName, consul.Agent())

	createOrUpdateKubernetesTLSCertificateForConsulConnect(
		kubeTLSSecretName,
		kubeTLSSecretNamespace,
		rootCertChannel,
		leafCertChannel)

	log.Info("Ambassador Consul Connect Integration has started!")
	select {}
}

func getEnvOrFallback(name string, fallback string) string {
	if result := os.Getenv(name); result != "" {
		return result
	} else {
		return fallback
	}
}

func createAmbassadorConsulServiceName(ambassadorID string) string {
	base := "ambassador"
	if ambassadorID != "" {
		base += "-" + ambassadorID
	}

	return base
}

func registerAmbassadorAsConsulService(serviceName string, agent *consulapi.Agent) error {
	svc := consulapi.AgentServiceRegistration{
		Name:    serviceName,
		Port:    80,
		Address: "localhost",
	}

	return agent.ServiceRegister(&svc)
}

func createOrUpdateKubernetesTLSCertificateForConsulConnect(secretName string, secretNamespace string, rootCertChan chan rootCert, leafCertChan chan leafCert) {
	var rootCertificate *rootCert
	var leafCertificate *leafCert

	for {
		select {
		case cert := <-rootCertChan:
			rootCertificate = &cert
		case cert := <-leafCertChan:
			leafCertificate = &cert
		}

		if rootCertificate != nil && leafCertificate != nil {
			log.Info("Received root and leaf certificates!")
			chain := createCertificateChain(rootCertificate.CertPEM, leafCertificate.CertPEM, rootCertificate.IntermediateCertsPEM)
			secret := createSecretYAMLDocument(secretName, chain, leafCertificate.PrivateKeyPEM)
			err := applySecret(secretNamespace, secret)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func createCertificateChain(root string, leaf string, intermediaries []string) string {
	result := intermediaries
	result = append(intermediaries, root)
	result = append([]string{leaf}, result...)
	return strings.Join(result, "")
}

func createSecretYAMLDocument(name string, chain string, key string) string {
	chain64 := base64.StdEncoding.EncodeToString([]byte(chain))
	key64 := base64.StdEncoding.EncodeToString([]byte(key))

	return fmt.Sprintf(secretTemplate, name, chain64, key64)
}

func applySecret(namespace string, yaml string) error {
	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		panic(err)
	}

	args := []string{"apply", "-f", "-"}

	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}

	log.WithFields(log.Fields{
		"kubectl": kubectl,
		"args":    args,
	}).Debug("Computed kubectl command and arguments")

	cmd := exec.Command(kubectl, args...)

	var errBuffer bytes.Buffer
	cmd.Stderr = &errBuffer

	cmd.Stdin = bytes.NewBuffer([]byte(yaml))

	_, err = cmd.Output()
	fmt.Println(errBuffer.String())

	return err
}

func subscribeToRootCertificateChanges(ch chan rootCert, agent *consulapi.Agent) {
	currentIndex := uint64(0)

	for {
		log.WithFields(log.Fields{"current-index": currentIndex}).Info("Waiting for Root CA certificate")
		res, meta, err := agent.ConnectCARoots(&consulapi.QueryOptions{
			WaitIndex: currentIndex,
		})

		if err != nil {
			log.Fatalln(err)
		}

		if res == nil || meta == nil {
			time.Sleep(1 * time.Second)
		} else {
			for _, root := range res.Roots {
				if root.Active {
					ch <- rootCert{
						CertPEM:              root.RootCertPEM,
						IntermediateCertsPEM: []string{},
					}
					break
				}
			}

			currentIndex = meta.LastIndex
		}
	}
}

func subscribeToServiceCertificateChanges(ch chan leafCert, service string, agent *consulapi.Agent) {
	currentIndex := uint64(0)

	for {
		log.WithFields(log.Fields{"service": service, "current-index": currentIndex}).Info("Waiting for leaf certificate")
		res, meta, err := agent.ConnectCALeaf(service, &consulapi.QueryOptions{
			WaitIndex: currentIndex,
		})

		if err != nil {
			log.Fatalln(err)
		}

		if res == nil || meta == nil {
			time.Sleep(1 * time.Second)
		} else {
			ch <- leafCert{
				CertPEM:       res.CertPEM,
				PrivateKeyPEM: res.PrivateKeyPEM,
			}
			currentIndex = meta.LastIndex
		}
	}
}
