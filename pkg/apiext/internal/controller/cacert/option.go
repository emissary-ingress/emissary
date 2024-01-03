package cacert

import "k8s.io/apimachinery/pkg/types"

// Option allows overriding the default config options for `CACertManager`
type Option func(*caCertController)

// WithCASecreSettings overrides the default secret used to store the root CA
func WithCASecretSettings(secretSettings types.NamespacedName) Option {
	return func(c *caCertController) {
		c.secretSettings = secretSettings
	}
}
