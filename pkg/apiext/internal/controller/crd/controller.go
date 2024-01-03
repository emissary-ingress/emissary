package crd

import (
	"context"
	"fmt"
	"reflect"

	"github.com/emissary-ingress/emissary/v3/pkg/apiext/defaults"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/ca"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/controller/predicateutils"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext/path"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	crdGroupIndexField      = ".spec.group"
	getambassadorioCRDGroup = "getambassador.io"
)

// crdPatchController will watch for the getambassador.io CRDs and ensure the conversion webhook CA is
// injected correctly
type crdPatchController struct {
	client client.Client
	logger *zap.Logger

	serviceSettings     types.NamespacedName
	caSecretSettings    types.NamespacedName
	certifcateAuthority ca.CertificateAuthority
}

func NewCRDPatchController(client client.Client, logger *zap.Logger, certifcateAuthority ca.CertificateAuthority,
	serviceSettings types.NamespacedName, caSecretSettings types.NamespacedName) *crdPatchController {
	return &crdPatchController{
		client:              client,
		logger:              logger.Named("crd-patch-controller"),
		serviceSettings:     serviceSettings,
		caSecretSettings:    caSecretSettings,
		certifcateAuthority: certifcateAuthority,
	}
}

// SetupWithManager will register controller with manager
func (c *crdPatchController) SetupWithManager(mgr manager.Manager) error {

	if err := registerGetAmbassadorioGroupIndexer(mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&apiextv1.CustomResourceDefinition{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(getAmbassadorioPredicate()),
			),
		).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(enqueueCRDForCASecretChanges(mgr.GetClient(), c.logger)),
			builder.WithPredicates(predicate.NewPredicateFuncs(predicateutils.CASecretPredicate(c.caSecretSettings))),
		).
		Complete(c)
}

// Reconcile implements reconcile.Reconciler.
func (c *crdPatchController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	c.logger.Info("CustomResourceDefinition reconcile triggered by object", zap.String("namespace", request.Namespace), zap.String("name", request.Name))

	crDefKey := types.NamespacedName{
		Name:      request.Name,
		Namespace: request.Namespace,
	}

	crDef := &apiextv1.CustomResourceDefinition{}

	if err := c.client.Get(ctx, crDefKey, crDef); err != nil {
		c.logger.Error("error getting CustomResourceDefition",
			zap.String("name", request.Name),
			zap.String("namespace", request.Namespace),
			zap.Error(err),
		)
		return reconcile.Result{}, nil
	}

	if !crDef.ObjectMeta.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	if len(crDef.Spec.Versions) <= 1 {
		c.logger.Info("skipping reconciliation, CustomResourceDefinition only has one version",
			zap.String("name", crDef.Name),
		)
		return reconcile.Result{}, nil
	}

	caCert := c.certifcateAuthority.GetCACert()
	if caCert == nil {
		return reconcile.Result{RequeueAfter: defaults.RequeueAfter}, nil
	}

	if err := c.reconcileCRD(ctx, crDef, caCert); err != nil {
		if k8serrors.IsUnauthorized(err) {
			c.logger.Info("unable to update custom resource definition due to missing permissions, not requeuing reconcile",
				zap.String("name", crDef.Name),
			)
		}
		return reconcile.Result{RequeueAfter: defaults.RequeueAfter}, fmt.Errorf("error reconciling CRD, requeuing event")
	}

	return reconcile.Result{}, nil
}

func (c *crdPatchController) reconcileCRD(ctx context.Context, crDef *apiextv1.CustomResourceDefinition, cert *ca.CACert) error {
	logger := c.logger.With(zap.String("name", crDef.Name))

	conversionConfig := createConversionConfig(cert, c.serviceSettings)
	if reflect.DeepEqual(crDef.Spec.Conversion, conversionConfig) {
		logger.Info("already configured, skipping reconciliation")
		return nil
	}

	crDef.Spec.Conversion = conversionConfig

	logger.Info("patching CustomResourceDefinition with new conversion webhook CABundle")

	if err := c.client.Update(ctx, crDef); err != nil && !k8serrors.IsConflict(err) {
		logger.Error("unable to update CustomResourceDefinition", zap.Error(err))
		return err
	}

	if err := c.client.Status().Update(ctx, crDef); err != nil && !k8serrors.IsConflict(err) {
		logger.Error("unable to update CRD status", zap.Error(err))
		return err
	}

	return nil
}

func createConversionConfig(cert *ca.CACert, serviceSettings types.NamespacedName) *apiextv1.CustomResourceConversion {
	webhookPath := path.WebhooksCrdConvert
	webhookPort := int32(443)

	conversionConfig := &apiextv1.CustomResourceConversion{
		Strategy: apiextv1.WebhookConverter,
		Webhook: &apiextv1.WebhookConversion{
			ClientConfig: &apiextv1.WebhookClientConfig{
				Service: &apiextv1.ServiceReference{
					Name:      serviceSettings.Name,
					Namespace: serviceSettings.Namespace,
					Port:      &webhookPort,
					Path:      &webhookPath,
				},
				CABundle: cert.CertificatePEM,
			},
			ConversionReviewVersions: []string{"v1"},
		},
	}

	return conversionConfig
}

func enqueueCRDForCASecretChanges(k8sclient client.Reader, logger *zap.Logger) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		logger := logger.With(
			zap.String("objName", obj.GetName()),
			zap.String("objNamespace", obj.GetNamespace()),
		)
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			logger.Error("obj not a valid Secret")
			return nil
		}
		if !secret.ObjectMeta.DeletionTimestamp.IsZero() {
			// ignore deletes, we only care to requeue if create/update
			return nil
		}

		crdList := &apiextv1.CustomResourceDefinitionList{}
		if err := k8sclient.List(ctx, crdList); err != nil {
			logger.Error("unable to get CRDs from cluster", zap.Error(err))
			return nil
		}

		requests := make([]reconcile.Request, 0, len(crdList.Items))
		for _, crDef := range crdList.Items {
			if crDef.Spec.Group != getambassadorioCRDGroup {
				continue
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: crDef.Namespace,
					Name:      crDef.Name,
				},
			})
		}

		logger.Info("enqueued multipie request items", zap.Int("items", len(requests)))

		return requests
	}
}
