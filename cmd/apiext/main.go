package apiext

import (
	"context"
	"fmt"
	"os"
	"strings"

	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	k8sRuntimeUtil "k8s.io/apimachinery/pkg/util/runtime"

	v2 "github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/busy"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/ambassador/v2/pkg/logutil"
	"github.com/datawire/dlib/dgroup"
	"github.com/datawire/dlib/dlog"
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

// Main is a `github.com/datawire/ambassador/v2/pkg/busy`-compatible wrapper around 'Run()', using
// values appropriate for the stock Emissary.
func Main(ctx context.Context, version string, args ...string) error {
	dlog.Infof(ctx, "Emissary Ingress apiext (version %q)", version)
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "%s: error: expected exactly one argument, got %d\n", os.Args[0], len(args))
		fmt.Fprintf(os.Stderr, "Usage: %s APIEXT_SVCNAME\n", os.Args[0])
		os.Exit(2)
	}

	scheme := k8sRuntime.NewScheme()
	k8sRuntimeUtil.Must(v2.AddToScheme(scheme))
	k8sRuntimeUtil.Must(v3alpha1.AddToScheme(scheme))

	return Run(ctx, PodNamespace(), args[0], 8080, 8443, scheme)
}

// Run runs the Emissary apiext server process, but takes enough arguments that you should be able
// to reuse it to implement your own apiext server.
func Run(ctx context.Context, namespace, svcname string, httpPort, httpsPort int, scheme *k8sRuntime.Scheme) error {
	if lvl, err := logutil.ParseLogLevel(os.Getenv("APIEXT_LOGLEVEL")); err == nil {
		busy.SetLogLevel(lvl)
	}
	dlog.Infof(ctx, "APIEXT_LOGLEVEL=%v", busy.GetLogLevel())

	kubeinfo := k8s.NewKubeInfo("", "", "")
	restConfig, err := kubeinfo.GetRestConfig()
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
