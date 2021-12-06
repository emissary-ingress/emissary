package apiext

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"

	// k8s utils
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	"github.com/datawire/dlib/dhttp"
)

const (
	pathWebhooksCrdConvert = "/webhooks/crd-convert"
	pathProbesReady        = "/probes/ready"
	pathProbesLive         = "/probes/live"
)

func ServeHTTPS(ctx context.Context, port int, ca *CA, scheme *k8sRuntime.Scheme) error {
	webhook := &conversion.Webhook{}
	if err := webhook.InjectScheme(scheme); err != nil {
		return err
	}

	mux := http.NewServeMux()

	mux.Handle(pathWebhooksCrdConvert, webhook)

	sc := &dhttp.ServerConfig{
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return ca.GenServerCert(ctx, clientHello.ServerName)
			},
		},
	}
	return sc.ListenAndServeTLS(ctx, fmt.Sprintf(":%d", port), "", "")
}

func ServeHTTP(ctx context.Context, port int) error {
	mux := http.NewServeMux()

	mux.Handle(pathProbesReady, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "Ready!\n")
	}))
	mux.Handle(pathProbesLive, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "Living!\n")
	}))

	sc := &dhttp.ServerConfig{
		Handler: mux,
	}
	return sc.ListenAndServe(ctx, fmt.Sprintf(":%d", port))
}
