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
	PrivateKey       []byte
	CertificateChain []byte
	CACertChain      []byte
}

type Secrets struct {
	Secrets []*v3tlsconfig.Secret
}

// MakeSecrets takes all the Secrets in a snapshot and packages them up for
// consumption by ambex.
func MakeSecrets(ctx context.Context, k8sSnapshot *snapshotTypes.KubernetesSnapshot) []*v3tlsconfig.Secret {
	secrets := []*v3tlsconfig.Secret{}

	for _, secret := range k8sSnapshot.Secrets {
		name := secret.GetName()
		namespace := secret.GetNamespace()

		if namespace == "" {
			namespace = "default"
		}

		fullName := fmt.Sprintf("secret/%s/%s", namespace, name)

		var v3secret v3tlsconfig.Secret

		if secret.Type == v1.SecretTypeTLS {
			dlog.Warnf(ctx, "%s: TLS", fullName)

			privateKey := secret.Data["tls.key"]
			certificate := secret.Data["tls.crt"]

			v3secret = v3tlsconfig.Secret{
				Name: fullName,
				Type: &v3tlsconfig.Secret_TlsCertificate{
					TlsCertificate: &v3tlsconfig.TlsCertificate{
						PrivateKey: &v3core.DataSource{
							Specifier: &v3core.DataSource_InlineBytes{InlineBytes: privateKey},
						},
						CertificateChain: &v3core.DataSource{
							Specifier: &v3core.DataSource_InlineBytes{InlineBytes: certificate},
						},
					},
				},
			}
		} else if secret.Type == v1.SecretTypeOpaque {
			dlog.Warnf(ctx, "%s: Opaque", fullName)
			caCertificate := secret.Data["user.key"]

			v3secret = v3tlsconfig.Secret{
				Name: fullName,
				Type: &v3tlsconfig.Secret_ValidationContext{
					ValidationContext: &v3tlsconfig.CertificateValidationContext{
						TrustedCa: &v3core.DataSource{
							Specifier: &v3core.DataSource_InlineBytes{InlineBytes: caCertificate},
						},
					},
				},
			}
		} else {
			dlog.Errorf(ctx, "%s: unknown %s", fullName, secret.Type)
			continue
		}

		secrets = append(secrets, &v3secret)
	}

	return secrets
}
