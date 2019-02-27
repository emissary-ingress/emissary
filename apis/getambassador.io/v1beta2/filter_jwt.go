package v1

import (
	"net/url"

	"github.com/pkg/errors"
)

// FilterJWT
//
// Currently supported algorithms:
//
// - RSA
//   * "RS256"
//   * "RS384"
//   * "RS512"
// - RSA-PSS
//   * "PS256"
//   * "PS384"
//   * "PS512"
// - ECDSA
//   * "ES256"
//   * "ES384"
//   * "ES512"
// - HMAC-SHA
//   * "HS256"
//   * "HS384"
//   * "HS512"
// - "none"
//
// This is this list of algos built-in to github.com/dgrijalva/jwt-go
// v3.2.0.  Keep this list in sync if we pull in a jwt-go update.
// More algorithms can be added with jwt.RegistersigningMethod().
//
// Haha, JK, our JWKS parser only understands RSA keys.
type FilterJWT struct {
	ValidAlgorithms     []string `json:"validAlgorithms"`
	RawJSONWebKeySetURI string   `json:"jwksURI"` // URI to a JWK Set (RFC 7517)
	JSONWebKeySetURI    *url.URL `json:"-"`

	Audience        string `json:"audience"`
	RequireAudience bool   `json:"requireAudience"`

	Issuer        string `json:"issuer"`
	RequireIssuer bool   `json:"requireIssuer"`

	RequireIssuedAt  bool `json:"requireIssuedAt"`
	RequireExpiresAt bool `json:"requireExpiresAt"`
	RequireNotBefore bool `json:"requireNotBefore"`
}

func (m *FilterJWT) Validate() error {
	if m.RawJSONWebKeySetURI == "" {
		if !(len(m.ValidAlgorithms) == 1 && m.ValidAlgorithms[0] == "none") {
			return errors.New("jwksURI is required unless validAlgorithms=[\"none\"]")
		}
	} else {
		u, err := url.Parse(m.RawJSONWebKeySetURI)
		if err != nil {
			return errors.Wrapf(err, "parsing jwksURI: %q", m.RawJSONWebKeySetURI)
		}
		if !u.IsAbs() {
			return errors.New("jwksURI is not an absolute URI")
		}
		m.JSONWebKeySetURI = u
	}

	return nil
}
