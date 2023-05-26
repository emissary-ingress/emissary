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

// PodNamespace determines the current Pods namespace
//
// Logic is borrowed from "k8s.io/client-go/tools/clientcmd".inClusterConfig.Namespace()
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

// Run the Emissary-ingress apiext conversion webhook using the provided configuration
func Run(ctx context.Context, namespace, svcname string, httpPort, httpsPort int, scheme *runtime.Scheme) error {
	if lvl, err := logutil.ParseLogLevel(os.Getenv("APIEXT_LOGLEVEL")); err == nil {
		busy.SetLogLevel(lvl)
	}
	dlog.Infof(ctx, "APIEXT_LOGLEVEL=%v", busy.GetLogLevel())

	kubeinfo := k8s.NewKubeInfo("", "", "")
	restConfig, err := kubeinfo.GetRestConfig()
	if err != nil {
		return err
	}

	ca, caSecret, err := apiext.EnsureCA(ctx, restConfig, namespace)
	if err != nil {
		return err
	}

	grp := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		EnableSignalHandling: true,
	})

	grp.Go("configure-crds", func(ctx context.Context) error {
		return apiext.ConfigureCRDs(ctx,
			restConfig,
			svcname,
			namespace,
			caSecret,
			scheme)
	})

	grp.Go("serve-http", func(ctx context.Context) error {
		return apiext.ServeHTTP(ctx, httpPort)
	})

	grp.Go("serve-https", func(ctx context.Context) error {
		return apiext.ServeHTTPS(ctx, httpsPort, ca, scheme)
	})

	return grp.Wait()
}
