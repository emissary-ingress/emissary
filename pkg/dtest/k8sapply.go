package dtest

import (
	"context"
	"os"
	"time"

	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/ambassador/v2/pkg/kubeapply"
	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
)

// K8sApply applies the supplied manifests to the cluster indicated by
// the supplied kubeconfig.
func K8sApply(ctx context.Context, ver KubeVersion, files ...string) {
	if os.Getenv("DOCKER_REGISTRY") == "" {
		os.Setenv("DOCKER_REGISTRY", DockerRegistry(ctx))
	}
	kubeconfig := KubeVersionConfig(ctx, ver)
	kubeclient, err := kates.NewClient(kates.ClientConfig{Kubeconfig: kubeconfig})
	if err != nil {
		dlog.Errorln(ctx, err)
		os.Exit(1)
	}
	if err := kubeapply.Kubeapply(ctx, kubeclient, 300*time.Second, false, false, files...); err != nil {
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
