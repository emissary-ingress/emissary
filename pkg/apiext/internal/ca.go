package apiext

import (
	// stdlib
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"sync"
	"time"

	// k8s types
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// k8s clients
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// k8s utils
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"

	// 1st-party
	"github.com/datawire/dlib/dlog"
)

const (
	certValidDuration = 365 * 24 * time.Hour
	caSecretName      = "emissary-ingress-webhook-ca"
)

// CA is a Certificat Authority that can mint new TLS certificates.
type CA struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey

	cacheMu sync.Mutex
	cache   map[string]*tls.Certificate
}

// EnsureCA ensures that a Kubernetes Secret named "emissary-ingress-webhook-ca" exists in the given
// namespace (creating it if it doesn't), and returns both the Secret itself and a CA using the
// information from the Secret.
func EnsureCA(ctx context.Context, restConfig *rest.Config, namespace string) (*CA, *k8sTypesCoreV1.Secret, error) {
	coreClient, err := k8sClientCoreV1.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	secretsClient := coreClient.Secrets(namespace)

	for ctx.Err() == nil {
		// Does It already exist?
		caSecret, err := secretsClient.Get(ctx, caSecretName, k8sTypesMetaV1.GetOptions{})
		if err == nil {
			ca, err := parseCA(caSecret)
			if err != nil {
				return nil, nil, err
			}
			return ca, caSecret, nil
		}
		if !k8sErrors.IsNotFound(err) {
			return nil, nil, err
		}

		// Try to create it.
		caSecret, err = genCASecret(namespace)
		if err != nil {
			return nil, nil, err
		}
		caSecret, err = secretsClient.Create(ctx, caSecret, k8sTypesMetaV1.CreateOptions{})
		if err == nil {
			ca, err := parseCA(caSecret)
			if err != nil {
				return nil, nil, err
			}
			return ca, caSecret, nil
		}
		if !k8sErrors.IsAlreadyExists(err) {
			return nil, nil, err
		}

		// Loop around, try again.
	}
	return nil, nil, ctx.Err()
}

func parseCA(caSecret *k8sTypesCoreV1.Secret) (*CA, error) {
	// key
	caKeyPEMBytes, ok := caSecret.Data[k8sTypesCoreV1.TLSPrivateKeyKey]
	if !ok {
		return nil, fmt.Errorf("no key found in CA secret")
	}
	caKeyBlock, _ := pem.Decode(caKeyPEMBytes)
	_caKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("bad key loaded in CA secret: %w", err)
	}
	caKey, ok := _caKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key in CA secret is not an RSA key")
	}

	// cert
	caCertPEMBytes, ok := caSecret.Data[k8sTypesCoreV1.TLSCertKey]
	if !ok {
		return nil, fmt.Errorf("no cert found in CA secret!")
	}
	caCertBlock, _ := pem.Decode(caCertPEMBytes)
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return &CA{
		Cert: caCert,
		Key:  caKey,
	}, nil
}

// genKey generates an RSA key, returning both the key object, as well as a representation of it as
// PEM-encoded PKCS#8 DER.
func genKey() (*rsa.PrivateKey, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	derBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	})
	return key, pemBytes, nil
}

// genCACert generates a Certificate Authority's certificate, returning PEM-encoded DER.
func genCACert(key *rsa.PrivateKey) ([]byte, error) {
	// Generate CA Certificate and key...
	notBefore := time.Now()
	notAfter := notBefore.Add(certValidDuration)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Ambassador Labs"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	return pemBytes, nil
}

func genCASecret(namespace string) (*k8sTypesCoreV1.Secret, error) {
	key, keyPEMBytes, err := genKey()
	if err != nil {
		return nil, err
	}
	certPEMBytes, err := genCACert(key)
	if err != nil {
		return nil, err
	}
	return &k8sTypesCoreV1.Secret{
		ObjectMeta: k8sTypesMetaV1.ObjectMeta{
			Name:      caSecretName,
			Namespace: namespace,
		},
		Type: k8sTypesCoreV1.SecretTypeTLS,
		Data: map[string][]byte{
			k8sTypesCoreV1.TLSPrivateKeyKey: keyPEMBytes,
			k8sTypesCoreV1.TLSCertKey:       certPEMBytes,
		},
	}, nil
}

func (ca *CA) GenServerCert(ctx context.Context, hostname string) (*tls.Certificate, error) {
	dlog.Debugf(ctx, "GenServerCert(ctx, %q)", hostname)

	ca.cacheMu.Lock()
	defer ca.cacheMu.Unlock()

	if ca.cache == nil {
		ca.cache = make(map[string]*tls.Certificate)
	}

	now := time.Now()
	if cached, ok := ca.cache[hostname]; ok && cached != nil && cached.Leaf != nil {
		if age, lifespan := now.Sub(cached.Leaf.NotBefore), cached.Leaf.NotAfter.Sub(cached.Leaf.NotBefore); age < 2*lifespan/3 {
			dlog.Debugf(ctx, "GenServerCert(ctx, %q) => from cache (age=%v lifespan=%v)", hostname, age, lifespan)
			return cached, nil
		} else {
			dlog.Debugf(ctx, "GenServerCert(ctx, %q) => cache entry too old (age=%v lifespan=%v)", hostname, age, lifespan)
		}
	}
	dlog.Infof(ctx, "GenServerCert(ctx, %q) => generating new cert", hostname)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Ambassador Labs"},
			CommonName:   "Webhook API",
		},
		NotBefore:             now,
		NotAfter:              now.Add(certValidDuration),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{hostname},
	}

	certPEMBytes, err := x509.CreateCertificate(
		rand.Reader,
		cert,
		ca.Cert,
		priv.Public(),
		ca.Key,
	)
	if err != nil {
		return nil, err
	}

	certChain := &tls.Certificate{
		Certificate: [][]byte{certPEMBytes},
		PrivateKey:  priv,
		Leaf:        cert,
	}

	ca.cache[hostname] = certChain
	return certChain, nil
}
