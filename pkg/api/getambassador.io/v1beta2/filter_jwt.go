package v1

import (
	"crypto/tls"
	"net/url"
	"time"

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

	RequireExpiresAt      bool          `json:"requireExpiresAt"`
	RawLeewayForExpiresAt string        `json:"leewayForExpiresAt"` // allow tokens expired by this much
	LeewayForExpiresAt    time.Duration `json:"-"`

	RequireNotBefore      bool          `json:"requireNotBefore"`
	RawLeewayForNotBefore string        `json:"leewayForNotBefore"` // allow tokens that shouldn't be used until this much in the future
	LeewayForNotBefore    time.Duration `json:"-"`

	RequireIssuedAt      bool          `json:"requireIssuedAt"`
	RawLeewayForIssuedAt string        `json:"leewayForIssuedAt"` // allow tokens issued this much in the future
	LeewayForIssuedAt    time.Duration `json:"-"`

	InjectRequestHeaders []HeaderFieldTemplate `json:"injectRequestHeaders"`

	InsecureTLS       bool                     `json:"insecureTLS"`
	RawRenegotiateTLS string                   `json:"renegotiateTLS"`
	RenegotiateTLS    tls.RenegotiationSupport `json:"-"`

	ErrorResponse JWTErrorResponse `json:"errorResponse"`
}

type JWTErrorResponse struct {
	Realm                 string `json:"realm"`
	DeprecatedContentType string `json:"contentType"`
	ErrorResponse
}

func (er *JWTErrorResponse) Validate(qname string) error {
	// Handle deprecated .ContentType
	if er.DeprecatedContentType != "" {
		er.Headers = append(er.Headers, HeaderFieldTemplate{
			Name:  "Content-Type",
			Value: er.DeprecatedContentType,
		})
	}

	// Fill defaults
	if er.Realm == "" {
		er.Realm = qname
	}

	return er.ErrorResponse.Validate()
}

func (m *FilterJWT) Validate(qname string) error {
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

	for i := range m.InjectRequestHeaders {
		hf := &(m.InjectRequestHeaders[i])
		if err := hf.Validate(); err != nil {
			return errors.Wrap(err, "injectRequestHeaders")
		}
	}

	if m.RawLeewayForExpiresAt != "" {
		d, err := time.ParseDuration(m.RawLeewayForExpiresAt)
		if err != nil {
			return errors.Wrapf(err, "parsing leewayForExpiresAt: %q", m.RawLeewayForExpiresAt)
		}
		m.LeewayForExpiresAt = d
	}
	if m.RawLeewayForNotBefore != "" {
		d, err := time.ParseDuration(m.RawLeewayForNotBefore)
		if err != nil {
			return errors.Wrapf(err, "parsing leewayForNotBefore: %q", m.RawLeewayForNotBefore)
		}
		m.LeewayForNotBefore = d
	}
	if m.RawLeewayForIssuedAt != "" {
		d, err := time.ParseDuration(m.RawLeewayForIssuedAt)
		if err != nil {
			return errors.Wrapf(err, "parsing leewayForIssuedAt: %q", m.RawLeewayForIssuedAt)
		}
		m.LeewayForIssuedAt = d
	}

	switch m.RawRenegotiateTLS {
	case "", "never":
		m.RenegotiateTLS = tls.RenegotiateNever
	case "onceAsClient":
		m.RenegotiateTLS = tls.RenegotiateOnceAsClient
	case "freelyAsClient":
		m.RenegotiateTLS = tls.RenegotiateFreelyAsClient
	default:
		return errors.Errorf("invalid renegotiateTLS: %q", m.RawRenegotiateTLS)
	}

	if err := m.ErrorResponse.Validate(qname); err != nil {
		return errors.Wrap(err, "errorResponse")
	}

	return nil
}

//////////////////////////////////////////////////////////////////////

type FilterJWTArguments struct {
	Scope []string `json:"scope"`
}
