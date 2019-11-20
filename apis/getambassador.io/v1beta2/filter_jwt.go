package v1

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/url"
	"text/template"

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

	InjectRequestHeaders []HeaderFieldTemplate `json:"injectRequestHeaders"`

	InsecureTLS       bool                     `json:"insecureTLS"`
	RawRenegotiateTLS string                   `json:"renegotiateTLS"`
	RenegotiateTLS    tls.RenegotiationSupport `json:"-"`

	ErrorResponse ErrorResponse `json:"errorResponse"`
}

type ErrorResponse struct {
	ContentType string                `json:"contentType"`
	Headers     []HeaderFieldTemplate `json:"headers"`

	RawBodyTemplate string             `json:"bodyTemplate"`
	BodyTemplate    *template.Template `json:"-"`
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

	for i := range m.InjectRequestHeaders {
		hf := &(m.InjectRequestHeaders[i])
		if err := hf.Validate(); err != nil {
			return errors.Wrap(err, "injectRequestHeaders")
		}
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

	if err := m.ErrorResponse.Validate(); err != nil {
		return errors.Wrap(err, "errorResponse")
	}

	return nil
}

func (er *ErrorResponse) Validate() error {
	// Handle deprecated .ContentType
	if er.ContentType != "" {
		er.Headers = append(er.Headers, HeaderFieldTemplate{
			Name:  "Content-Type",
			Value: er.ContentType,
		})
	}

	// Fill defaults
	if len(er.Headers) == 0 {
		er.Headers = append(er.Headers, HeaderFieldTemplate{
			Name:  "Content-Type",
			Value: "application/json",
		})
	}
	if er.RawBodyTemplate == "" {
		er.RawBodyTemplate = `{{ . | json "" }}`
	}

	// Parse+validate the header-field templates
	for i := range er.Headers {
		hf := &(er.Headers[i])
		if err := hf.Validate(); err != nil {
			return errors.Wrap(err, "headers")
		}
	}
	// Parse+validate the bodyTemplate
	tmpl, err := template.
		New("bodyTemplate").
		Funcs(template.FuncMap{
			"json": func(prefix string, data interface{}) (string, error) {
				nonIdentedJSON, err := json.Marshal(data)
				if err != nil {
					return "", err
				}
				var indentedJSON bytes.Buffer
				if err := json.Indent(&indentedJSON, nonIdentedJSON, prefix, "\t"); err != nil {
					return "", err
				}
				return indentedJSON.String(), nil
			},
		}).
		Parse(er.RawBodyTemplate)
	if err != nil {
		return errors.Wrap(err, "parsing template for bodyTemplate")
	}
	er.BodyTemplate = tmpl

	return nil
}
