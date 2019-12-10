// k8s_certificate.go deals with talking to Kubernetes, regarding ACME
// TLS certificates.

package acmeclient

import (
	"crypto/x509"
	"encoding/pem"

	"github.com/pkg/errors"

	k8sTypesCoreV1 "k8s.io/api/core/v1"
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
