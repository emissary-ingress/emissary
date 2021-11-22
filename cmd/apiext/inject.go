package apiext

import (
	"context"
	"fmt"
	"reflect"

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
	"github.com/datawire/dlib/dlog"
)

// ConfigureCRDs uses 'restConfig' to look at all CustomResourceDefinitions that are mentioned in
// 'scheme', and adjusts each of their .spec.conversion.webhook.clientConfig.caBundle to match the
// "tls.crt" field in 'caSecret'.
func ConfigureCRDs(
	ctx context.Context,
	restConfig *rest.Config,
	serviceName, serviceNamespace string,
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

	webhookPath := pathWebhooksCrdConvert // because pathWebhooksCrdConvert is a 'const' and you can't take the address of a const
	conversionConfig := &k8sTypesAPIExtV1.CustomResourceConversion{
		Strategy: k8sTypesAPIExtV1.WebhookConverter,
		Webhook: &k8sTypesAPIExtV1.WebhookConversion{
			ClientConfig: &k8sTypesAPIExtV1.WebhookClientConfig{
				Service: &k8sTypesAPIExtV1.ServiceReference{
					Name:      serviceName,
					Namespace: serviceNamespace,
					Path:      &webhookPath,
				},
				CABundle: caSecret.Data[k8sTypesCoreV1.TLSCertKey],
			},
			// Which versions of the conversion API our webhook supports.  Since we use
			// sigs.k8s.io/controller-runtime/pkg/webhook/conversion to implement the
			// webhook this list should be kept in-sync with what that package supports.
			ConversionReviewVersions: []string{
				"v1beta1",
			},
		},
	}

	var count int
	var errs derror.MultiError
	for _, crd := range crds.Items {
		if len(crd.Spec.Versions) < 2 {
			// Nothing to convert.
			dlog.Debugf(ctx, "Skipping %q because it only has one version", crd.ObjectMeta.Name)
			continue
		}
		if !scheme.Recognizes(k8sSchema.GroupVersionKind{
			Group:   crd.Spec.Group,
			Version: crd.Spec.Versions[0].Name,
			Kind:    crd.Spec.Names.Kind,
		}) {
			// Don't know how to convert.
			dlog.Debugf(ctx, "Skipping %q because it not a recognized type", crd.ObjectMeta.Name)
			continue
		}
		count++
		if reflect.DeepEqual(crd.Spec.Conversion, conversionConfig) {
			// Already done.
			dlog.Infof(ctx, "Skipping %q because it is already configured", crd.ObjectMeta.Name)
			continue
		}
		dlog.Infof(ctx, "Configuring conversion for %q", crd.ObjectMeta.Name)
		crd.Spec.Conversion = conversionConfig
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
