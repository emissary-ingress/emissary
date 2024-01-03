package ca

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/emissary-ingress/emissary/v3/pkg/apiext/defaults"
	"go.uber.org/zap"
)

const (
	defaultServerCertValidDuration = 14 * 24 * time.Hour
)

var (
	InvalidClientHelloErr = errors.New("invalid ClientHello received, unable to determine server name")
)

type CertificateAuthority interface {
	// SetCACert updates the CACert used for generating server certificates
	SetCACert(*CACert)
	// GetCACert returns the currently stored CACert
	GetCACert() *CACert
	// GetCertificates matches `crypto/tls` to provide a valid server cert for listening to incoming connections
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
	// Ready indicates that the CA is ready to provide server certificates
	Ready() bool
}

type apiextCertificateAuthority struct {
	logger   *zap.Logger
	caCert   *CACert
	caCertMu sync.RWMutex

	serverCerts   map[string]*tls.Certificate
	serverCertsMu sync.RWMutex
}

var _ CertificateAuthority = (*apiextCertificateAuthority)(nil)

func NewAPIExtCertificateAuthority(logger *zap.Logger) *apiextCertificateAuthority {
	return &apiextCertificateAuthority{
		logger:      logger.Named("apiext-cert-authority"),
		serverCerts: make(map[string]*tls.Certificate),
	}
}

// GetCABundle implements CertificateAuthority.
func (a *apiextCertificateAuthority) GetCACert() *CACert {
	a.caCertMu.RLock()
	cert := a.caCert
	a.caCertMu.RUnlock()

	return cert
}

// SetCABundle implements CertificateAuthority.
func (a *apiextCertificateAuthority) SetCACert(caCert *CACert) {
	a.caCertMu.Lock()
	a.caCert = caCert
	a.caCertMu.Unlock()

	a.invalidateServerCerts()
}

// GetCertificate implements CertificateAuthority.
func (a *apiextCertificateAuthority) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	a.logger.Info("receivedGetCertificate call", zap.Any("clientHello", clientHello))
	if clientHello == nil {
		return nil, InvalidClientHelloErr
	}
	cachedCert := a.getCachedCert(clientHello.ServerName)
	if cachedCert != nil {
		a.logger.Info("reusing cached server cert", zap.String("serverName", clientHello.ServerName))
		return cachedCert, nil
	}

	return a.generateServerCert(context.Background(), clientHello.ServerName)
}

func (a *apiextCertificateAuthority) Ready() bool {
	if a.GetCACert() == nil {
		return false
	}

	return true
}

func (a *apiextCertificateAuthority) getCachedCert(serverName string) *tls.Certificate {
	a.serverCertsMu.RLock()
	defer a.serverCertsMu.RUnlock()

	cachedCert, found := a.serverCerts[serverName]
	if !found || cachedCert == nil || cachedCert.Leaf == nil {
		return nil
	}

	age := time.Now().Sub(cachedCert.Leaf.NotBefore)
	lifespan := cachedCert.Leaf.NotAfter.Sub(cachedCert.Leaf.NotBefore)
	leeway := 2 * lifespan / 3
	if age < leeway {
		a.logger.Debug("using cached server cert",
			zap.String("serverName", serverName),
			zap.Duration("age", age),
			zap.Duration("lifespan", lifespan),
			zap.Duration("leeway", leeway),
		)
		return cachedCert
	}

	a.logger.Debug("cached server cert is within expiration leeway, generating new server cert",
		zap.String("serverName", serverName),
		zap.Duration("age", age),
		zap.Duration("lifespan", lifespan),
		zap.Duration("leeway", leeway),
	)

	return nil
}

// genServerCert will provide hostname a server cert from the cache or will
// generate a new one using the CA Certificate
func (a *apiextCertificateAuthority) generateServerCert(ctx context.Context, serverName string) (*tls.Certificate, error) {
	a.logger.Info("generating new server cert", zap.String("serverName", serverName))

	now := time.Now()
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
			Organization: []string{defaults.SubjectOrganization},
			CommonName:   "Webhook API",
		},
		NotBefore:             now,
		NotAfter:              now.Add(defaultServerCertValidDuration),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{serverName},
	}

	caCert := a.GetCACert()
	if caCert == nil {
		a.logger.Error("ca bundle is missing, unable to generate new server certificate", zap.String("serverName", serverName))
		return nil, fmt.Errorf("ca bundle is missing, unable generate new server certifcate for %s", serverName)
	}

	certPEMBytes, err := x509.CreateCertificate(
		rand.Reader,
		cert,
		caCert.Certifcate,
		priv.Public(),
		caCert.PrivateKey,
	)
	if err != nil {
		return nil, err
	}

	serverCert := &tls.Certificate{
		Certificate: [][]byte{certPEMBytes},
		PrivateKey:  priv,
		Leaf:        cert,
	}

	a.setServerCert(serverName, serverCert)

	return serverCert, nil
}

func (a *apiextCertificateAuthority) setServerCert(serverName string, serverCert *tls.Certificate) {
	a.serverCertsMu.Lock()
	a.serverCerts[serverName] = serverCert
	a.serverCertsMu.Unlock()
}

func (a *apiextCertificateAuthority) invalidateServerCerts() {
	a.serverCertsMu.Lock()
	a.serverCerts = make(map[string]*tls.Certificate)
	a.serverCertsMu.Unlock()
}
