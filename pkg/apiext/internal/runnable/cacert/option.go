package cacert

// CAConfigOption allows overriding the default config options for `CACertManager`
type Option func(*CACertManager)

// WithCASecretNamespace overrides the default namespace where the CA Cert Secret will be stored and watched
// Note: if changed you will need to ensure the permissions for the POD Service Account are allowed
// to watch, list, update and create in this namespace
func WithCASecretNamespace(namespace string) Option {
	return func(m *CACertManager) {
		m.secretNamespace = namespace
	}
}

// WithCASecretName overrides the default name used to store the CA Cert in a Secret.
// Note: if changed you will need to ensure the permissions for the POD Service Account are allowed
// to watch, list, update and create secrets for this resource.
func WithCASecretName(secretName string) Option {
	return func(m *CACertManager) {
		m.secretName = secretName
	}
}
