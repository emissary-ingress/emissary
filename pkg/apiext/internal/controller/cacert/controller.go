package cacert

import (
	"context"

	"github.com/emissary-ingress/emissary/v3/pkg/apiext/defaults"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/ca"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/controller/predicateutils"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var needLeaderElection = false

// caCertController watches for changes to the Secret that contains the RootCA and ensures the CertificateAuthority
// has the latest CA Cert
type caCertController struct {
	client client.Client
	logger *zap.Logger

	secretSettings types.NamespacedName

	certificateAuthority ca.CertificateAuthority
}

func NewCACertController(client client.Client, logger *zap.Logger, certificateAuthority ca.CertificateAuthority, options ...Option) *caCertController {
	caController := &caCertController{
		client:               client,
		logger:               logger.Named("ca-cert-controller"),
		certificateAuthority: certificateAuthority,
		secretSettings: types.NamespacedName{
			Namespace: defaults.WebhookCASecretNamespace,
			Name:      defaults.WebhookCASecretName,
		},
	}

	for _, optFn := range options {
		optFn(caController)
	}

	return caController
}

// SetupWithManager will register indexes, watches and registers the caCertController with the manager
func (c *caCertController) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			NeedLeaderElection: &needLeaderElection,
		}).
		For(&v1.Secret{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(predicateutils.CASecretPredicate(c.secretSettings)),
			),
		).
		Complete(c)
}

// Reconcile implements reconcile.Reconciler to watch for CA Cert Secret changes and to update the CertificateAuthority
func (c *caCertController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	c.logger.Info("Secret reconcile triggered by object",
		zap.String("namespace", request.Namespace),
		zap.String("name", request.Name),
	)

	secretKey := types.NamespacedName{
		Name:      request.Name,
		Namespace: request.Namespace,
	}
	secret := &corev1.Secret{}
	if err := c.client.Get(ctx, secretKey, secret); err != nil {
		c.logger.Error("unable to load secret, skipping reconcile", zap.String("secret", secretKey.String()), zap.Error(err))
		c.certificateAuthority.SetCACert(nil)
		return reconcile.Result{}, nil
	}

	if !secret.ObjectMeta.DeletionTimestamp.IsZero() {
		c.logger.Info("secret is already marked for deletion, skipping reconcile", zap.String("secret", secretKey.String()))
		c.certificateAuthority.SetCACert(nil)
		return reconcile.Result{}, nil
	}

	caCert, err := ca.CACertFromSecret(secret)
	if err != nil {
		c.logger.Error("unable to obtain a valid root CA cert from the secret", zap.Error(err))
		c.certificateAuthority.SetCACert(nil)
		return reconcile.Result{}, nil
	}

	c.logger.Info("CA cert being set", zap.Any("caCert", caCert))

	c.certificateAuthority.SetCACert(&caCert)
	return reconcile.Result{}, nil
}
