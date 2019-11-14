package secret

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

const SecretName = "ambassador-internal"

// GetKeyPair loads an RSA key pair from a Kubernetes secret (named by
// `SecretName` and `cfg.PodNamespace`).  If the secret does not yet
// exist, it attempts to create it in a way that should be safe for
// multiple replicas to be trying the same thing.
//
// Will return errors for things like RBAC failures or malformed keys.
func GetKeyPair(cfg types.Config, secretsGetter k8sClientCoreV1.SecretsGetter) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	secretInterface := secretsGetter.Secrets(cfg.PodNamespace)
	for {
		secret, err := secretInterface.Get(SecretName, k8sTypesMetaV1.GetOptions{})
		if err == nil {
			privatePEM, ok := secret.Data["rsa.key"]
			if !ok {
				return nil, nil, errors.Errorf("secret name=%q namespace=%q exists but does not contain an %q %s field",
					SecretName, cfg.PodNamespace, "rsa.key", "private-key")
			}
			publicPEM, ok := secret.Data["rsa.crt"]
			if !ok {
				return nil, nil, errors.Errorf("secret name=%q namespace=%q exists but does not contain an %q %s field",
					SecretName, cfg.PodNamespace, "rsa.crt", "public-key")
			}
			return parsePEM(privatePEM, publicPEM)
		}
		if !k8sErrors.IsNotFound(err) {
			return nil, nil, err
		}
		// Try to create the secret, but ignore already-exists
		// errors because there might be other replicas doing
		// the same thing.
		privatePEM, publicPEM, err := generatePEM()
		if err != nil {
			return nil, nil, err
		}
		_, err = secretInterface.Create(&k8sTypesCoreV1.Secret{
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      SecretName,
				Namespace: cfg.PodNamespace,
			},
			Type: k8sTypesCoreV1.SecretTypeOpaque,
			Data: map[string][]byte{
				"rsa.key": privatePEM,
				"rsa.crt": publicPEM,
			},
		})
		if err != nil && !k8sErrors.IsAlreadyExists(err) {
			return nil, nil, err
		}
		// fall-through / retry
	}
}

func parsePEM(privatePEM, publicPEM []byte) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse private-key")
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicPEM)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parse public-key")
	}
	return privateKey, publicKey, nil
}

func generatePEM() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate key-pair")
	}
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privateKeyPEMBytes := pem.EncodeToMemory(privateKeyPEM)

	publicKey := privateKey.Public()
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, errors.Wrap(err, "generate key-pair")
	}
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyPEMBytes := pem.EncodeToMemory(publicKeyPEM)

	return privateKeyPEMBytes, publicKeyPEMBytes, nil
}
