package k3d

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// CreateCluster returns an env.Func that is used to
// create a K3d cluster that is then injected in the context
// using the name as a key.
//
// NOTE: the returned function will update its env config with the
// kubeconfig file for the config client.
func CreateCluster(clusterName string, k3sVersion string, registryName string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		args := []string{
			"cluster",
			"create",
			"--wait",
			"--kubeconfig-update-default=false",
			"--k3s-arg=--disable=traefik@server:*",
			"--k3s-arg=--kubelet-arg=max-pods=255@server:*",
			"--k3s-arg=--egress-selector-mode=disabled@server:*",
		}

		if registryName != "" {
			args = append(args, fmt.Sprintf("--registry-create=%s", registryName))
		}

		if k3sVersion != "" {
			args = append(args, fmt.Sprintf("--image=docker.io/rancher/k3s:%s", k3sVersion))
		}

		args = append(args, clusterName)

		cmd := exec.CommandContext(ctx, "k3d", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return ctx, err
		}

		homedir, err := os.UserHomeDir()
		if err != nil {
			return ctx, nil
		}

		kubeconfigFile := filepath.Join(homedir, ".k3d", fmt.Sprintf("kubeconfig-%s.yaml", clusterName))

		// k3d kubeconfig get
		cmd = exec.CommandContext(ctx, "k3d", "kubeconfig", "merge",
			clusterName, "--output", kubeconfigFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return ctx, err
		}

		// update envconfig  with kubeconfig
		cfg.WithKubeconfigFile(kubeconfigFile)

		return ctx, nil
	}
}

// DestroyCluster returns an EnvFunc that
// retrieves a previously saved k3d cluster in the context (using the name), then deletes it.
//
// NOTE: this should be used in a Environment.Finish step.
func DestroyCluster(clusterName string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		cmd := exec.CommandContext(ctx, "k3d", "cluster", "delete", clusterName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return ctx, err
		}

		kubePath, err := GetK3dClusterConfigPath(clusterName)
		if err != nil {
			return ctx, err
		}

		return ctx, os.Remove(kubePath)
	}
}

// GetK3dClusterConfigPath provides the path to the kubeconfig file for
// a the provided k3d cluster.
//
// This is the same path used by "CreateK3dCluster" so it can be used
// in conjunction.
func GetK3dClusterConfigPath(clusterName string) (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(homedir, ".k3d", fmt.Sprintf("kubeconfig-%s.yaml", clusterName))

	return configPath, nil
}
