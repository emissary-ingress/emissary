package apiext

import (
	"context"
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
	k8sWatch "k8s.io/apimachinery/pkg/watch"
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
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
	}()
	apiExtClient, err := k8sClientAPIExtV1.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	crdsClient := apiExtClient.CustomResourceDefinitions()

	crdList, err := crdsClient.List(ctx, k8sTypesMetaV1.ListOptions{})
	if err != nil {
		return err
	}

	webhookPath := pathWebhooksCrdConvert // because pathWebhooksCrdConvert is a 'const' and you can't take the address of a const
	webhookPort := int32(443)
	conversionConfig := &k8sTypesAPIExtV1.CustomResourceConversion{
		Strategy: k8sTypesAPIExtV1.WebhookConverter,
		Webhook: &k8sTypesAPIExtV1.WebhookConversion{
			ClientConfig: &k8sTypesAPIExtV1.WebhookClientConfig{
				Service: &k8sTypesAPIExtV1.ServiceReference{
					Name:      serviceName,
					Namespace: serviceNamespace,
					Port:      &webhookPort,
					Path:      &webhookPath,
				},
				CABundle: caSecret.Data[k8sTypesCoreV1.TLSCertKey],
			},
			// Which versions of the conversion API our webhook supports.  Since we use
			// sigs.k8s.io/controller-runtime/pkg/webhook/conversion to implement the
			// webhook this list should be kept in-sync with what that package supports.
			ConversionReviewVersions: []string{
				"v1",
			},
		},
	}

	var errs derror.MultiError
	for _, crd := range crdList.Items {
		if err := updateCRD(ctx, scheme, crdsClient, crd, conversionConfig); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}

	// Watching for further changes is important so that re-applications of crds.yaml don't
	// break things.
	dlog.Infoln(ctx, "Initial configuration complete, now watching for further changes...")

	crdWatch, err := crdsClient.Watch(ctx, k8sTypesMetaV1.ListOptions{
		ResourceVersion: crdList.GetResourceVersion(),
	})
	if err != nil {
		return err
	}
	go func() { // Don't bother with dgroup because crdWatch.ResultChan() won't close until this goroutine returns.
		<-ctx.Done()
		crdWatch.Stop()
	}()
	for event := range crdWatch.ResultChan() {
		switch event.Type {
		case k8sWatch.Added, k8sWatch.Modified:
			crd := *(event.Object.(*k8sTypesAPIExtV1.CustomResourceDefinition))
			if err := updateCRD(ctx, scheme, crdsClient, crd, conversionConfig); err != nil {
				dlog.Errorln(ctx, err)
			}
		}
	}

	return nil
}

func updateCRD(
	ctx context.Context,
	scheme *k8sRuntime.Scheme,
	crdsClient k8sClientAPIExtV1.CustomResourceDefinitionInterface,
	crd k8sTypesAPIExtV1.CustomResourceDefinition,
	conversionConfig *k8sTypesAPIExtV1.CustomResourceConversion,
) error {
	if len(crd.Spec.Versions) < 2 {
		// Nothing to convert.
		dlog.Debugf(ctx, "Skipping %q because it only has one version", crd.ObjectMeta.Name)
		return nil
	}
	if !scheme.Recognizes(k8sSchema.GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: crd.Spec.Versions[0].Name,
		Kind:    crd.Spec.Names.Kind,
	}) {
		// Don't know how to convert.
		dlog.Debugf(ctx, "Skipping %q because it not a recognized type", crd.ObjectMeta.Name)
		return nil
	}
	if reflect.DeepEqual(crd.Spec.Conversion, conversionConfig) {
		// Already done.
		dlog.Infof(ctx, "Skipping %q because it is already configured", crd.ObjectMeta.Name)
		return nil
	}
	dlog.Infof(ctx, "Configuring conversion for %q", crd.ObjectMeta.Name)
	crd.Spec.Conversion = conversionConfig
	if _, err := crdsClient.Update(ctx, &crd, k8sTypesMetaV1.UpdateOptions{}); err != nil && !k8sErrors.IsConflict(err) {
		return err
	}

	if _, err := crdsClient.UpdateStatus(ctx, &crd, k8sTypesMetaV1.UpdateOptions{}); err != nil && !k8sErrors.IsConflict(err) {
		return err
	}

	return nil
}
