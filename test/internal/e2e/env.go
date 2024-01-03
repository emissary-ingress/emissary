package e2e

import (
	"fmt"
	"os"

	"github.com/emissary-ingress/emissary/v3/test/internal/e2e/k3d"

	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// TestEnvironmentConfig provides basic configuration that can be used when generation a new
// E2E test environment.
type TestEnvironmentConfig struct {
	CRDConfigLocation string
	CRDConfigPattern  string

	// ClusterNamePrefix along with a random generated string are used to create a unique
	// cluster name for the test. If not supplied it defaults to "e2etest".
	ClusterNamePrefix string

	SetupFuncs []env.Func
}

type RegistryConfig struct {
	Name        string
	BindAddress string
	Port        int
}

// RegistryName is the name passed to the k3d create cluster
func (r RegistryConfig) RegistryName() string {
	return fmt.Sprintf("%s:%s:%d", r.Name, r.BindAddress, r.Port)
}

// RepositoryHostName is the repository name that should be used on the host (OSX/linux)
// This is what should be used when tagging and pushing from host. It is recommended to
// keep it as something like eg(127.0.0.1:10000) so that it works in Linux and OSX
func (r RegistryConfig) RepositoryHostName() string {
	return fmt.Sprintf("%s:%d", r.BindAddress, r.Port)
}

// RepositoryInClusterName is the repository name used in cluster in Deployment/Pod Manifest when
// referencing the registry
func (r RegistryConfig) RepositoryInClusterName() string {
	return fmt.Sprintf("k3d-%s:%d", r.Name, r.Port)
}

// TestEnvironment contains the instance of the environment and config used to create it
type TestEnvironment struct {
	Environment    env.Environment
	Config         *envconf.Config
	RegistryConfig RegistryConfig
}

// NewTestEnvironment generates a new e2e testing environment based on the provided configuration.
//
// It currently supports bringing your own cluster or will generate a local k3d cluster
// that can be used for running e2e tests.
func NewTestEnvironment(testEnvConfig *TestEnvironmentConfig) *TestEnvironment {
	testEnv := env.New()
	var cfg *envconf.Config

	registryConfig := RegistryConfig{
		Name:        "e2e-registry",
		BindAddress: "0.0.0.0",
		Port:        10000,
	}
	// Bring-your-own-cluster for faster iteration when debugging tests locally
	if _, ok := os.LookupEnv("BYO_CLUSTER"); ok {
		kubeConfigPath := conf.ResolveKubeConfigFile()
		cfg = envconf.NewWithKubeConfig(kubeConfigPath)
		testEnv = env.NewWithConfig(cfg)
	} else {
		cfg, _ = envconf.NewFromFlags()
		testEnv = env.NewWithConfig(cfg)

		clusterNamePrefix := "e2e"
		if testEnvConfig.ClusterNamePrefix != "" {
			clusterNamePrefix = testEnvConfig.ClusterNamePrefix
		}

		clusterName := envconf.RandomName(clusterNamePrefix, 16)

		k3sVersion := "v1.28.5-k3s1"
		if version, ok := os.LookupEnv("K3S_VERSION"); ok && version != "" {
			k3sVersion = version
		}

		setupFuncs := make([]env.Func, 0, len(testEnvConfig.SetupFuncs)+1)
		setupFuncs = append(setupFuncs, k3d.CreateCluster(clusterName, k3sVersion, registryConfig.RegistryName()))
		setupFuncs = append(setupFuncs, testEnvConfig.SetupFuncs...)
		testEnv.Setup(setupFuncs...)

		testEnv.Finish(
			k3d.DestroyCluster(clusterName),
		)
	}

	return &TestEnvironment{
		Environment:    testEnv,
		Config:         cfg,
		RegistryConfig: registryConfig,
	}
}
