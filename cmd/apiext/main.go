package apiext

import (
	"context"
	"fmt"
	"os"
	"strings"

	k8sRuntime "k8s.io/apimachinery/pkg/runtime"

	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
	crdAll "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io"
	"github.com/emissary-ingress/emissary/v3/pkg/busy"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"github.com/emissary-ingress/emissary/v3/pkg/logutil"
)

// PodNamespace is borrowed from
// "k8s.io/client-go/tools/clientcmd".inClusterConfig.Namespace()
func PodNamespace() string {
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

// Main is a `github.com/emissary-ingress/emissary/v3/pkg/busy`-compatible wrapper around 'Run()', using
// values appropriate for the stock Emissary.
func Main(ctx context.Context, version string, args ...string) error {
	dlog.Infof(ctx, "Emissary Ingress apiext (version %q)", version)
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "%s: error: expected exactly one argument, got %d\n", os.Args[0], len(args))
		fmt.Fprintf(os.Stderr, "Usage: %s APIEXT_SVCNAME\n", os.Args[0])
		os.Exit(2)
	}

	scheme := crdAll.BuildScheme()

	return Run(ctx, PodNamespace(), args[0], 8080, 8443, scheme)
}

// Run runs the Emissary apiext server process, but takes enough arguments that you should be able
// to reuse it to implement your own apiext server.
func Run(ctx context.Context, namespace, svcname string, httpPort, httpsPort int, scheme *k8sRuntime.Scheme) error {
	if lvl, err := logutil.ParseLogLevel(os.Getenv("APIEXT_LOGLEVEL")); err == nil {
		busy.SetLogLevel(lvl)
	}
	dlog.Infof(ctx, "APIEXT_LOGLEVEL=%v", busy.GetLogLevel())

	kubeConfig := kates.NewConfigFlags(false)
	restConfig, err := kubeConfig.ToRESTConfig()
	if err != nil {
		return err
	}

	ca, caSecret, err := EnsureCA(ctx, restConfig, namespace)
	if err != nil {
		return err
	}

	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		EnableSignalHandling: true,
	})

	grp.Go("configure-crds", func(ctx context.Context) error {
		return ConfigureCRDs(ctx,
			restConfig,
			svcname,
			namespace,
			caSecret,
			scheme)
	})

	grp.Go("serve-http", func(ctx context.Context) error {
		return ServeHTTP(ctx, httpPort)
	})

	grp.Go("serve-https", func(ctx context.Context) error {
		return ServeHTTPS(ctx, httpsPort, ca, scheme)
	})

	return grp.Wait()
}
