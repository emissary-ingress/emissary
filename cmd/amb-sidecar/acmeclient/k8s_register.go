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

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	ambassadorTypesV2 "github.com/datawire/ambassador/pkg/api/getambassador.io/v2"
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func createUserPrivateKey(secretsGetter k8sClientCoreV1.SecretsGetter, namespace, name string) error {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}
	privateKeyPEMBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	_, err = secretsGetter.Secrets(namespace).Create(&k8sTypesCoreV1.Secret{
		ObjectMeta: k8sTypesMetaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: k8sTypesCoreV1.SecretTypeOpaque,
		Data: map[string][]byte{
			"user.key": privateKeyPEMBytes,
		},
	})
	// Ignore already-exists errors, because (1) there are other
	// replicas, and coordinating is hard (with a little-H?), and
	// (2) it means that the code that de-duplicates acme
	// providers canb e simpler.
	if err != nil && k8sErrors.IsAlreadyExists(err) {
		return nil
	}
	return err
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

func (c *Controller) userRegister(namespace string, spec *ambassadorTypesV2.ACMEProviderSpec) error {
	privateKeySecret := c.getSecret(namespace, spec.PrivateKeySecret.Name)
	privateKey, err := parseUserPrivateKey(privateKeySecret)
	if err != nil {
		return err
	}
	user, err := registerUser(c.httpClient,
		spec.Authority,
		spec.Email,
		privateKey)
	if err != nil {
		return err
	}
	reg, err := json.Marshal(user.GetRegistration())
	if err != nil {
		return err
	}
	spec.Registration = string(reg)
	return nil
}
