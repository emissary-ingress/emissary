package cacert

import (
	"context"
	"fmt"
	"time"

	"github.com/emissary-ingress/emissary/v3/pkg/apiext/certutils"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext/defaults"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	defaultCACertValidDuration    = 365 * 24 * time.Hour
	defaultValidityTickerDuration = 10 * time.Second
)

// CACertManager ensures that the CA Cert is created and valid in the cluster
type CACertManager struct {
	logger *zap.Logger
	client client.Client

	secretNamespace string
	secretName      string

	validityTickerDuration time.Duration
	certValidityDuration   time.Duration
}

var _ manager.Runnable = (*CACertManager)(nil)
var _ manager.LeaderElectionRunnable = (*CACertManager)(nil)

// NewCACertManager will initialize a new instance of the CACertManager.
func NewCACertManager(logger *zap.Logger, client client.Client, options ...Option) *CACertManager {
	manager := &CACertManager{
		client:                 client,
		logger:                 logger.Named("ca-cert-mgr"),
		secretNamespace:        defaults.APIExtNamespace,
		secretName:             defaults.WebhookCASecretName,
		validityTickerDuration: defaultValidityTickerDuration,
		certValidityDuration:   defaultCACertValidDuration,
	}

	for _, optFn := range options {
		optFn(manager)
	}

	return manager
}

// NeedLeaderElection implements manager.LeaderElectionRunnable.
func (*CACertManager) NeedLeaderElection() bool {
	return true
}

// Start implements manager.Runnable so that it will start watching for a valid CA Cert when the manager starts.
func (cm *CACertManager) Start(ctx context.Context) error {
	cm.logger.Info("starting the CACertManager to ensure that the CA Cert is available and valid")
	if err := cm.ensureCA(ctx); err != nil {
		return err
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		ticker := time.NewTicker(cm.validityTickerDuration)
		defer ticker.Stop()
	checker:
		for {
			select {
			case <-gctx.Done():
				break checker
			case <-ticker.C:
				if err := cm.ensureCA(ctx); err != nil {
					cm.logger.Error("unable to ensure CA cert is valid", zap.Error(err))
				}
			}
		}
		cm.logger.Info("CACertManager shutting down due to shutdown signal")
		return nil
	})

	return g.Wait()
}

// ensureCA  ensures that a CA Cert is valid and available for generating Server certs
func (cm *CACertManager) ensureCA(ctx context.Context) error {
	secretKey := types.NamespacedName{
		Name:      cm.secretName,
		Namespace: cm.secretNamespace,
	}
	secret := &corev1.Secret{}

	err := cm.client.Get(ctx, secretKey, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			cm.logger.Info("ca secret not found", zap.String("secretName", cm.fullSecretName()))

			return cm.generateCACert(ctx, nil)
		}
		cm.logger.Error("error getting ca secret from cluster",
			zap.String("secretName", cm.fullSecretName()),
			zap.Error(err),
		)
		return err
	}

	if cm.isCASecretValid(secret) {
		return nil
	}

	return cm.generateCACert(ctx, secret.DeepCopy())
}

func (cm *CACertManager) generateCACert(ctx context.Context, secret *corev1.Secret) error {
	cm.logger.Info("generating new root ca certificate", zap.String("secretName", cm.fullSecretName()))

	privateKey, cert, err := certutils.GenerateRootCACert(defaults.SubjectOrganization, defaultCACertValidDuration)
	if err != nil {
		return err
	}

	secretData := map[string][]byte{
		corev1.TLSPrivateKeyKey: privateKey,
		corev1.TLSCertKey:       cert,
	}

	if secret == nil {
		return cm.client.Create(ctx,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cm.secretName,
					Namespace: cm.secretNamespace,
				},
				Type: corev1.SecretTypeTLS,
				Data: secretData,
			}, &client.CreateOptions{})

	}

	secret.Data = secretData
	return cm.client.Update(ctx, secret, &client.UpdateOptions{})
}

// isCASecretInvalid determines if the CA cert within the secret is invalid. This will indicate
// whether a new CA Cert needs to be generated.
func (cm *CACertManager) isCASecretValid(secret *corev1.Secret) bool {
	_, cert, err := certutils.ParseCASecret(secret)
	if err != nil {
		cm.logger.Error("unable to parse ca Secret",
			zap.String("secretName", cm.fullSecretName()),
			zap.Error(err))

		return false
	}

	// automatically force re-generating when 14 days until expires to give plenty of leeway
	notAfterLeeway := cert.NotAfter.Add(-(14 * 24 * time.Hour))
	if time.Now().After(notAfterLeeway) {
		cm.logger.Info("root x509 ca certificate is expiring soon",
			zap.String("secretName", cm.fullSecretName()),
			zap.Time("checkTime", time.Now()),
			zap.Time("notAfter", cert.NotAfter),
			zap.Time("notAfterLeeway", notAfterLeeway),
		)
		return false
	}

	cm.logger.Debug("root x509 ca certificate is valid",
		zap.String("secretName", cm.fullSecretName()),
		zap.Time("expirtsAt", cert.NotAfter),
	)

	return true
}

// fullSecretName returns the fully qualified namespace/name for the ca secret to simplify logging
func (ca *CACertManager) fullSecretName() string {
	return fmt.Sprintf("%s/%s", ca.secretNamespace, ca.secretName)
}
