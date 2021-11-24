package entrypoint

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"time"
	"fmt"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/k8scerts"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"
)

const WEBHOOK_PORT int = 8043
// TODO: automatic cert regeneration
const CERT_VALID_DAYS int = 365

func handleWebhooks() error {
	scheme := runtime.NewScheme()

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v2.AddToScheme(scheme))
	utilruntime.Must(v3alpha1.AddToScheme(scheme))

	// Create the webhook server
	webhook := &conversion.Webhook{}
	webhook.InjectScheme(scheme)
	mux := http.NewServeMux()
	mux.HandleFunc("/crdconvert", webhook.ServeHTTP)

	// try to get a CA Cert out of the k8scerts stored PEM
	certBlock, rest := pem.Decode(k8scerts.K8sCert)
	if rest != "" || block.Type != "CERTIFICATE" {
		return fmt.Errorf("Bad cert loaded in k8scerts")
	}
	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return err
	}

	// Try to parse a Private Key out of the k8scerts stored PEM
	var caKey crypto.PrivateKey
	keyBlock, _ := pem.Decode(k8scerts.K8sKey)
	if key, err := x509.ParsePKCS1PrivateKey(keyBlock); err == nil {
		caKey = key
	} else if key, err := x509.ParsePKCS8PrivateKey(keyBlock); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey:
			caKey = key
		default:
			return fmt.Errorf("Found unknown private key type in PKCS#8 wrapping")
		}
	} else if key, err := x509.ParseECPrivateKey(keyBlock); err == nil {
		caKey = key
	} else {
		return fmt.Errorf("Bad key loaded in k8scerts")
	}

	addr := fmt.Sprintf(":%d", WEBHOOK_PORT)
	srv := &http.Server{
		Addr: addr,
		Handler: mux,
		TLSConfig: &tls.Config {
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return getCert(clientHello.ServerName, caCert, caKey)
			},
		},
	}

	return srv.ListenAndServeTLS("", "")
}

func getCert(hostname string, rootCert x509.Certificate, rootKey *rsa.PrivateKey) (*tls.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(CERT_VALID_DAYS*24)*time.Hour)

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
		DNSNames: []string{hostname},
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
