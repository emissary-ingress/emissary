package apiext_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/emissary-ingress/emissary/v3/pkg/apiext/certutils"
	apiextdefaults "github.com/emissary-ingress/emissary/v3/pkg/apiext/defaults"
	"github.com/emissary-ingress/emissary/v3/test/internal/e2e"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	apiextDirPath                     = "testdata"
	apiextDeploymentPattern           = "deployment.yaml"
	apiextDeploymentExtManagedPattern = "ext-managed-deployment.yaml"
	crdDirPath                        = "testdata"
	crdPattern                        = "crds.yaml"
	getambassadorioDirPath            = "testdata"
	getambassadorioPattern            = "getambassadorio-resources.yaml"
	rbacDirPath                       = "testdata"
	rbacPattern                       = "rbac.yaml"
	certmgrVer                        = "v1.13.1"
	certMgrUrl                        = fmt.Sprintf("https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml", certmgrVer)
	certMgrDirPath                    = "testdata"
	certMgrDirPattern                 = "cert.yaml"
)

func TestAPIExtWatchesCACertChanges(t *testing.T) {
	feature := features.New("apiext self managed ca cert with renewal").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			log.Println("installing CRDs into cluster")
			if _, err := installCRDs(crdDirPath, crdPattern)(ctx, cfg); err != nil {
				t.Fatal(err)
			}

			if err := installAPIExtRBAC(ctx, cfg.Client().Resources()); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("APIExt manages CA Cert and CRD patching", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			if err := installAPIExtDeployment(ctx, cfg, apiextDeploymentPattern); err != nil {
				t.Fatal(err)
			}

			namespace := ctx.Value(e2e.GetNamespaceKey(t)).(string)
			if err := createGetAmbassadorioResources(ctx, r, namespace); err != nil {
				t.Fatal(err)
			}

			if err := deleteRootCACert(ctx, r); err != nil {
				t.Fatal(err)
			}

			if err := assertGetAmbassadorioResources(ctx, r, namespace); err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := cleanupCRDs(ctx, cfg, crdDirPath, crdPattern); err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Feature()

	_ = testEnv.Environment.Test(t, feature)
}

func TestAPIExtRecreatesExpiredCACert(t *testing.T) {
	feature := features.New("apiext self-managed with expired-cert renewal").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if _, err := installCRDs(crdDirPath, crdPattern)(ctx, cfg); err != nil {
				t.Fatal(err)
			}

			if err := installAPIExtRBAC(ctx, cfg.Client().Resources()); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("APIExt manages CA Cert and CRD patching", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			expiredCACert, err := generateExpiredRootCACertSecret()
			if err != nil {
				t.Fatal(err)
			}

			if err := r.Create(ctx, expiredCACert); err != nil {
				t.Fatal(err)
			}

			if err := installAPIExtDeployment(ctx, cfg, apiextDeploymentPattern); err != nil {
				t.Fatal(err)
			}
			namespace := ctx.Value(e2e.GetNamespaceKey(t)).(string)
			if err := createGetAmbassadorioResources(ctx, r, namespace); err != nil {
				t.Fatal(err)
			}

			if err := assertGetAmbassadorioResources(ctx, r, namespace); err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := cleanupCRDs(ctx, cfg, crdDirPath, crdPattern); err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Feature()

	_ = testEnv.Environment.Test(t, feature)
}

func TestAPIExtExternallyManageCACert(t *testing.T) {
	feature := features.New("apiext externally managed ca cert").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := installCertManager(ctx, cfg); err != nil {
				t.Fatal(err)
			}

			if err := installCertManagerCertificate(ctx, cfg.Client().Resources(), apiextdefaults.WebhookCASecretNamespace); err != nil {
				t.Fatal(err)
			}

			injectCABundleAnnotation := decoder.MutateAnnotations(map[string]string{
				"cert-manager.io/inject-ca-from": fmt.Sprintf("%s/%s", apiextdefaults.APIExtNamespace, apiextdefaults.WebhookCASecretName),
			})

			if _, err := installCRDs(crdDirPath, crdPattern, injectCABundleAnnotation)(ctx, cfg); err != nil {
				t.Fatal(err)
			}

			if err := installAPIExtRBAC(ctx, cfg.Client().Resources()); err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("APIExt manages CA Cert and CRD patching", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			if err := installAPIExtDeployment(ctx, cfg, apiextDeploymentExtManagedPattern); err != nil {
				t.Fatal(err)
			}

			namespace := ctx.Value(e2e.GetNamespaceKey(t)).(string)
			if err := createGetAmbassadorioResources(ctx, r, namespace); err != nil {
				t.Fatal(err)
			}

			if err := assertGetAmbassadorioResources(ctx, r, namespace); err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := cleanupCRDs(ctx, cfg, crdDirPath, crdPattern); err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Feature()

	_ = testEnv.Environment.Test(t, feature)
}

func installCRDs(crdPath, pattern string, options ...decoder.DecodeOption) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		log.Println("installing CRDs into cluster")
		r, err := getResourcesWithAPIExtScheme(cfg)
		if err != nil {
			return ctx, err
		}
		err = decoder.ApplyWithManifestDir(ctx, r, crdPath, pattern, []resources.CreateOption{}, options...)
		if err != nil {
			return ctx, err
		}

		crds, err := getGetAmbassadorioCRDList(ctx, cfg)
		if err != nil {
			return ctx, err
		}

		if err := wait.For(
			conditions.New(cfg.Client().Resources()).
				ResourcesMatch(crds, e2e.CustomResourceDefinitionConditions()),
			wait.WithTimeout(10*time.Second),
			wait.WithInterval(1*time.Second),
		); err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}

func cleanupCRDs(ctx context.Context, cfg *envconf.Config, crdPath, pattern string) error {
	r, err := getResourcesWithAPIExtScheme(cfg)
	if err != nil {
		return err
	}
	crds, err := getGetAmbassadorioCRDList(ctx, cfg)
	if err != nil {
		return err
	}

	if err := decoder.DeleteWithManifestDir(ctx, r, crdPath, pattern,
		[]resources.DeleteOption{}); err != nil {
		return err
	}

	return wait.For(
		conditions.New(r).ResourcesDeleted(crds),
		wait.WithTimeout(30*time.Second),
	)
}

func getGetAmbassadorioCRDList(ctx context.Context, cfg *envconf.Config) (*apiextv1.CustomResourceDefinitionList, error) {
	r, err := getResourcesWithAPIExtScheme(cfg)
	if err != nil {
		return nil, err
	}

	crdList := &apiextv1.CustomResourceDefinitionList{}
	options := []resources.ListOption{
		resources.WithLabelSelector("app.kubernetes.io/part-of=emissary-apiext"),
	}

	err = r.List(ctx, crdList, options...)
	if err != nil {
		return nil, err
	}

	return crdList, nil
}

func getResourcesWithAPIExtScheme(cfg *envconf.Config) (*resources.Resources, error) {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return nil, err
	}

	if err := apiextv1.AddToScheme(r.GetScheme()); err != nil {
		return nil, err
	}

	return r, nil
}

func installAPIExtRBAC(ctx context.Context, r *resources.Resources) error {
	log.Println("installing APIEX RBAC into cluster...")
	return decoder.DecodeEachFile(ctx,
		os.DirFS(rbacDirPath),
		rbacPattern,
		decoder.CreateHandler(r),
	)
}

func installAPIExtDeployment(ctx context.Context, cfg *envconf.Config, name string) error {
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return err
	}

	log.Println("Deploying Emissary-ingress Apiext deployment...")

	if err := decoder.DecodeEachFile(ctx, os.DirFS(apiextDirPath),
		name,
		decoder.CreateHandler(r)); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	log.Println("Emissary-ingress apiext deployment created")
	log.Println("waiting for Emissary-ingress apiext deployment to be ready...")
	apiextDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiextdefaults.APIEXTDeploymentName,
			Namespace: apiextdefaults.APIExtNamespace,
		},
	}

	if err := wait.For(conditions.New(r).ResourceMatch(apiextDeployment, func(object k8s.Object) bool {
		d := object.(*appsv1.Deployment)
		log.Printf("AvailableReplicas == %d, ReadyReplicas == %d\n",
			d.Status.AvailableReplicas, d.Status.ReadyReplicas)
		return d.Status.AvailableReplicas == 3 && d.Status.ReadyReplicas == 3
	}),
		wait.WithContext(ctx),
		wait.WithInterval(2*time.Second),
		wait.WithTimeout(20*time.Second),
	); err != nil {
		return fmt.Errorf("emissary-ingress apiext failed to become ready: %w", err)
	}

	return nil
}

// generateExpiredRootCACertSecret to simulate an expired cert needing to be recreated.
func generateExpiredRootCACertSecret() (*corev1.Secret, error) {
	pk, cert, err := certutils.GenerateRootCACert("e2e-test", -48*time.Hour)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiextdefaults.APIExtNamespace,
			Namespace: apiextdefaults.APIExtNamespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSPrivateKeyKey: pk,
			corev1.TLSCertKey:       cert,
		},
	}, nil
}

func deleteRootCACert(ctx context.Context, r *resources.Resources) error {
	log.Println("deleting root CA secret from cluster...")
	return r.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiextdefaults.WebhookCASecretName,
			Namespace: apiextdefaults.APIExtNamespace,
		},
	},
	)
}

// createGetAmbassadorioResources loads and parse testdata manifests and applys them
// to the cluster. The apiext server must be working to conver them to the storage
// version of v2.
func createGetAmbassadorioResources(ctx context.Context, r *resources.Resources, namespace string) error {
	log.Println("creating getambassador.io resources in cluster")
	items, err := decoder.DecodeAllFiles(ctx,
		os.DirFS(getambassadorioDirPath),
		getambassadorioPattern,
		decoder.MutateNamespace(namespace),
	)
	if err != nil {
		return err
	}

	// AFAIK, there is no way to know if k8s has accepted the new CA Bundle and will use it for
	// verifying new conversion web hook connections. Due to this race condition there is a chance
	// that a user asks for a resource to be converted and thus gets an error because the server
	// cert cannot be validated. Therefore, to reduce flakiness of the test we should retry
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.MaxElapsedTime = 30 * time.Second

	for i, item := range items {
		if err := backoff.Retry(func() error {
			log.Printf("creating getambassadorio item-%d --> %s:%s\n", i,
				item.GetObjectKind().GroupVersionKind().Kind,
				item.GetName())
			return r.Create(ctx, item)
		}, backoffConfig); err != nil {
			return err
		}
	}

	return nil
}

// assertGetAmbassadorioResources verifies that we are successfully able to get the resources
// from the server. If they can't be found then the ApiExt server is not successfully converting
// v2 to v3alpha1
func assertGetAmbassadorioResources(ctx context.Context, r *resources.Resources, namespace string) error {
	crItems, err := decoder.DecodeAllFiles(ctx, os.DirFS(getambassadorioDirPath), getambassadorioPattern,
		decoder.MutateNamespace(namespace),
	)
	if err != nil {
		return err
	}
	for _, item := range crItems {
		log.Printf("assert getambassadorio item: %s-%s\n", item.GetObjectKind().GroupVersionKind().Kind, item.GetName())
		if err := wait.For(conditions.New(r).ResourceMatch(item, func(object k8s.Object) bool { return true })); err != nil {
			return err
		}
	}

	return nil
}

func installCertManager(ctx context.Context, cfg *envconf.Config) error {
	kubeconfig := cfg.KubeconfigFile()
	if p := utils.RunCommand(fmt.Sprintf("kubectl apply --kubeconfig %s -f %s", kubeconfig, certMgrUrl)); p.Err() != nil {
		log.Printf("Failed to deploy cert-manager: %s: %s", p.Err(), p.Out())
		return p.Err()
	}

	// wait for certmgr to be ready
	log.Println("Waiting for cert-manager deployment to be available...")
	if err := wait.For(
		conditions.New(cfg.Client().Resources()).DeploymentAvailable("cert-manager-webhook", "cert-manager"),
		wait.WithTimeout(5*time.Minute),
		wait.WithInterval(10*time.Second),
	); err != nil {
		log.Printf("Timedout while waiting for cert-manager deployment: %s", err)
		return err
	}

	return nil
}

func installCertManagerCertificate(ctx context.Context, r *resources.Resources, namespace string) error {
	log.Println("installing certmgtr Certificate and Issuer...")
	return decoder.ApplyWithManifestDir(ctx, r, certMgrDirPath, certMgrDirPattern, []resources.CreateOption{}, decoder.MutateNamespace(namespace))
}
