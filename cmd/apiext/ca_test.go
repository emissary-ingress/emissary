package apiext

import (
	"testing"

	"github.com/stretchr/testify/require"
	k8sTypesCoreV1 "k8s.io/api/core/v1"

	"github.com/datawire/dlib/dlog"
)

func TestCA(t *testing.T) {
	caKey, caKeyBytes, err := genKey()
	require.NoError(t, err)
	require.NotNil(t, caKey)
	require.True(t, len(caKeyBytes) > 0, "caKeyBytes should be non-empty")

	caCertBytes, err := genCACert(caKey)
	require.NoError(t, err)
	require.True(t, len(caCertBytes) > 0, "caCertBytes should be non-empty")

	ca, err := parseCA(&k8sTypesCoreV1.Secret{
		Type: k8sTypesCoreV1.SecretTypeTLS,
		Data: map[string][]byte{
			k8sTypesCoreV1.TLSPrivateKeyKey: caKeyBytes,
			k8sTypesCoreV1.TLSCertKey:       caCertBytes,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, ca)

	ctx := dlog.NewTestContext(t, true)

	a, err := ca.GenServerCert(ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, a)

	b, err := ca.GenServerCert(ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, b)

	// pointer equality
	require.True(t, a == b, "because of caching, certs should be pointer-equal")
}
