package predicateutils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CASecretPredicate(secretSettings types.NamespacedName) func(client.Object) bool {
	return func(obj client.Object) bool {
		secret, ok := obj.(*corev1.Secret)
		if !ok || secret == nil {
			return false
		}

		if secret.Name == secretSettings.Name && secret.Namespace == secretSettings.Namespace {
			return true
		}

		return false
	}
}
