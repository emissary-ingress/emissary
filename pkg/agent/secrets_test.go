package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	errorsv1 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"
	"testing"
)

func TestRunWithClientFactorySet(t *testing.T) {
	t.Run("No value", func(t *testing.T) {
		// given
		cmd := wrapNewCommand("api-keys-staging", secretSyncActionSet, nil)
		secretGetter := newSecretGetterMock()
		cmd.secret = nil

		// when
		err := cmd.RunWithClientFactory(
			context.Background(), wrapSecretGetterFactoryMock(secretGetter))

		// then
		assert.NoError(t, err, "no error")
		assert.Equal(t, 4, len(secretGetter.secrets), "empty secret created")
	})
	t.Run("Success empty secret", func(t *testing.T) {
		// given
		cmd := wrapNewCommand("api-keys-staging", secretSyncActionSet, nil)
		secretGetter := newSecretGetterMock()
		cmd.secret = map[string][]byte{}

		// when
		err := cmd.RunWithClientFactory(
			context.Background(), wrapSecretGetterFactoryMock(secretGetter))

		// then
		createdSecret := secretGetter.findSecret("api-keys-staging")
		assert.NoError(t, err, "no error")
		assert.Equal(t, len(createdSecret.Data), 0)
		assert.Equal(t, 4, len(secretGetter.secrets), "empty secret created")

	})
	t.Run("Success", func(t *testing.T) {
		// given
		cmd := wrapNewCommand("api-keys-staging", secretSyncActionSet, map[string][]byte{
			"key-1": []byte("1234"),
			"key-2": []byte("5678"),
		})
		secretGetter := newSecretGetterMock()

		// when
		err := cmd.RunWithClientFactory(
			context.Background(), wrapSecretGetterFactoryMock(secretGetter))

		// then
		createdSecret := secretGetter.findSecret("api-keys-staging")
		assert.NoError(t, err, "no error")
		assert.NotNil(t, createdSecret)
		assert.Equal(t, 2, len(createdSecret.Data))
		assert.Equal(t, 4, len(secretGetter.secrets), "new secret created")
	})
	t.Run("Already exists", func(t *testing.T) {
		// given
		cmd := wrapNewCommand("some-existing-api-key", secretSyncActionSet, nil)
		secretGetter := newSecretGetterMock()

		// when
		err := cmd.RunWithClientFactory(
			context.Background(), wrapSecretGetterFactoryMock(secretGetter))

		// then
		assert.NoError(t, err, "no error")
		assert.Equal(t, 3, len(secretGetter.secrets), "no new secrets")
	})
	t.Run("Random error existing secret", func(t *testing.T) {
		// given
		cmd := wrapNewCommand("random-error-patch", secretSyncActionSet, nil)
		secretGetter := newSecretGetterMock()

		// when
		err := cmd.RunWithClientFactory(
			context.Background(), wrapSecretGetterFactoryMock(secretGetter))

		// then
		assert.Error(t, err, "fail when client error")
	})
	t.Run("Random error new secret", func(t *testing.T) {
		// given
		cmd := wrapNewCommand("random-error-create", secretSyncActionSet, nil)
		secretGetter := newSecretGetterMock()

		// when
		err := cmd.RunWithClientFactory(
			context.Background(), wrapSecretGetterFactoryMock(secretGetter))

		// then
		assert.Error(t, err, "fail when client error")
	})
}

func TestRunWithClientFactoryDelete(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// given
		cmd := wrapNewCommand("some-existing-api-key", secretSyncActionDelete, map[string][]byte{
			"some-secret": []byte("1234"),
		})
		secretGetter := newSecretGetterMock()

		// when
		err := cmd.RunWithClientFactory(
			context.Background(), wrapSecretGetterFactoryMock(secretGetter))

		// then
		assert.NoError(t, err, "no error")
		assert.Equal(t, 2, len(secretGetter.secrets), "secret deleted")
	})
	t.Run("Not found", func(t *testing.T) {
		// given
		cmd := wrapNewCommand("some-non-existing-api-key", secretSyncActionDelete, nil)
		secretGetter := newSecretGetterMock()

		// when
		err := cmd.RunWithClientFactory(
			context.Background(), wrapSecretGetterFactoryMock(secretGetter))

		// then
		assert.NoError(t, err, "no error")
		assert.Equal(t, 3, len(secretGetter.secrets), "nothing deleted")
	})
	t.Run("Random error", func(t *testing.T) {
		// given
		cmd := wrapNewCommand("random-error-delete", secretSyncActionDelete, nil)
		secretGetter := newSecretGetterMock()

		// when
		err := cmd.RunWithClientFactory(
			context.Background(), wrapSecretGetterFactoryMock(secretGetter))

		// then
		assert.Error(t, err, "fail when client error")
	})

}

func wrapNewCommand(name string, action secretSyncAction, secret map[string][]byte) *secretSyncCommand {
	if secret == nil {
		secret = map[string][]byte{
			"key-1": []byte("1234"),
		}
	}
	return &secretSyncCommand{
		name:      name,
		namespace: "ambassador",
		action:    action,
		secret:    secret,
	}
}

func newSecretGetterMock() *secretGetterMock {
	return &secretGetterMock{
		secrets: []*apiv1.Secret{
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-existing-api-key",
					Namespace: "ambassador",
				},
				Data: map[string][]byte{
					"some-secret": []byte("some-value"),
				},
				Type: apiv1.SecretTypeOpaque,
			},
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "random-error-patch",
					Namespace: "ambassador",
				},
				Data: map[string][]byte{
					"some-secret": []byte("some-value"),
				},
				Type: apiv1.SecretTypeOpaque,
			},
			{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "random-error-delete",
					Namespace: "ambassador",
				},
				Data: map[string][]byte{},
				Type: apiv1.SecretTypeOpaque,
			},
		},
	}
}

func wrapSecretGetterFactoryMock(secretGetter *secretGetterMock) func(namespace string) (SecretInterface, error) {
	secretGetterFactoryMock := func(namespace string) (SecretInterface, error) {
		secretGetter.Namespace = namespace
		return secretGetter, nil
	}
	return secretGetterFactoryMock
}

type secretGetterMock struct {
	Namespace string

	secrets []*apiv1.Secret
}

func (s *secretGetterMock) findSecret(name string) *apiv1.Secret {
	for i := range s.secrets {
		if s.secrets[i].Name == name && s.secrets[i].Namespace == s.Namespace {
			return s.secrets[i]
		}
	}
	return nil
}

func (s *secretGetterMock) Get(ctx context.Context, name string, opts metav1.GetOptions) (*apiv1.Secret, error) {
	if strings.Contains(name, "random-error-get") {
		return nil, fmt.Errorf("random error")
	}

	secret := s.findSecret(name)
	if secret == nil {
		return nil, &errorsv1.StatusError{
			ErrStatus: metav1.Status{
				Reason: metav1.StatusReasonNotFound,
			},
		}
	}
	return secret, nil
}

func (s *secretGetterMock) Create(ctx context.Context, secret *apiv1.Secret, opts metav1.CreateOptions) (*apiv1.Secret, error) {
	if strings.Contains(secret.Name, "random-error-create") {
		return nil, fmt.Errorf("random error")
	}

	existingSecret := s.findSecret(secret.Name)
	if existingSecret != nil {
		return nil, &errorsv1.StatusError{
			ErrStatus: metav1.Status{
				Reason: metav1.StatusReasonConflict,
			},
		}
	}
	s.secrets = append(s.secrets, secret)
	return secret, nil
}

func (s *secretGetterMock) Patch(
	ctx context.Context,
	name string,
	pt types.PatchType,
	data []byte,
	opts metav1.PatchOptions,
	subresources ...string,
) (result *apiv1.Secret, err error) {
	if strings.Contains(name, "random-error-patch") {
		return nil, fmt.Errorf("random error")
	}

	var existingSecret = s.findSecret(name)

	if existingSecret == nil {
		return nil, &errorsv1.StatusError{
			ErrStatus: metav1.Status{
				Reason: metav1.StatusReasonNotFound,
			},
		}
	}

	var ops []map[string]string

	if err := json.Unmarshal(data, &ops); err != nil {
		return nil, err
	}

	for _, op := range ops {
		if op["path"] == "/data" {
			continue
		}
		key := strings.Split(op["path"], "/")[2]

		switch op["op"] {
		case "add":
			existingSecret.Data[key] = []byte(op["value"])
		default:
			delete(existingSecret.Data, key)
		}

	}

	return existingSecret, nil
}

func (s *secretGetterMock) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	if strings.Contains(name, "random-error-delete") {
		return fmt.Errorf("random error")
	}

	for i := range s.secrets {
		if s.secrets[i].Name == name {
			s.secrets = append(s.secrets[:i], s.secrets[i+1:]...)
			return nil
		}
	}
	return &errorsv1.StatusError{
		ErrStatus: metav1.Status{
			Reason: metav1.StatusReasonNotFound,
		},
	}
}
