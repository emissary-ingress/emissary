package ca

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/emissary-ingress/emissary/v3/pkg/apiext/certutils"
	corev1 "k8s.io/api/core/v1"
)

type CACert struct {
	CertificatePEM []byte
	Certifcate     *x509.Certificate
	PrivateKey     *rsa.PrivateKey
}

func NewCACert(organization string, validDuration time.Duration) (CACert, error) {
	privateKeyBytes, certBytes, err := certutils.GenerateRootCACert("apiext-unit-test", 1*time.Hour)
	if err != nil {
		return CACert{}, nil
	}
	return caCertFromPEMBytes(privateKeyBytes, certBytes)

}

// CACertFromSecret generates a CACert from a Secret. If the Secret contains
// invalid data or an expired cert then it will return an error.
func CACertFromSecret(secret *corev1.Secret) (CACert, error) {
	privateKey, certificate, err := certutils.ParseCASecret(secret)
	if err != nil {
		return CACert{}, err
	}

	if time.Now().After(certificate.NotAfter) {
		return CACert{}, fmt.Errorf("root ca certifcate is expired")
	}

	return CACert{
		CertificatePEM: secret.Data[corev1.TLSCertKey],
		Certifcate:     certificate,
		PrivateKey:     privateKey,
	}, nil
}

// caCertFromPEMBytes parses, validates the PemBytes to return a CACert
func caCertFromPEMBytes(privateKeyPem []byte, certPem []byte) (CACert, error) {
	privateKey, certificate, err := certutils.ParseCAPemBytes(privateKeyPem, certPem)
	if err != nil {
		return CACert{}, nil
	}

	if time.Now().After(certificate.NotAfter) {
		return CACert{}, fmt.Errorf("root ca certifcate is expired")
	}

	return CACert{
		CertificatePEM: certPem,
		Certifcate:     certificate,
		PrivateKey:     privateKey,
	}, nil
}
