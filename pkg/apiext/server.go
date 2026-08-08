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
	webhookHostname      string
	httpPort             int
	httpsPort            int

	caMgmtEnabled       bool
	crdPatchMgmtEnabled bool
	crdLabelSelectors   map[string]string
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
		crdLabelSelectors:    map[string]string{},
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

	// This is the hostname that the Kubernetes apiserver will use as the SNI ServerName
	// when it dials us through our Service to deliver a CRD conversion request; it is
	// derived entirely from well-known values (our Service's name/namespace), so we don't
	// need to wait for an incoming request to know what server certificate to generate.
	server.webhookHostname = fmt.Sprintf("%s.%s.svc", server.serviceSettings.Name, server.serviceSettings.Namespace)

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

	s.logger.Info("CA management", zap.Bool("enabled", s.caMgmtEnabled))
	s.logger.Info("CRD patch management", zap.Bool("enabled", s.crdPatchMgmtEnabled))

	mgr, err := manager.New(k8sConfig, manager.Options{
		Scheme:                        scheme,
		LeaderElection:                leaderElectionEnabled,
		LeaderElectionID:              leaderElectionID,
		LeaderElectionNamespace:       s.namespace,
		LeaderElectionReleaseOnCancel: true,
		Metrics:                       server.Options{BindAddress: "0"},
		Cache:                         s.buildCacheOptions(),
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

	if s.crdPatchMgmtEnabled {
		crdCAController := crdcontroller.NewCRDPatchController(mgr.GetClient(), s.logger,
			s.certificateAuthority,
			s.serviceSettings,
			s.caSecretSettings,
		)
		if err := crdCAController.SetupWithManager(mgr); err != nil {
			return err
		}
	}

	if s.caMgmtEnabled {
		caCertMgr := cacertrunnable.NewCACertManager(s.logger, mgr.GetClient(),
			cacertrunnable.WithCASecretNamespace(s.caSecretSettings.Namespace),
			cacertrunnable.WithCASecretName(s.caSecretSettings.Name),
		)
		if err := mgr.Add(caCertMgr); err != nil {
			return err
		}
	}

	grp, gctx := errgroup.WithContext(ctx)

	grp.Go(func() error {
		return mgr.Start(gctx)
	})

	// We will wait until we have successfully obtained a CA root certificate, and used it
	// to eagerly generate our webhook server certificate, before we start the web servers.
	// This ensures we don't become ready too early: if we waited to lazily generate the
	// server certificate on the first incoming TLS handshake, that handshake would be
	// competing with certificate generation for time against the apiserver's webhook call
	// timeout, and could time out under load.
	runImmediately := true
	pollInterval := 1 * time.Second
	if err := wait.PollUntilContextCancel(gctx, pollInterval, runImmediately, s.ready); err != nil {
		return fmt.Errorf("apiext server unable to prepare a root ca and server certificate during startup")
	}

	grp.Go(func() error {
		return s.serveHTTPS(gctx, scheme)
	})

	grp.Go(func() error {
		return s.serveHealthz(gctx)
	})

	return grp.Wait()
}

// ready reports whether the webhook server is prepared to start serving traffic: we need a
// CA root certificate, and we need to have eagerly generated (and cached) the server
// certificate for our well-known webhook hostname, so that the apiserver's first TLS
// handshake with us doesn't pay the cost of certificate generation.
func (s *WebhookServer) ready(_ context.Context) (done bool, err error) {
	if !s.certificateAuthority.Ready() {
		return false, nil
	}

	if _, err := s.certificateAuthority.GetCertificate(&tls.ClientHelloInfo{ServerName: s.webhookHostname}); err != nil {
		s.logger.Error("unable to eagerly generate webhook server certificate",
			zap.String("serverName", s.webhookHostname), zap.Error(err))
		return false, nil
	}

	return true, nil
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
	err := s.k8sClient.List(ctx, crdList)
	if err != nil {
		s.logger.Error("ready check unable to list getambassadorio crds", zap.Error(err))
		return false
	}

	for _, item := range crdList.Items {
		if item.Spec.Group != "getambassador.io" || len(item.Spec.Versions) < 2 {
			continue
		}

		if item.Spec.Conversion == nil {
			return false
		}

		if item.Spec.Conversion.Strategy == apiextv1.NoneConverter {
			continue
		}

		if item.Spec.Conversion.Webhook == nil || item.Spec.Conversion.Webhook.ClientConfig == nil {
			return false
		}

		if !bytes.Equal(item.Spec.Conversion.Webhook.ClientConfig.CABundle, caCert.CertificatePEM) {
			return false
		}
	}

	return true
}

func (s *WebhookServer) buildCacheOptions() cache.Options {
	crdLabelSectors := labels.SelectorFromSet(s.crdLabelSelectors)

	return cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&apiextv1.CustomResourceDefinition{}: {
				Label: crdLabelSectors,
			},
			&corev1.Secret{}: {
				Namespaces: map[string]cache.Config{
					s.namespace: {},
				},
			},
		},
	}
}
