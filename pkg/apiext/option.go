package apiext

// CAConfigOption allows overriding the default config options for `CACertManager`
type WebhookOption func(*WebhookServer)

// WithNamespace overrides the default namespace where the APIExt server is running
func WithNamespace(namespace string) WebhookOption {
	return func(s *WebhookServer) {
		s.namespace = namespace
	}
}

// WithHTTPSPort overrides the default port used by the WebConversionWebhook
func WithHTTPSPort(port int) WebhookOption {
	return func(s *WebhookServer) {
		s.httpsPort = port
	}
}

// WithHTTPPort overrides the default port used by the Healthz server
func WithHTTPPort(port int) WebhookOption {
	return func(s *WebhookServer) {
		s.httpPort = port
	}
}

// WithDisableCACertManagement disables the CA CertManager so that it will no longer
// create the root CA Cert and ensure it is valid.
//
// You should only disable this if you want to manage this externally using
// something like CertManager. The Webhook server still requires CA Secret
// to properly serve conversion webhook traffic.
func WithDisableCACertManagement() WebhookOption {
	return func(s *WebhookServer) {
		s.caMgmtEnabled = false
	}
}

// WithDisableCRDPatchManagement disable CRD Patching of CA Bundle, and allow
//
// You should only disable this if you want to manage this externally using
// something like CertManager. The Webhook server still requires the CRD
// to match the CA Secret to properly perform custom resource conversion.
func WithDisableCRDPatchManagement() WebhookOption {
	return func(s *WebhookServer) {
		s.crdPatchMgmtEnabled = false
	}
}

// WithCRDLabelSelectors provides a set of labels to use to limit what CRDs are
// watched and cached.
//
// By default, the "app.kubernetes.io/part-of": "emissary-apiext" label is
// used to filter the getambassador.io CRD's which in the default case is
// sufficient. However, if you modify or want to further limit the CRD
// Patcher then modified these label-selectors.
//
// Setting the label selectors to an empty map is effectively turning
// off selectors and the apiext server will watch all "getambassador.io" CRDs
func WithCRDLabelSelectors(selectors map[string]string) WebhookOption {
	return func(s *WebhookServer) {
		if selectors == nil {
			selectors = make(map[string]string)
		}
		s.crdLabelSelectors = selectors
	}
}
