package entrypoint

import (
	"crypto/ed25519"
	"crypto/x509"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509/pkix"
	"math/big"
	"net/http"
	"time"
	"fmt"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"
)

const WEBHOOK_PORT int = 8043
const WEBHOOK_HOST string = "emissary-ingress.emissary.svc"
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

	cert, err := getCert()
	if err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", WEBHOOK_PORT)
	srv := &http.Server{
		Addr: addr,
		Handler: mux,
		TLSConfig: &tls.Config {
			Certificates: []tls.Certificate{*cert},
		},
	}

	return srv.ListenAndServeTLS("", "")
}

func getCert() (*tls.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(CERT_VALID_DAYS*24)*time.Hour)

	pubkey, privkey, err := ed25519.GenerateKey(rand.Reader)
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
		DNSNames: []string{WEBHOOK_HOST},
	}

	certRaw, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&pubkey,
		privkey,
	)
	if err != nil {
		return nil, err
	}

	var cert tls.Certificate
	cert.Certificate = append(cert.Certificate, certRaw)
	cert.PrivateKey = privkey
	return &cert, nil
}
