package apiext

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	// k8s utils
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	"github.com/datawire/dlib/dhttp"
	"github.com/datawire/dlib/dlog"
)

const (
	pathWebhooksCrdConvert = "/webhooks/crd-convert"
	pathProbesReady        = "/probes/ready"
	pathProbesLive         = "/probes/live"
)

func doConversion(wh *conversion.Webhook, w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		dlog.Errorf(r.Context(), "no conversion request provided?")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	inputBytes, err := ioutil.ReadAll(r.Body)

	if err != nil {
		dlog.Errorf(r.Context(), "could not read conversion request: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dlog.Infof(r.Context(), "INPUT: %s", string(inputBytes))

	r.Body = io.NopCloser(bytes.NewBuffer(inputBytes))
	rec := httptest.NewRecorder()
	wh.ServeHTTP(rec, r)

	dlog.Infof(r.Context(), "OUTPUT: %s", rec.Body)

	for k, v := range rec.Result().Header {
		w.Header()[k] = v
	}

	w.WriteHeader(rec.Code)
	//nolint:errcheck
	rec.Body.WriteTo(w)
}

func ServeHTTPS(ctx context.Context, port int, ca *CA, scheme *k8sRuntime.Scheme) error {
	webhook := &conversion.Webhook{}
	if err := webhook.InjectScheme(scheme); err != nil {
		return err
	}

	mux := http.NewServeMux()

	mux.Handle(pathWebhooksCrdConvert, webhook)

	dlog.Infof(ctx, "Serving HTTPS on port %d", port)

	sc := &dhttp.ServerConfig{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			doConversion(webhook, w, r)
		}),
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
