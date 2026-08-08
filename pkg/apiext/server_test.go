package apiext

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"

	"github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/ca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// fakeCertificateAuthority is a test double for ca.CertificateAuthority that lets us assert on
// how the WebhookServer's readiness check drives certificate generation, without needing a real
// CA or paying the cost of RSA key generation.
type fakeCertificateAuthority struct {
	caReady             bool
	getCertificateErr   error
	getCertificateCalls []string
}

func (f *fakeCertificateAuthority) SetCACert(*ca.CACert) {}

func (f *fakeCertificateAuthority) GetCACert() *ca.CACert { return nil }

func (f *fakeCertificateAuthority) Ready() bool { return f.caReady }

func (f *fakeCertificateAuthority) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	f.getCertificateCalls = append(f.getCertificateCalls, hello.ServerName)
	if f.getCertificateErr != nil {
		return nil, f.getCertificateErr
	}
	return &tls.Certificate{}, nil
}

var _ ca.CertificateAuthority = (*fakeCertificateAuthority)(nil)

func TestWebhookServerWebhookHostname(t *testing.T) {
	server := NewWebhookServer(zaptest.NewLogger(t), "emissary-apiext", WithNamespace("emissary-system"))
	assert.Equal(t, "emissary-apiext.emissary-system.svc", server.webhookHostname)
}

func TestWebhookServerReadyWaitsForCACert(t *testing.T) {
	fake := &fakeCertificateAuthority{caReady: false}
	server := &WebhookServer{
		logger:               zaptest.NewLogger(t),
		certificateAuthority: fake,
		webhookHostname:      "emissary-apiext.emissary-system.svc",
	}

	done, err := server.ready(context.Background())
	require.NoError(t, err)
	assert.False(t, done)
	assert.Empty(t, fake.getCertificateCalls, "should not attempt to generate a server cert before the CA is ready")
}

func TestWebhookServerReadyEagerlyGeneratesServerCertificate(t *testing.T) {
	fake := &fakeCertificateAuthority{caReady: true}
	server := &WebhookServer{
		logger:               zaptest.NewLogger(t),
		certificateAuthority: fake,
		webhookHostname:      "emissary-apiext.emissary-system.svc",
	}

	done, err := server.ready(context.Background())
	require.NoError(t, err)
	assert.True(t, done, "server should be ready once the CA cert and server cert are both prepared")
	require.Len(t, fake.getCertificateCalls, 1)
	assert.Equal(t, "emissary-apiext.emissary-system.svc", fake.getCertificateCalls[0],
		"should eagerly generate a server cert for our well-known webhook hostname")
}

func TestWebhookServerReadyNotReadyIfServerCertGenerationFails(t *testing.T) {
	fake := &fakeCertificateAuthority{caReady: true, getCertificateErr: errors.New("boom")}
	server := &WebhookServer{
		logger:               zaptest.NewLogger(t),
		certificateAuthority: fake,
		webhookHostname:      "emissary-apiext.emissary-system.svc",
	}

	done, err := server.ready(context.Background())
	require.NoError(t, err)
	assert.False(t, done, "should not report ready if we failed to eagerly generate the server certificate")
}
