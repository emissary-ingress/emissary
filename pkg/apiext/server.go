package apiext

import (
	"context"
	"os"
	"strings"

	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
	apiext "github.com/emissary-ingress/emissary/v3/pkg/apiext/internal"
	"github.com/emissary-ingress/emissary/v3/pkg/busy"
	"github.com/emissary-ingress/emissary/v3/pkg/k8s"
	"github.com/emissary-ingress/emissary/v3/pkg/logutil"
	"k8s.io/apimachinery/pkg/runtime"
)

// podNamespace determines the current Pods namespace
//
// Logic is borrowed from "k8s.io/client-go/tools/clientcmd".inClusterConfig.Namespace()
func podNamespace() string {
	// This way assumes you've set the POD_NAMESPACE environment variable using the downward API.
	// This check has to be done first for backwards compatibility with the way InClusterConfig was originally set up
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return "default"
}

// Webhook provides a simple abstraction for apiext webhook server
type WebhookRunner interface {
	Run(ctx context.Context, resourceScheme *runtime.Scheme) error
}

// WebhookServerConfig provides settings to configure the WebhookServer at runtime.
type WebhookServerConfig struct {
	Namespace   string
	ServiceName string
	HTTPPort    int
	HTTPSPort   int
}

type WebhookServer struct {
	namespace   string
	serviceName string
	httpPort    int
	httpsPort   int
}

func NewWebhookServer(config WebhookServerConfig) *WebhookServer {
	server := &WebhookServer{
		namespace:   config.Namespace,
		serviceName: config.ServiceName,
		httpPort:    config.HTTPPort,
		httpsPort:   config.HTTPSPort,
	}

	if server.namespace == "" {
		server.namespace = podNamespace()
	}

	if server.httpPort == 0 {
		server.httpPort = 8080
	}

	if server.httpsPort == 0 {
		server.httpsPort = 8443
	}

	if server.serviceName == "" {
		server.serviceName = "emissary-apiext"
	}

	return server
}

// Run the Emissary-ingress apiext conversion webhook using the provided configuration
func (s *WebhookServer) Run(ctx context.Context, scheme *runtime.Scheme) error {
	if lvl, err := logutil.ParseLogLevel(os.Getenv("APIEXT_LOGLEVEL")); err == nil {
		busy.SetLogLevel(lvl)
	}
	dlog.Infof(ctx, "APIEXT_LOGLEVEL=%v", busy.GetLogLevel())

	kubeinfo := k8s.NewKubeInfo("", "", "")
	restConfig, err := kubeinfo.GetRestConfig()
	if err != nil {
		return err
	}

	ca, caSecret, err := apiext.EnsureCA(ctx, restConfig, s.namespace)
	if err != nil {
		return err
	}

	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		EnableSignalHandling: true,
	})

	grp.Go("configure-crds", func(ctx context.Context) error {
		return apiext.ConfigureCRDs(ctx,
			restConfig,
			s.serviceName,
			s.namespace,
			caSecret,
			scheme)
	})

	grp.Go("serve-http", func(ctx context.Context) error {
		return apiext.ServeHTTP(ctx, s.httpPort)
	})

	grp.Go("serve-https", func(ctx context.Context) error {
		return apiext.ServeHTTPS(ctx, s.httpsPort, ca, scheme)
	})

	return grp.Wait()
}
