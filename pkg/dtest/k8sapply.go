package dtest

import (
	"context"
	"os"
	"time"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/k8s"
	"github.com/emissary-ingress/emissary/v3/pkg/kubeapply"
)

// K8sApply applies the supplied manifests to the cluster indicated by
// the supplied kubeconfig.
func K8sApply(ctx context.Context, ver KubeVersion, files ...string) {
	if os.Getenv("DOCKER_REGISTRY") == "" {
		os.Setenv("DOCKER_REGISTRY", DockerRegistry(ctx))
	}
	kubeconfig := KubeVersionConfig(ctx, ver)
	err := kubeapply.Kubeapply(ctx, k8s.NewKubeInfo(kubeconfig, "", ""), 300*time.Second, false, false, files...)
	if err != nil {
		dlog.Println(ctx)
		dlog.Println(ctx, err)
		dlog.Printf(ctx,
			"Please note, if this is a timeout, then your kubernetes cluster may not "+
				"exist or may be unreachable. Check access to your cluster with \"kubectl --kubeconfig %s\".",
			kubeconfig)
		dlog.Println(ctx)
		cmd := dexec.CommandContext(ctx,
			"kubectl", "--kubeconfig", kubeconfig,
			"get", "--all-namespaces", "ns,svc,deploy,po",
		)
		_ = cmd.Run() // Command output and any error will be logged

		os.Exit(1)
	}
}
