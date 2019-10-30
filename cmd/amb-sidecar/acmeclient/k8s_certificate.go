// k8s_certificate.go deals with talking to Kubernetes, regarding ACME
// TLS certificates.

package acmeclient

import (
	"crypto/x509"
	"encoding/pem"

	"github.com/go-acme/lego/v3/certificate"
	"github.com/pkg/errors"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func parseTLSSecret(secret *k8sTypesCoreV1.Secret) (*x509.Certificate, error) {
	if secret.Type != k8sTypesCoreV1.SecretTypeTLS {
		return nil, errors.Errorf("secret does not have type %q", k8sTypesCoreV1.SecretTypeTLS)
	}
	data, ok := secret.Data["tls.crt"]
	if !ok {
		return nil, errors.Errorf("secret does not contain a %q field", "tls.crt")
	}
	n := 0
	for len(data) > 0 {
		n++
		var certPem *pem.Block
		certPem, data = pem.Decode(data)
		cert, err := x509.ParseCertificate(certPem.Bytes)
		if err != nil {
			return nil, err
		}
		if !cert.IsCA {
			return cert, nil
		}
	}
	if n == 0 {
		return nil, errors.New("empty certificate chain")
	}
	return nil, errors.Errorf("certificate chain contained %d certificates, but none of them with CA=FALSE", n)
}

func storeCertificate(secretsGetter k8sClientCoreV1.SecretsGetter, name, namespace string, certResource *certificate.Resource) error {
	secretInterface := secretsGetter.Secrets(namespace)

	secret := &k8sTypesCoreV1.Secret{
		ObjectMeta: k8sTypesMetaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: k8sTypesCoreV1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.key": certResource.PrivateKey,
			"tls.crt": certResource.Certificate,
		},
	}

	_, err := secretInterface.Create(secret)
	if err != nil && k8sErrors.IsAlreadyExists(err) {
		_, err = secretInterface.Update(secret)
	}
	return err
}
