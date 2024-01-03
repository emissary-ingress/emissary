package apiext_test

import (
	"context"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	apiextdefaults "github.com/emissary-ingress/emissary/v3/pkg/apiext/defaults"
	"github.com/emissary-ingress/emissary/v3/test/internal/e2e"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

var testEnv *e2e.TestEnvironment

func TestMain(m *testing.M) {
	if _, ok := os.LookupEnv("APIEXT_E2E"); !ok {
		log.Println("skipping apiext e2e tests main, set APIEXT_E2E environment variable")
		os.Exit(0)
	}

	testEnvConfig := &e2e.TestEnvironmentConfig{
		SetupFuncs: []env.Func{
			setupAPIExtTestDependencies,
		},
	}
	testEnv = e2e.NewTestEnvironment(testEnvConfig)

	runID := "apiexte2e"
	testEnv.Environment.BeforeEachTest(func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
		ctx, err := e2e.CreateNSForTest(ctx, cfg, t, runID)
		if err != nil {
			return ctx, err
		}

		return envfuncs.CreateNamespace(apiextdefaults.WebhookCASecretNamespace)(ctx, cfg)
	})
	testEnv.Environment.AfterEachTest(func(ctx context.Context, cfg *envconf.Config, t *testing.T) (context.Context, error) {
		ctx, err := e2e.DeleteNSForTest(ctx, cfg, t)
		if err != nil {
			return ctx, err
		}

		ctx, err = envfuncs.DeleteNamespace(apiextdefaults.WebhookCASecretNamespace)(ctx, cfg)
		if err != nil {
			return ctx, err
		}
		r, err := getResourcesWithAPIExtScheme(cfg)
		if err != nil {
			return ctx, err
		}

		if err := rbacv1.AddToScheme(r.GetScheme()); err != nil {
			return ctx, err
		}

		ns := &corev1.Namespace{}
		ns.Name = apiextdefaults.WebhookCASecretNamespace

		err = wait.For(
			conditions.New(r).ResourceDeleted(ns),
			wait.WithTimeout(30*time.Second),
		)

		crb := &rbacv1.ClusterRoleBinding{}
		crb.Name = "emissary-apiext"
		if err := r.Delete(ctx, crb); err != nil {
			return ctx, err
		}

		cr := &rbacv1.ClusterRole{}
		cr.Name = "emissary-apiext"
		if err := r.Delete(ctx, cr); err != nil {
			return ctx, err
		}
		return ctx, err
	})
	os.Exit(testEnv.Environment.Run(m))
}

func setupAPIExtTestDependencies(ctx context.Context, c *envconf.Config) (context.Context, error) {
	log.Println("Running e2e setup for apiext server (gen crd manifest, deployment manifest and build container).")

	cmd := exec.CommandContext(ctx, "make", "-C", "../..", "apiext-e2e-setup")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Failed to setup apiext for e2e testing: %s\n", err)
		return ctx, err
	}

	return ctx, nil
}
