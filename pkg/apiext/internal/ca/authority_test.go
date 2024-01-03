package ca_test

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/ca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// func TestCA(t *testing.T) {
// 	caKey, caKeyBytes, err := genKey()
// 	require.NoError(t, err)
// 	require.NotNil(t, caKey)
// 	require.True(t, len(caKeyBytes) > 0, "caKeyBytes should be non-empty")

// 	caCertBytes, err := genCACert(caKey)
// 	require.NoError(t, err)
// 	require.True(t, len(caCertBytes) > 0, "caCertBytes should be non-empty")

// 	ca, err := parseCA(&k8sTypesCoreV1.Secret{
// 		Type: k8sTypesCoreV1.SecretTypeTLS,
// 		Data: map[string][]byte{
// 			k8sTypesCoreV1.TLSPrivateKeyKey: caKeyBytes,
// 			k8sTypesCoreV1.TLSCertKey:       caCertBytes,
// 		},
// 	})
// 	require.NoError(t, err)
// 	require.NotNil(t, ca)

// 	ctx := dlog.NewTestContext(t, true)

// 	a, err := ca.GenServerCert(ctx, "foo")
// 	require.NoError(t, err)
// 	require.NotNil(t, a)

// 	b, err := ca.GenServerCert(ctx, "foo")
// 	require.NoError(t, err)
// 	require.NotNil(t, b)

// 	// pointer equality
// 	require.True(t, a == b, "because of caching, certs should be pointer-equal")
// }

func TestReady(t *testing.T) {
	certAuthority := ca.NewAPIExtCertificateAuthority(zaptest.NewLogger(t))
	assert.False(t, certAuthority.Ready())

	certAuthority.SetCACert(&ca.CACert{})
	assert.True(t, certAuthority.Ready())

	certAuthority.SetCACert(nil)
	assert.False(t, certAuthority.Ready())
}

func TestCertificateAuthorityCache(t *testing.T) {
	certAuthority := ca.NewAPIExtCertificateAuthority(zaptest.NewLogger(t))

	assert.Nil(t, certAuthority.GetCACert())

	caCert, err := ca.NewCACert("apiext-unit-teset", 1*time.Hour)
	assert.NoError(t, err)

	certAuthority.SetCACert(&caCert)
	require.True(t, &caCert == certAuthority.GetCACert())

	caCert2, err := ca.NewCACert("apiext-unit-test-2", 1*time.Hour)
	assert.NoError(t, err)

	certAuthority.SetCACert(&caCert2)
	require.False(t, &caCert == certAuthority.GetCACert())
	require.True(t, &caCert2 == certAuthority.GetCACert())
}

func TestGetCertificateCARotation(t *testing.T) {
	certAuthority := ca.NewAPIExtCertificateAuthority(zaptest.NewLogger(t))

	clientHello := &tls.ClientHelloInfo{
		ServerName: "ambassador",
	}

	serverCert, err := certAuthority.GetCertificate(nil)
	require.Nil(t, serverCert)
	require.Error(t, err)
	require.ErrorIs(t, err, ca.InvalidClientHelloErr)

	caCert, err := ca.NewCACert("apiext-unit-test", 1*time.Hour)
	require.NoError(t, err)
	certAuthority.SetCACert(&caCert)

	serverCert, err = certAuthority.GetCertificate(clientHello)
	require.NoError(t, err)
	require.NotNil(t, serverCert)

	serverCert2, err := certAuthority.GetCertificate(clientHello)
	require.NoError(t, err)
	require.NotNil(t, serverCert2)
	require.True(t, serverCert == serverCert2)

	renewedCACert, err := ca.NewCACert("apiext-unit-test-rotated-ca", 1*time.Hour)
	require.NoError(t, err)
	certAuthority.SetCACert(&renewedCACert)

	// internally cached server certs should be invalidated and a new server cert generated
	serverCert3, err := certAuthority.GetCertificate(clientHello)
	require.NoError(t, err)
	require.NotNil(t, serverCert2)
	require.True(t, serverCert != serverCert3)
	require.True(t, serverCert2 != serverCert3)
}

func TestGetCertificateCachesMultipleCerts(t *testing.T) {
	certAuthority := ca.NewAPIExtCertificateAuthority(zaptest.NewLogger(t))

	clientHello1 := &tls.ClientHelloInfo{
		ServerName: "ambassador",
	}

	clientHello2 := &tls.ClientHelloInfo{
		ServerName: "ambassador-host2",
	}

	caCert, err := ca.NewCACert("apiext-unit-test", 1*time.Hour)
	require.NoError(t, err)
	certAuthority.SetCACert(&caCert)

	serverCert, err := certAuthority.GetCertificate(clientHello1)
	require.NoError(t, err)
	require.NotNil(t, serverCert)

	serverCert2, err := certAuthority.GetCertificate(clientHello2)
	require.NoError(t, err)
	require.NotNil(t, serverCert)
	require.True(t, serverCert != serverCert2)
}
