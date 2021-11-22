package apiext

import (
	"bytes"
	"context"
	"fmt"

	// k8s types
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesAPIExtV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// k8s clients
	k8sClientAPIExtV1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"

	// k8s utils
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	k8sSchema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	"github.com/datawire/dlib/derror"
)

// ConfigureCRDs uses 'restConfig' to look at all CustomResourceDefinitions that are mentioned in
// 'scheme', and adjusts each of their .spec.conversion.webhook.clientConfig.caBundle to match the
// "tls.crt" field in 'caSecret'.
func ConfigureCRDs(
	ctx context.Context,
	restConfig *rest.Config,
	caSecret *k8sTypesCoreV1.Secret,
	scheme *k8sRuntime.Scheme,
) error {
	apiExtClient, err := k8sClientAPIExtV1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	crdsClient := apiExtClient.CustomResourceDefinitions()

	crds, err := crdsClient.List(ctx, k8sTypesMetaV1.ListOptions{})
	if err != nil {
		return err
	}

	var count int
	var errs derror.MultiError
	for _, crd := range crds.Items {
		// Versions is a mandatory field we can rely on to have at least 1 value.
		// Regardless, we protect against len=0 in the conditional
		if len(crd.Spec.Versions) < 1 || !scheme.Recognizes(k8sSchema.GroupVersionKind{
			Group:   crd.Spec.Group,
			Version: crd.Spec.Versions[0].Name,
			Kind:    crd.Spec.Names.Kind,
		}) {
			continue
		}

		count++
		if crd.Spec.Conversion == nil {
			crd.Spec.Conversion = &k8sTypesAPIExtV1.CustomResourceConversion{}
		}
		if crd.Spec.Conversion.Webhook == nil {
			crd.Spec.Conversion.Webhook = &k8sTypesAPIExtV1.WebhookConversion{}
		}
		if crd.Spec.Conversion.Webhook.ClientConfig == nil {
			crd.Spec.Conversion.Webhook.ClientConfig = &k8sTypesAPIExtV1.WebhookClientConfig{}
		}
		if bytes.Equal(crd.Spec.Conversion.Webhook.ClientConfig.CABundle, caSecret.Data[k8sTypesCoreV1.TLSCertKey]) {
			continue
		}
		_, err := crdsClient.Update(ctx, &crd, k8sTypesMetaV1.UpdateOptions{})
		if err != nil && !k8sErrors.IsConflict(err) {
			errs = append(errs, err)
		}
	}
	if count == 0 {
		return fmt.Errorf("found no CRD types to add webhooks to")
	}
	if len(errs) > 0 {
		return errs
	}

	return nil
}
