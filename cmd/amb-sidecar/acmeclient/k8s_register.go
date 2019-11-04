// k8s_register.go deals with talking to Kubernetes, regarding ACME user registration.

package acmeclient

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"

	"github.com/pkg/errors"

	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateUserPrivateKeySecret(namespace, name string) (*k8sTypesCoreV1.Secret, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	privateKeyPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return &k8sTypesCoreV1.Secret{
		ObjectMeta: k8sTypesMetaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: k8sTypesCoreV1.SecretTypeOpaque,
		Data: map[string][]byte{
			"user.key": privateKeyPEMBytes,
		},
	}, nil
}

func parseUserPrivateKey(secret *k8sTypesCoreV1.Secret) (crypto.PrivateKey, error) {
	privateKeyPEMBytes, ok := secret.Data["user.key"]
	if !ok {
		return nil, errors.Errorf("secret name=%q namespace=%q: exists but does not contain an %q %s field",
			secret.GetName(), secret.GetNamespace(),
			"user.key", "private-key")
	}
	privateKeyPEM, _ := pem.Decode(privateKeyPEMBytes)
	privateKey, err := x509.ParseECPrivateKey(privateKeyPEM.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "secret name=%q namespace=%q: parse private-key",
			secret.GetName(), secret.GetNamespace())
	}
	return privateKey, nil
}

func (c *Controller) userRegister(namespace string, spec *ambassadorTypesV2.ACMEProviderSpec) (string, error) {
	privateKeySecret := c.getSecret(namespace, spec.PrivateKeySecret.Name)
	privateKey, err := parseUserPrivateKey(privateKeySecret)
	if err != nil {
		return "", err
	}
	user, err := registerUser(c.httpClient,
		spec.Authority,
		spec.Email,
		privateKey)
	if err != nil {
		return "", err
	}
	reg, err := json.Marshal(user.GetRegistration())
	if err != nil {
		return "", err
	}
	return string(reg), nil
}
