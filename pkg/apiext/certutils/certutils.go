package certutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// GenerateRootCACert will generate a private key and a basic RootCA
func GenerateRootCACert(organization string, validDuration time.Duration) (privateKeyPem []byte, certPem []byte, err error) {
	var rsaPrivateKey *rsa.PrivateKey

	rsaPrivateKey, privateKeyPem, err = genKey()
	if err != nil {
		return privateKeyPem, certPem, err
	}
	certPem, err = genCACert(rsaPrivateKey, organization, validDuration)
	if err != nil {
		return privateKeyPem, certPem, err
	}

	return privateKeyPem, certPem, nil
}

func ParseCASecret(secret *corev1.Secret) (*rsa.PrivateKey, *x509.Certificate, error) {
	if secret == nil || secret.Data == nil {
		return nil, nil, fmt.Errorf("invalid secret, unable to parse")
	}
	privateKeyPEMBytes, ok := secret.Data[corev1.TLSPrivateKeyKey]
	if !ok {
		return nil, nil, fmt.Errorf("%q not found in secret", corev1.TLSPrivateKeyKey)
	}

	certPEMBytes, ok := secret.Data[corev1.TLSCertKey]
	if !ok {
		return nil, nil, fmt.Errorf("%q not found in secret", corev1.TLSCertKey)
	}

	return ParseCAPemBytes(privateKeyPEMBytes, certPEMBytes)
}

func ParseCAPemBytes(privateKeyPEMBytes []byte, certPEMBytes []byte) (*rsa.PrivateKey, *x509.Certificate, error) {
	caKeyBlock, _ := pem.Decode(privateKeyPEMBytes)
	caKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse private key PEM using PKCS8: %w", err)
	}

	privateKey, ok := caKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, fmt.Errorf("private key not a valid RSA key")
	}

	caCertBlock, _ := pem.Decode(certPEMBytes)
	cert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("public certificate data found is not a valid x509 certificate: %w", err)
	}

	if !cert.IsCA {
		return nil, nil, fmt.Errorf("public x509 certificate is not marked as a CA")
	}

	if time.Now().Before(cert.NotBefore) {
		return nil, nil, fmt.Errorf("current time is before the root x509 ca certificate notBefore time")
	}

	return privateKey, cert, nil
}

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
func genCACert(key *rsa.PrivateKey, subject string, validDuration time.Duration) ([]byte, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(validDuration)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{subject},
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
