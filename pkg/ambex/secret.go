package ambex

import (
	"context"
	"fmt"

	v2auth "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/auth"
	v2core "github.com/datawire/ambassador/v2/pkg/api/envoy/api/v2/core"
	v3core "github.com/datawire/ambassador/v2/pkg/api/envoy/config/core/v3"
	v3tlsconfig "github.com/datawire/ambassador/v2/pkg/api/envoy/extensions/transport_sockets/tls/v3"
	"github.com/datawire/dlib/dlog"
	v1 "k8s.io/api/core/v1"
)

type Secret struct {
	Name             string
	Type             v1.SecretType
	PrivateKey       []byte
	CertificateChain []byte
	CACertChain      []byte
	Crl              []byte
}

type Secrets struct {
	TlsSecrets        []*Secret
	ValidationSecrets []*Secret
}

func (secrets *Secrets) ToV2List(ctx context.Context, validationGroups [][]string) []*v2auth.Secret {
	v2secrets := make([]*v2auth.Secret, 0, (len(secrets.TlsSecrets) + len(secrets.ValidationSecrets)))

	// Iterate over the validation groups and create a validation context for each
	// There is room for performance improvements here
	for _, vGroup := range validationGroups {
		vContext := &v2auth.CertificateValidationContext{}
		crlName, caName := "", "" // names of the CA and CRL secrets that are needed later to combine into the final shared name

		dlog.Debugf(ctx, "[V2Secrets] building validation_context for secret group: %v\n", vGroup)

		for _, secret := range secrets.ValidationSecrets {
			if containsElem(vGroup, secret.Name) {
				if secret.Type == v1.SecretTypeTLS {
					// If it is a Tls secret for validation_context then it has to be a CACert
					vContext.TrustedCa = &v2core.DataSource{
						Specifier: &v2core.DataSource_InlineBytes{
							InlineBytes: secret.CertificateChain,
						},
					}
					caName = secret.Name
				} else if secret.Type == v1.SecretTypeOpaque {
					// If it is Opaque then its either a CACert or a Crl
					if caCert := secret.CACertChain; len(caCert) != 0 {
						vContext.TrustedCa = &v2core.DataSource{
							Specifier: &v2core.DataSource_InlineBytes{
								InlineBytes: secret.CACertChain,
							},
						}
						caName = secret.Name
					}
					if crl := secret.Crl; len(crl) != 0 {
						vContext.Crl = &v2core.DataSource{
							Specifier: &v2core.DataSource_InlineBytes{
								InlineBytes: secret.Crl,
							},
						}
						crlName = secret.Name
					}
				} else {
					dlog.Errorf(ctx, "[V2Secrets] %s: unknown secret type: %s", secret.Name, secret.Type)
					continue
				}
			}
		}

		// Make a name for the new validation_context built from one or more secrets
		// Default to joining their names. The name of the CA secret will always be first
		groupName := ""
		if caName != "" && crlName != "" {
			groupName = fmt.Sprintf("%s-%s", caName, crlName)
		} else if caName != "" && crlName == "" {
			groupName = caName
		} else {
			groupName = crlName
		}

		dlog.Debugf(ctx, "[V2Secrets] built validation_context Group: %v", groupName)
		v2secrets = append(v2secrets, &v2auth.Secret{
			Name: groupName,
			Type: &v2auth.Secret_ValidationContext{
				ValidationContext: vContext,
			},
		})

	}

	// Do the same for tls secrets
	for _, secret := range secrets.TlsSecrets {
		var v2secret v2auth.Secret

		if secret.Type == v1.SecretTypeTLS {
			v2secret = v2auth.Secret{
				Name: secret.Name,
				Type: &v2auth.Secret_TlsCertificate{
					TlsCertificate: &v2auth.TlsCertificate{
						PrivateKey: &v2core.DataSource{
							Specifier: &v2core.DataSource_InlineBytes{
								InlineBytes: secret.PrivateKey,
							},
						},
						CertificateChain: &v2core.DataSource{
							Specifier: &v2core.DataSource_InlineBytes{
								InlineBytes: secret.CertificateChain,
							},
						},
					},
				},
			}
		} else if secret.Type == v1.SecretTypeOpaque {
			dlog.Errorf(ctx, "[V2Secrets] %s: Opaque TLS secret cannot be used", secret.Name)
		} else {
			dlog.Errorf(ctx, "[V2Secrets] %s: unknown secret type: %s", secret.Name, secret.Type)
			continue
		}

		v2secrets = append(v2secrets, &v2secret)
	}

	return v2secrets
}

func (secrets *Secrets) ToV3List(ctx context.Context, validationGroups [][]string) []*v3tlsconfig.Secret {

	v3secrets := make([]*v3tlsconfig.Secret, 0, (len(secrets.TlsSecrets) + len(secrets.ValidationSecrets)))

	// Iterate over the validation groups and create a validation context for each
	// There is room for performance improvements here
	for _, vGroup := range validationGroups {
		vContext := &v3tlsconfig.CertificateValidationContext{}
		crlName, caName := "", "" // names of the CA and CRL secrets that are needed later to combine into the final shared name

		dlog.Debugf(ctx, "[V3Secrets] building validation_context for secret group: %v\n", vGroup)

		for _, secret := range secrets.ValidationSecrets {
			if containsElem(vGroup, secret.Name) {
				if secret.Type == v1.SecretTypeTLS {
					// If it is a Tls secret for validation_context then it has to be a CACert
					vContext.TrustedCa = &v3core.DataSource{
						Specifier: &v3core.DataSource_InlineBytes{
							InlineBytes: secret.CertificateChain,
						},
					}
					caName = secret.Name
				} else if secret.Type == v1.SecretTypeOpaque {
					// If it is Opaque then its either a CACert or a Crl
					if caCert := secret.CACertChain; len(caCert) != 0 {
						vContext.TrustedCa = &v3core.DataSource{
							Specifier: &v3core.DataSource_InlineBytes{
								InlineBytes: secret.CACertChain,
							},
						}
						caName = secret.Name
					}
					if crl := secret.Crl; len(crl) != 0 {
						vContext.Crl = &v3core.DataSource{
							Specifier: &v3core.DataSource_InlineBytes{
								InlineBytes: secret.Crl,
							},
						}
						crlName = secret.Name
					}
				} else {
					dlog.Errorf(ctx, "[V3Secrets] %s: unknown secret type: %s", secret.Name, secret.Type)
					continue
				}
			}
		}

		// Make a name for the new validation_context built from one or more secrets
		// Default to joining their names. The name of the CA secret will always be first
		groupName := ""
		if caName != "" && crlName != "" {
			groupName = fmt.Sprintf("%s-%s", caName, crlName)
		} else if caName != "" && crlName == "" {
			groupName = caName
		} else {
			groupName = crlName
		}

		dlog.Debugf(ctx, "[V3Secrets] built validation_context Group: %v", groupName)

		v3secrets = append(v3secrets, &v3tlsconfig.Secret{
			Name: groupName,
			Type: &v3tlsconfig.Secret_ValidationContext{
				ValidationContext: vContext,
			},
		})

	}

	// Do the same for tls secrets
	for _, secret := range secrets.TlsSecrets {
		var v3secret v3tlsconfig.Secret

		if secret.Type == v1.SecretTypeTLS {
			v3secret = v3tlsconfig.Secret{
				Name: secret.Name,
				Type: &v3tlsconfig.Secret_TlsCertificate{
					TlsCertificate: &v3tlsconfig.TlsCertificate{
						PrivateKey: &v3core.DataSource{
							Specifier: &v3core.DataSource_InlineBytes{
								InlineBytes: secret.PrivateKey,
							},
						},
						CertificateChain: &v3core.DataSource{
							Specifier: &v3core.DataSource_InlineBytes{
								InlineBytes: secret.CertificateChain,
							},
						},
					},
				},
			}
		} else if secret.Type == v1.SecretTypeOpaque {
			dlog.Errorf(ctx, "[V3Secrets] %s: Opaque TLS secret cannot be used", secret.Name)
		} else {
			dlog.Errorf(ctx, "[V3Secrets] %s: unknown secret type: %s", secret.Name, secret.Type)
			continue
		}

		v3secrets = append(v3secrets, &v3secret)
	}

	return v3secrets

}

// Just checks if the list contains the provided string
// I really wish go just had this builtin....
func containsElem(s []string, key string) bool {
	for _, e := range s {
		if e == key {
			return true
		}
	}
	return false
}
