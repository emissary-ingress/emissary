package entrypoint

import (
	b "bytes"
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
	"time"

	k8sTypesCoreV1 "k8s.io/api/core/v1"
	k8sApiExtTypes "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8sApiExtClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	k8sTypesMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8sClientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/k8s"
	"github.com/datawire/dlib/dhttp"
)

const (
	webhookPort = 8006
	webhookPath = "/crdconvert"
)

// TODO: automatic cert regeneration
const CERT_VALID_DAYS int = 365
const CA_SECRET_NAME string = "emissary-ingress-webhook-ca"

func stringPtr(x string) *string {
	return &x
}

func int32Ptr(x int32) *int32 {
	return &x
}

func handleWebhooks(ctx context.Context) error {
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v2.AddToScheme(scheme))
	utilruntime.Must(v3alpha1.AddToScheme(scheme))

	// Create the webhook server
	webhook := &conversion.Webhook{}
	if err := webhook.InjectScheme(scheme); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc(webhookPath, webhook.ServeHTTP)

	// need a k8s client
	kubeinfo := k8s.NewKubeInfo("", "", "")
	restConfig, err := kubeinfo.GetRestConfig()
	if err != nil {
		return err
	}
	coreClient, err := k8sClientCoreV1.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// get CA secret
	secrets := coreClient.Secrets(GetAmbassadorNamespace())
	caSecret, err := secrets.Get(ctx, CA_SECRET_NAME, k8sTypesMetaV1.GetOptions{})
	var caPEM *b.Buffer
	var caTemplate *x509.Certificate
	var caPrivKey *rsa.PrivateKey
	if err != nil {
		// Error here most likely means the secret doesnt exist yet
		// (then we make one). Otherwise if its an actual error we exit here
		if !k8sErrors.IsNotFound(err) {
			return err
		}

		// Generate CA Certificate and key...
		notBefore := time.Now()
		notAfter := notBefore.Add(time.Duration(CERT_VALID_DAYS*24) * time.Hour)
		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
		if err != nil {
			return err
		}
		caTemplate = &x509.Certificate{
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
		caPrivKey, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return err
		}
		caBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caPrivKey.PublicKey, caPrivKey)
		if err != nil {
			return err
		}

		// PEM encode certificate and key
		caPEM = new(b.Buffer)
		if err := pem.Encode(caPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caBytes,
		}); err != nil {
			return err
		}
		caPrivKeyPEM := new(b.Buffer)
		if err := pem.Encode(caPrivKeyPEM, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
		}); err != nil {
			return err
		}

		// Create and write the secret
		_, err = secrets.Create(ctx, &k8sTypesCoreV1.Secret{
			ObjectMeta: k8sTypesMetaV1.ObjectMeta{
				Name:      CA_SECRET_NAME,
				Namespace: GetAmbassadorNamespace(),
			},
			Type: k8sTypesCoreV1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.key": caPrivKeyPEM.Bytes(),
				"tls.crt": caPEM.Bytes(),
			},
		}, k8sTypesMetaV1.CreateOptions{})
		if err != nil {
			if k8sErrors.IsAlreadyExists(err) {
				caSecret, err = secrets.Get(ctx, CA_SECRET_NAME, k8sTypesMetaV1.GetOptions{})
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// CA Secret already exists, so load data from it
	if caPEM == nil || caTemplate == nil || caPrivKey == nil {
		if caSecret == nil {
			return fmt.Errorf("Couldnt get or generate CA secret")
		}

		// Parse CA Key
		caPrivKeyPEMBytes, ok := caSecret.Data["tls.key"]
		if ok {
			keyBlock, _ := pem.Decode(caPrivKeyPEMBytes)
			if key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes); err == nil {
				caPrivKey = key
			} else {
				return fmt.Errorf("Bad key loaded in CA secret: %s", err.Error())
			}
		} else {
			return fmt.Errorf("No key found in CA secret!")
		}

		// parse ca cert
		caPEMBytes, ok := caSecret.Data["tls.crt"]
		if ok {
			// we need the PEM and the parsed cert
			caPEM = b.NewBuffer(caPEMBytes)
			certBlock, rest := pem.Decode(caPEMBytes)
			if string(rest) != "" || certBlock.Type != "CERTIFICATE" {
				return fmt.Errorf("Bad cert loaded in CA secret")
			}
			caTemplate, err = x509.ParseCertificate(certBlock.Bytes)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("No cert found in CA secret!")
		}
	}

	// these need to be pointer vals for some reason
	webhookConfig := k8sApiExtTypes.CustomResourceConversion{
		Strategy: k8sApiExtTypes.WebhookConverter,
		WebhookClientConfig: &k8sApiExtTypes.WebhookClientConfig{
			Service: &k8sApiExtTypes.ServiceReference{
				Namespace: GetAmbassadorNamespace(),
				Name:      GetAdminService(),
				Path:      stringPtr(webhookPath),
				Port:      int32Ptr(webhookPort),
			},
			CABundle: caPEM.Bytes(),
		},
	}

	// get list of CRDs
	apiExtClient, err := k8sApiExtClient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	crdInterface := apiExtClient.CustomResourceDefinitions()
	crds, err := crdInterface.List(ctx, k8sTypesMetaV1.ListOptions{})
	if err != nil {
		return err
	}
	var etext string
	for _, crd := range crds.Items {
		crd.Spec.Conversion = &webhookConfig
		_, err := crdInterface.Update(ctx, &crd, k8sTypesMetaV1.UpdateOptions{})
		if err != nil {
			etext += err.Error() + "\n"
		}
	}
	if len(etext) > 0 {
		return fmt.Errorf(etext)
	}

	// finally, put up that webhook server
	sc := &dhttp.ServerConfig{
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return getCert(clientHello.ServerName, *caTemplate, caPrivKey)
			},
		},
	}
	return sc.ListenAndServeTLS(ctx, fmt.Sprintf(":%d", webhookPort), "", "")
}

// generates a server cert given hostname and CA cert
func getCert(hostname string, rootCert x509.Certificate, rootKey *rsa.PrivateKey) (*tls.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(CERT_VALID_DAYS*24) * time.Hour)

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
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
		&template,
		&rootCert,
		priv.Public(),
		rootKey,
	)
	if err != nil {
		return nil, err
	}

	var cert tls.Certificate
	cert.Certificate = append(cert.Certificate, certRaw)
	cert.PrivateKey = priv
	return &cert, nil
}
