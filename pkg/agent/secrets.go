package agent

import (
	"context"
	"encoding/json"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type secretSyncAction string

const (
	secretSyncActionSet    = secretSyncAction("SET")
	secretSyncActionDelete = secretSyncAction("DELETE")
)

type SecretInterface interface {
	Create(ctx context.Context, secret *apiv1.Secret, opts metav1.CreateOptions) (*apiv1.Secret, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*apiv1.Secret, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *apiv1.Secret, err error)
}

// secretsGetterFactory is a factory for creating SecretsGetter.
type secretsGetterFactory func(namespace string) (SecretInterface, error)

type secretSyncCommand struct {
	name      string
	namespace string
	action    secretSyncAction
	secret    map[string][]byte
}

func (s *secretSyncCommand) String() string {
	return fmt.Sprintf("<secret=%s namespace=%s action=%s>", s.name, s.namespace, s.action)
}

func (s *secretSyncCommand) RunWithClientFactory(
	ctx context.Context, secretGetterFactory secretsGetterFactory,
) error {
	client, err := secretGetterFactory(s.namespace)
	if err != nil {
		return err
	}

	return s.syncSecret(ctx, client)
}

func (s *secretSyncCommand) getOps(insertRoot bool) (ops []map[string]string) {
	// if the secret is empty, this is required.
	if insertRoot {
		ops = append(ops, map[string]string{
			"op":    "add",
			"path":  "/data",
			"value": "{}",
		})
	}

	switch s.action {
	case secretSyncActionDelete:
		for key := range s.secret {
			ops = append(ops, map[string]string{
				"op":   "remove",
				"path": fmt.Sprintf("/data/%s", key),
			})
		}
	case secretSyncActionSet:
		for key, value := range s.secret {
			ops = append(ops, map[string]string{
				"op":    "add",
				"path":  fmt.Sprintf("/data/%s", key),
				"value": string(value),
			})
		}
	default:
		panic(fmt.Sprintf("action %s is not supported by the secret sync directive", s.action))
	}
	return ops
}

func (s *secretSyncCommand) syncSecret(ctx context.Context, client SecretInterface) error {
	if s.secret == nil && s.action != secretSyncActionDelete {
		return nil
	}

	var (
		secret *apiv1.Secret
		err    error
	)

	secret, err = client.Get(ctx, s.name, metav1.GetOptions{})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			if s.action == secretSyncActionSet {
				secret, err = client.Create(ctx, &apiv1.Secret{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      s.name,
						Namespace: s.namespace,
					},
					Data: s.secret,
					Type: apiv1.SecretTypeOpaque,
				}, metav1.CreateOptions{})

				if err != nil {
					return fmt.Errorf("failed to create the secret: %w", err)
				}
			}

			return nil
		}
		return fmt.Errorf("failed to get the secret %s: %w", s.name, err)
	}

	opsJSON, err := json.Marshal(s.getOps(len(secret.Data) == 0))

	if err != nil {
		return fmt.Errorf("failed to generate patch ops: %w", err)
	}

	secret, err = client.Patch(ctx, s.name, types.JSONPatchType, opsJSON, metav1.PatchOptions{})

	if err != nil {
		return fmt.Errorf("failed to update the secret: %w", err)
	}

	// if no keys left, we should delete the secret.
	if len(secret.Data) == 0 {
		err := client.Delete(ctx, s.name, metav1.DeleteOptions{})

		if err != nil {
			return fmt.Errorf("failed to clean up the secret %s: %w", s.name, err)
		}
	}

	return nil
}

// NewSecretsGetter instantiates a client to interact with the Kubernetes secret API.
func NewSecretsGetter(namespace string) (SecretInterface, error) {
	kubeConfig, err := newK8sRestClient()
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		panic(err.Error())
	}

	return clientSet.CoreV1().Secrets(namespace), nil
}
