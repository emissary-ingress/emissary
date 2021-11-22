package apiext

import (
	"context"
	"os"
	"strings"

	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	k8sRuntimeUtil "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/k8s"
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
// PodNamespace() and a scheme containing the
// `github.com/datawire/ambassador/v2/pkg/api/getambassador.io/...` types.
func Main(ctx context.Context, version string, args ...string) error {
	dlog.Infof(ctx, "Emissary Ingress apiext (version %q)", version)

	scheme := k8sRuntime.NewScheme()
	k8sRuntimeUtil.Must(v2.AddToScheme(scheme))
	k8sRuntimeUtil.Must(v3alpha1.AddToScheme(scheme))

	return Run(ctx, PodNamespace(), 8443, scheme)
}

// Run runs the Emissary apiext server process, but takes enough arguments that you should be able
// to reuse it to implement your own apiext server.
func Run(ctx context.Context, namespace string, port int, scheme *k8sRuntime.Scheme) error {
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
		return ConfigureCRDs(ctx, restConfig, caSecret, scheme)
	})

	grp.Go("serve-https", func(ctx context.Context) error {
		return Serve(ctx, port, ca, scheme)
	})

	return grp.Wait()
}
