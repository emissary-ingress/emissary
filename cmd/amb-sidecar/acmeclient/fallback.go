// fallback.go deals with creating the fallback self-signed
// certificate, for when we receive a request to a domain that hasn't
// had a Host resource set up for it.

package acmeclient

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"time"

	"github.com/pkg/errors"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"

	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypesUnstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8sClientDynamic "k8s.io/client-go/dynamic"
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/ambassador/pkg/k8s"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

const (
	SelfSignedSecretName  = "fallback-self-signed-cert"
	SelfSignedContextName = "fallback-self-signed-context"
)

func EnsureFallback(cfg types.Config, kubeinfo *k8s.KubeInfo) error {
	restconfig, err := kubeinfo.GetRestConfig()
	if err != nil {
		return err
	}
	coreClient, err := k8sClientCoreV1.NewForConfig(restconfig)
	if err != nil {
		return err
	}
	dynamicClient, err := k8sClientDynamic.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	if err := ensureFallbackSecret(cfg, coreClient); err != nil {
		return err
	}
	if err := ensureFallbackContext(cfg, dynamicClient); err != nil {
		return err
	}
	return nil
}

func ensureFallbackContext(cfg types.Config, dynamicClient k8sClientDynamic.Interface) error {
	tlsContextGetter := dynamicClient.Resource(k8sSchema.GroupVersionResource{Group: "getambassador.io", Version: "v1", Resource: "tlscontexts"})
	tlsContextInterface := tlsContextGetter.Namespace(cfg.AmbassadorNamespace)
	_, err := tlsContextInterface.Create(&k8sTypesUnstructured.Unstructured{map[string]interface{}{
		"apiVersion": "getambassador.io/v1",
		"kind":       "TLSContext",
		"metadata": map[string]string{
			"name":      SelfSignedContextName,
			"namespace": cfg.AmbassadorNamespace,
		},
		"spec": map[string]interface{}{
			"hosts": []string{
				"*",
			},
			"secret": SelfSignedSecretName,
		},
	}}, k8sTypesMetaV1.CreateOptions{})
	if err != nil && !k8sErrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func ensureFallbackSecret(cfg types.Config, secretsGetter k8sClientCoreV1.SecretsGetter) error {
	secretInterface := secretsGetter.Secrets(cfg.AmbassadorNamespace)
	for {
		_, err := secretInterface.Get(SelfSignedSecretName, k8sTypesMetaV1.GetOptions{})
		if err == nil {
			// already done; nothing to do
			return nil
		}
		if !k8sErrors.IsNotFound(err) {
			return err
		}
		// Try to create the secret, but ignore already-exists
		// errors because there might be other replicas doing
		// the same thing.
		privatePEM, publicPEM, err := generateSelfSignedPEM()
		if err != nil {
			return err
		}
		_, err = secretInterface.Create(&k8sTypesCoreV1.Secret{
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      SelfSignedSecretName,
				Namespace: cfg.AmbassadorNamespace,
			},
			Type: k8sTypesCoreV1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.key": privatePEM,
				"tls.crt": publicPEM,
			},
		})
		if err != nil && !k8sErrors.IsAlreadyExists(err) {
			return err
		}
		return nil
		// fall-through / retry
	}
}

func generateSelfSignedPEM() ([]byte, []byte, error) {
	// See https://golang.org/src/crypto/tls/generate_cert.go

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate key-pair")
	}
	privateKeyDERBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "marshal private key")
	}
	privateKeyPEM := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDERBytes,
	}
	privateKeyPEMBytes := pem.EncodeToMemory(privateKeyPEM)

	publicKey := privateKey.Public()

	notBefore := time.Now()
	// I don't want to write code to update the friggin'
	// *fallback* certificate.  Just let it last impossibly long.
	notAfter := notBefore.Add(100 * 365 * 24 * time.Hour)

	// Generate a random 128-bit number for the serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate serial number")
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,

		Subject: pkix.Name{Organization: []string{"Ambassador Edge Stack Self-Signed"}},
		// Subject Alternative Name
		DNSNames:       []string{""}, // I'm not sure why we need "", but Caddy includes it in their self-signed certs, so mimic that
		EmailAddresses: []string{},
		IPAddresses:    []net.IP{},
		URIs:           []*url.URL{},

		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDERBytes, err := x509.CreateCertificate(
		rand.Reader, // rand
		template,    // template
		template,    // parent
		publicKey,   // pub
		privateKey)  // priv
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate certificate")
	}
	certPEM := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDERBytes,
	}
	certPEMBytes := pem.EncodeToMemory(certPEM)

	return privateKeyPEMBytes, certPEMBytes, nil
}
