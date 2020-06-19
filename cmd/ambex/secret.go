package ambex

// Most of the following code for creation of go-control-plane secret resources happen
// here and it has been taken from go-control-plane code base.

import (
	auth "github.com/datawire/ambassador/pkg/api/envoy/api/v2/auth"
	core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/cache/types"
)

func GetCertificateChain(secrets *Secrets) types.Resource {
	return &auth.Secret{
		Name: "tls." + secrets.Name,
		Type: &auth.Secret_TlsCertificate{
			TlsCertificate: &auth.TlsCertificate{
				PrivateKey: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(secrets.Data.TLSKey)},
				},
				CertificateChain: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(secrets.Data.TLSCert)},
				},
			},
		},
	}
}

func GetTrustedCA(secrets *Secrets) types.Resource {
	return &auth.Secret{
		Name: "root." + secrets.Name,
		Type: &auth.Secret_ValidationContext{
			ValidationContext: &auth.CertificateValidationContext{
				TrustedCa: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{InlineBytes: []byte(secrets.Data.TLSCert)},
				},
			},
		},
	}
}
