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

// conversionWithLogging is a wrapper around our real conversion method that logs the JSON
// input and output for the conversion request. It's used only when we have debug logging
// enabled.
func conversionWithLogging(handler http.Handler, w http.ResponseWriter, r *http.Request) {
	// This is a little more obnoxious than you'd think because r.Body is a ReadCloser,
	// not an io.Reader, and because the handler expects to be handed an io.Writer for
	// response. So we need to buffer both directions (obviously, this works partly
	// because we know that requests and responses are fairly small... ish).
	//
	// So, start by reading the request body into a byte array using iotuil.ReadAll
	// That's the easy bit.
	if r.Body == nil {
		dlog.Errorf(r.Context(), "no conversion request provided?")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	inputBytes, err := ioutil.ReadAll(r.Body)

	// This is mirrored from wh.ServeHttp (cf sigs.k8s.io/controller-runtime/pkg/webhook/conversion.go).
	if err != nil {
		dlog.Errorf(r.Context(), "could not read conversion request: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Go ahead and log the input...
	dlog.Debugf(r.Context(), "INPUT: %s", string(inputBytes))

	// ...then replace the request body with a new io.NopCloser that feeds back
	// the contents of the input buffer to the actual conversion method...
	r.Body = io.NopCloser(bytes.NewBuffer(inputBytes))

	// ...then use an httptest.ResponseRecorder to capture the output of the real
	// conversion method.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)

	// Log the output...
	dlog.Debugf(r.Context(), "OUTPUT: %s", rec.Body)

	// ...and then copy the recorded output to the real response.
	for k, v := range rec.Result().Header {
		w.Header()[k] = v
	}

	w.WriteHeader(rec.Code)

	// There's kind of nothing we can do if we can't write the body to w, so, uh... do
	// nothing? Should we panic instead??
	//nolint:errcheck
	rec.Body.WriteTo(w)
}

func ServeHTTPS(ctx context.Context, port int, ca *CA, scheme *k8sRuntime.Scheme) error {
	webhookHandler := conversion.NewWebhookHandler(scheme)

	mux := http.NewServeMux()

	mux.Handle(pathWebhooksCrdConvert, webhookHandler)

	dlog.Infof(ctx, "Serving HTTPS on port %d", port)

	// Assume that we'll use the conversion method directly, by using 'mux' for our
	// Handler...
	sc := &dhttp.ServerConfig{
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				return ca.GenServerCert(ctx, clientHello.ServerName)
			},
		},
	}

	// ...but if we're in debug mode, switch to using our conversionWithLogging handler
	// instead.
	if LogLevelIsAtLeastDebug() {
		sc.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conversionWithLogging(webhookHandler, w, r)
		})
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
