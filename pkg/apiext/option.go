package apiext

// CAConfigOption allows overriding the default config options for `CACertManager`
type WebhookOption func(*WebhookServer)

// WithNamespace overrides the default namespace where the APIExt server is running
func WithNamespace(namespace string) WebhookOption {
	return func(m *WebhookServer) {
		m.namespace = namespace
	}
}

// WithHTTPSPort overrides the default port used by the WebConversionWebhook
func WithHTTPSPort(port int) WebhookOption {
	return func(m *WebhookServer) {
		m.httpsPort = port
	}
}

// WithHTTPPort overrides the default port used by the Healthz server
func WithHTTPPort(port int) WebhookOption {
	return func(m *WebhookServer) {
		m.httpPort = port
	}
}

// WithDisableCACertManagement disables the CA CertManager so that it will no longer
// create the root CA Cert and ensure it is valid.
//
// You should only disable this if you want to manage this externally using
// something like CertManager. The Webhook server still requires CA Secret
// to properly serve conversion webhook traffic.
func WithDisableCACertManagement() WebhookOption {
	return func(m *WebhookServer) {
		m.caMgmtEnabled = false
	}
}

// WithDisableCRDPatchManagement disable CRD Patching of CA Bundle, and allow
//
// You should only disable this if you want to manage this externally using
// something like CertManager. The Webhook server still requires the CRD
// to match the CA Secret to properly perform custom resource conversion.
func WithDisableCRDPatchManagement() WebhookOption {
	return func(m *WebhookServer) {
		m.crdPatchMgmtEnabled = false
	}
}
