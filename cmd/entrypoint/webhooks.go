package entrypoint

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"reflect"
	"time"

	// k8s types
	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sTypesAPIExtV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// k8s clients
	k8sClientAPIExtV1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// k8s utils
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/dlib/derror"
	"github.com/datawire/dlib/dhttp"
)

const (
	webhookPath   = "/crdconvert"
	certValidDays = 365
	caSecretName  = "emissary-ingress-webhook-ca"
)

// TODO: automatic cert regeneration

func stringPtr(x string) *string {
	return &x
}

func int32Ptr(x int32) *int32 {
	return &x
}

func GetEmissaryScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	utilruntime.Must(v2.AddToScheme(scheme))
	utilruntime.Must(v3alpha1.AddToScheme(scheme))
	return scheme
}

// genKey generates an RSA key, returning both the key object, as well as a representation of it as
// PEM-encoded PKCS#8 DER.
func genKey() (*rsa.PrivateKey, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	derBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	})
	return key, pemBytes, nil
}

// genCACert generates a Certificate Authority's certificate, returning both the certificate object,
// as well as a representation of it as PEM-encoded DER.
func genCACert(key *rsa.PrivateKey) (*x509.Certificate, []byte, error) {
	// Generate CA Certificate and key...
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(certValidDays*24) * time.Hour)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Ambassador Labs"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	return template, pemBytes, nil
}

func InitializeCRDs(ctx context.Context, webhookPort int32, scheme *runtime.Scheme) (*CA, error) {
	// need a k8s client
	kubeinfo := k8s.NewKubeInfo("", "", "")
	restConfig, err := kubeinfo.GetRestConfig()
	if err != nil {
		return nil, err
	}
	coreClient, err := k8sClientCoreV1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	secretsClient := coreClient.Secrets(GetAmbassadorNamespace())

	// get CA secret
	var (
		caSecret *k8sTypesCoreV1.Secret

		caCert         *x509.Certificate
		caCertPEMBytes []byte
		caKey          *rsa.PrivateKey
		caKeyPEMBytes  []byte
	)
	caSecret, err = secretsClient.Get(ctx, caSecretName, k8sTypesMetaV1.GetOptions{})
	if err != nil {
		// Error here most likely means the secret doesn't exist yet (so we should make
		// one).  Otherwise if it's an actual error we exit here.
		if !k8sErrors.IsNotFound(err) {
			return nil, err
		}

		// Generate the CA
		caKey, caKeyPEMBytes, err = genKey()
		if err != nil {
			return nil, err
		}
		caCert, caCertPEMBytes, err = genCACert(caKey)
		if err != nil {
			return nil, err
		}

		// Create and write the secret
		caSecret, err = secretsClient.Create(ctx, &k8sTypesCoreV1.Secret{
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      caSecretName,
				Namespace: GetAmbassadorNamespace(),
			},
			Type: k8sTypesCoreV1.SecretTypeTLS,
			Data: map[string][]byte{
				k8sTypesCoreV1.TLSPrivateKeyKey: caKeyPEMBytes,
				k8sTypesCoreV1.TLSCertKey:       caCertPEMBytes,
			},
		}, k8sTypesMetaV1.CreateOptions{})
		if err != nil {
			if !k8sErrors.IsAlreadyExists(err) {
				return nil, err
			}
			caSecret, err = secretsClient.Get(ctx, caSecretName, k8sTypesMetaV1.GetOptions{})
			if err != nil {
				return nil, err
			}
			caCert = nil
			caCertPEMBytes = nil
			caKey = nil
			caKeyPEMBytes = nil
		}
	}

	// Parse the Secret if nescessary
	if caKey == nil || len(caKeyPEMBytes) == 0 {
		// parse CA key
		var ok bool
		caKeyPEMBytes, ok = caSecret.Data[k8sTypesCoreV1.TLSPrivateKeyKey]
		if !ok {
			return nil, fmt.Errorf("no key found in CA secret!")
		}
		caKeyBlock, _ := pem.Decode(caKeyPEMBytes)
		_caKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("bad key loaded in CA secret: %w", err)
		}
		caKey, ok = _caKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key in CA secret is not an RSA key")
		}
	}
	if caCert == nil || len(caCertPEMBytes) == 0 {
		// parse CA cert
		var ok bool
		caCertPEMBytes, ok = caSecret.Data[k8sTypesCoreV1.TLSCertKey]
		if !ok {
			return nil, fmt.Errorf("no cert found in CA secret!")
		}
		caCertBlock, _ := pem.Decode(caCertPEMBytes)
		caCert, err = x509.ParseCertificate(caCertBlock.Bytes)
		if err != nil {
			return nil, err
		}
	}

	// Populate the CA cert and webhook info in to the CRDs.
	conversionConfig := &k8sTypesAPIExtV1.CustomResourceConversion{
		Strategy: k8sTypesAPIExtV1.WebhookConverter,
		Webhook: &k8sTypesAPIExtV1.WebhookConversion{
			ClientConfig: &k8sTypesAPIExtV1.WebhookClientConfig{
				Service: &k8sTypesAPIExtV1.ServiceReference{
					Namespace: GetAmbassadorNamespace(),
					Name:      GetAdminService(),
					Path:      stringPtr(webhookPath),
					Port:      int32Ptr(webhookPort),
				},
				CABundle: caSecret.Data[k8sTypesCoreV1.TLSPrivateKeyKey],
			},
			// Which versions of the conversion API our webhook supports.  Since we use
			// sigs.k8s.io/controller-runtime/pkg/webhook/conversion to implement the
			// webhook this list should be kept in-sync with what that package supports.
			ConversionReviewVersions: []string{
				"v1beta1",
			},
		},
	}
	apiExtClient, err := k8sClientAPIExtV1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	crdsClient := apiExtClient.CustomResourceDefinitions()
	crds, err := crdsClient.List(ctx, k8sTypesMetaV1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var count int
	var errs derror.MultiError
	for _, crd := range crds.Items {
		// Versions is a mandatory field we can rely on
		// to have at least 1 value. Regardless, we
		// protect against len=0 in the conditional
		if len(crd.Spec.Versions) < 1 || !scheme.Recognizes(schema.GroupVersionKind{
			Group:   crd.Spec.Group,
			Version: crd.Spec.Versions[0].Name,
			Kind:    crd.Spec.Names.Kind,
		}) {
			continue
		}

		count++
		if reflect.DeepEqual(crd.Spec.Conversion, conversionConfig) {
			continue
		}
		crd.Spec.Conversion = conversionConfig
		_, err := crdsClient.Update(ctx, &crd, k8sTypesMetaV1.UpdateOptions{})
		if err != nil && !k8sErrors.IsConflict(err) {
			errs = append(errs, err)
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("found no CRD types to add webhooks to!")
	}
	if len(errs) > 0 {
		return nil, errs
	}

	return &CA{
		Cert: caCert,
		Key:  caKey,
	}, nil
}

func ServeWebhooks(ctx context.Context, ca *CA, webhookPort int32, scheme *runtime.Scheme) error {
	webhook := &conversion.Webhook{}
	if err := webhook.InjectScheme(scheme); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc(webhookPath, webhook.ServeHTTP)

	sc := &dhttp.ServerConfig{
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return ca.GetCert(clientHello.ServerName)
			},
		},
	}
	return sc.ListenAndServeTLS(ctx, fmt.Sprintf(":%d", webhookPort), "", "")
}

type CA struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
}

// generates a server cert given hostname and CA cert
func (ca *CA) GetCert(hostname string) (*tls.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(certValidDays*24) * time.Hour)

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Ambassador Labs"},
			CommonName:   "Webhook API",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{hostname},
	}

	certRaw, err := x509.CreateCertificate(
		rand.Reader,
		template,
		ca.Cert,
		priv.Public(),
		ca.Key,
	)
	if err != nil {
		return nil, err
	}

	var cert tls.Certificate
	cert.Certificate = append(cert.Certificate, certRaw)
	cert.PrivateKey = priv
	return &cert, nil
}
