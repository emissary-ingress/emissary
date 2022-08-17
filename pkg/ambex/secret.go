package ambex

import (
	"context"
	"fmt"

	"github.com/datawire/dlib/dlog"
	v3core "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/config/core/v3"
	v3tlsconfig "github.com/emissary-ingress/emissary/v3/pkg/api/envoy/extensions/transport_sockets/tls/v3"
	snapshotTypes "github.com/emissary-ingress/emissary/v3/pkg/snapshot/v1"
	v1 "k8s.io/api/core/v1"
)

type Secret struct {
	Name             string
	Type             v1.SecretType
	PrivateKey       []byte
	CertificateChain []byte
	CACertChain      []byte
}

type Secrets struct {
	Secrets []*Secret
}

// MakeSecrets takes all the Secrets in a snapshot and packages them up for
// consumption by ambex.
func MakeSecrets(ctx context.Context, k8sSnapshot *snapshotTypes.KubernetesSnapshot) *Secrets {
	secrets := []*Secret{}

	for _, secret := range k8sSnapshot.Secrets {
		name := secret.GetName()
		namespace := secret.GetNamespace()

		if namespace == "" {
			namespace = "default"
		}

		fullName := fmt.Sprintf("secret/%s/%s", namespace, name)

		if secret.Type == v1.SecretTypeTLS {
			dlog.Warnf(ctx, "%s: TLS", fullName)

			privateKey := secret.Data["tls.key"]
			certificate := secret.Data["tls.crt"]

			secrets = append(secrets, &Secret{
				Name:             fullName,
				Type:             secret.Type,
				PrivateKey:       privateKey,
				CertificateChain: certificate,
			})
		} else if secret.Type == v1.SecretTypeOpaque {
			dlog.Warnf(ctx, "%s: Opaque", fullName)

			caCertificate := secret.Data["user.key"]

			secrets = append(secrets, &Secret{
				Name:        fullName,
				Type:        secret.Type,
				CACertChain: caCertificate,
			})
		}
	}

	return &Secrets{Secrets: secrets}
}

func (secrets *Secrets) ToV3List(ctx context.Context) []*v3tlsconfig.Secret {
	v3secrets := make([]*v3tlsconfig.Secret, 0, len(secrets.Secrets))

	for _, secret := range secrets.Secrets {
		var v3secret v3tlsconfig.Secret

		if secret.Type == v1.SecretTypeTLS {
			dlog.Warnf(ctx, "%s: TLS", secret.Name)

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
			dlog.Warnf(ctx, "%s: Opaque", secret.Name)

			v3secret = v3tlsconfig.Secret{
				Name: secret.Name,
				Type: &v3tlsconfig.Secret_ValidationContext{
					ValidationContext: &v3tlsconfig.CertificateValidationContext{
						TrustedCa: &v3core.DataSource{
							Specifier: &v3core.DataSource_InlineBytes{
								InlineBytes: secret.CACertChain,
							},
						},
					},
				},
			}
		} else {
			dlog.Errorf(ctx, "%s: unknown %s", secret.Name, secret.Type)
			continue
		}

		v3secrets = append(v3secrets, &v3secret)
	}

	return v3secrets
}
