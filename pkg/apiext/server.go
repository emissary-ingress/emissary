package apiext

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-logr/zapr"
	"golang.org/x/sync/errgroup"

	"github.com/emissary-ingress/emissary/v3/pkg/apiext/defaults"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/ca"
	cacertcontroller "github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/controller/cacert"
	crdcontroller "github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/controller/crd"
	cacertrunnable "github.com/emissary-ingress/emissary/v3/pkg/apiext/internal/runnable/cacert"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext/path"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"
)

const (
	leaderElectionID = "emissary-ca-mgr-leader"
)

// Webhook provides a simple abstraction for apiext webhook server
type WebhookRunner interface {
	Run(ctx context.Context, resourceScheme *runtime.Scheme) error
}

type WebhookServer struct {
	logger               *zap.Logger
	certificateAuthority ca.CertificateAuthority
	k8sClient            client.Reader
	namespace            string
	serviceSettings      types.NamespacedName
	caSecretSettings     types.NamespacedName
	httpPort             int
	httpsPort            int

	caMgmtEnabled       bool
	crdPatchMgmtEnabled bool
}

func NewWebhookServer(logger *zap.Logger, serviceName string, options ...WebhookOption) *WebhookServer {
	server := &WebhookServer{
		logger:               logger,
		certificateAuthority: ca.NewAPIExtCertificateAuthority(logger),
		namespace:            podNamespace(),
		httpPort:             8080,
		httpsPort:            8443,
		caMgmtEnabled:        true,
		crdPatchMgmtEnabled:  true,
	}

	for _, optFn := range options {
		optFn(server)
	}

	server.caSecretSettings = types.NamespacedName{
		Namespace: server.namespace,
		Name:      defaults.WebhookCASecretName,
	}

	server.serviceSettings = types.NamespacedName{
		Namespace: server.namespace,
		Name:      serviceName,
	}

	return server
}

// Run the Emissary-ingress apiext conversion webhook using the provided configuration
func (s *WebhookServer) Run(ctx context.Context, scheme *runtime.Scheme) error {
	if err := corev1.AddToScheme(scheme); err != nil {
		return err
	}

	if err := apiextv1.AddToScheme(scheme); err != nil {
		return err
	}

	zaprLogger := zapr.NewLoggerWithOptions(s.logger)
	ctrl.SetLogger(zaprLogger)
	klog.SetLogger(zaprLogger)

	k8sConfig, err := config.GetConfig()
	if err != nil {
		return err
	}

	leaderElectionEnabled := s.isLeaderElectionEnabled()
	s.logger.Info("leader election support", zap.Bool("enabled", leaderElectionEnabled))

	mgr, err := manager.New(k8sConfig, manager.Options{
		Scheme:                        scheme,
		LeaderElection:                leaderElectionEnabled,
		LeaderElectionID:              leaderElectionID,
		LeaderElectionNamespace:       s.namespace,
		LeaderElectionReleaseOnCancel: true,
		Metrics:                       server.Options{BindAddress: "0"},
		Cache:                         createCacheOptions(s.namespace),
	})
	if err != nil {
		return err
	}

	s.k8sClient = mgr.GetClient()

	caCertController := cacertcontroller.NewCACertController(
		mgr.GetClient(),
		s.logger,
		s.certificateAuthority,
		cacertcontroller.WithCASecretSettings(s.caSecretSettings),
	)
	if err := caCertController.SetupWithManager(mgr); err != nil {
		return err
	}

	if s.caMgmtEnabled {
		crdCAController := crdcontroller.NewCRDPatchController(mgr.GetClient(), s.logger,
			s.certificateAuthority,
			s.serviceSettings,
			s.caSecretSettings,
		)
		if err := crdCAController.SetupWithManager(mgr); err != nil {
			return err
		}
	}

	if s.crdPatchMgmtEnabled {
		caCertMgr := cacertrunnable.NewCACertManager(s.logger, mgr.GetClient())
		if err := mgr.Add(caCertMgr); err != nil {
			return err
		}
	}

	grp, gctx := errgroup.WithContext(ctx)

	grp.Go(func() error {
		return mgr.Start(gctx)
	})

	// we will wait until we have successfully obtained a CA root certificate
	// before we start the web servers, to ensure we don't become ready too early
	runImmediately := true
	pollInterval := 1 * time.Second
	if err := wait.PollUntilContextCancel(gctx, pollInterval, runImmediately, s.ready); err != nil {
		return fmt.Errorf("apiext server unable to obtain a root ca during startup")
	}

	grp.Go(func() error {
		return s.serveHTTPS(gctx, scheme)
	})

	grp.Go(func() error {
		return s.serveHealthz(gctx)
	})

	return grp.Wait()
}

func (s *WebhookServer) ready(_ context.Context) (done bool, err error) {
	return s.certificateAuthority.Ready(), nil
}

// serveHTTPS starts listening for incoming https request and handles ConversionWebhookRequuests.
func (s *WebhookServer) serveHTTPS(ctx context.Context, scheme *runtime.Scheme) error {
	errChan := make(chan error)

	mux := http.NewServeMux()
	mux.Handle(path.WebhooksCrdConvert, conversion.NewWebhookHandler(scheme))

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", s.httpsPort),
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion:     tls.VersionTLS13,
			GetCertificate: s.certificateAuthority.GetCertificate,
		},
	}

	go func() {
		s.logger.Info("starting conversion webhook server", zap.Int("port", s.httpsPort))
		if err := server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	defer server.Close()

	// block waiting for graceful shutdown or server error
	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	case err := <-errChan:
		return err
	}
}

// serveHealthz starts http server listening for http healthz (ready,liviness)
func (s *WebhookServer) serveHealthz(ctx context.Context) error {
	errChan := make(chan error)
	mux := http.NewServeMux()

	mux.Handle(path.ProbesReady, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.certificateAuthority.Ready() && s.areCRDsReady(r.Context()) {
			_, _ = io.WriteString(w, "Ready!\n")
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))

	mux.Handle(path.ProbesLive, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "Living!\n")
	}))

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", s.httpPort),
		Handler: mux,
	}

	go func() {
		s.logger.Info("starting healthz server", zap.Int("port", s.httpPort))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// block waiting for graceful shutdown or server error
	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	case err := <-errChan:
		return err
	}
}

func (s *WebhookServer) isLeaderElectionEnabled() bool {
	return s.caMgmtEnabled || s.crdPatchMgmtEnabled
}

func (s *WebhookServer) areCRDsReady(ctx context.Context) bool {
	caCert := s.certificateAuthority.GetCACert()
	if caCert == nil {
		return false
	}

	crdList := &apiextv1.CustomResourceDefinitionList{}
	options := []client.ListOption{
		client.MatchingLabels{"app.kubernetes.io/part-of": "emissary-apiext"},
	}

	err := s.k8sClient.List(ctx, crdList, options...)
	if err != nil {
		s.logger.Error("ready check unable to list getambassadorio crds", zap.Error(err))
		return false
	}

	for _, item := range crdList.Items {
		if len(item.Spec.Versions) < 2 {
			continue
		}

		if item.Spec.Conversion == nil || item.Spec.Conversion.Webhook == nil || item.Spec.Conversion.Webhook.ClientConfig == nil {
			return false
		}

		if !bytes.Equal(item.Spec.Conversion.Webhook.ClientConfig.CABundle, caCert.CertificatePEM) {
			return false
		}
	}

	return true
}

func createCacheOptions(secretNamespace string) cache.Options {
	return cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&apiextv1.CustomResourceDefinition{}: {
				Label: labels.SelectorFromSet(labels.Set{"app.kubernetes.io/part-of": "emissary-apiext"}),
			},
			&corev1.Secret{}: {
				Namespaces: map[string]cache.Config{
					secretNamespace: {},
				},
			},
		},
	}
}
