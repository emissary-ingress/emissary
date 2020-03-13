package discovery

import (
	"net/http"
	"net/url"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/lib/jwks"
	"github.com/datawire/apro/lib/util"
)

type OpenIDConfig struct {
	// Issuer is signing authority for the tokens
	Issuer string `json:"issuer"`

	// Endpoint used to perform token authorization
	AuthorizationEndpoint string `json:"authorization_endpoint"`

	// Endpoint used to perform token authorization
	TokenEndpoint string `json:"token_endpoint"`

	// Endpoint used to look up user info (which can be used to
	// validate opaque access tokens).
	UserInfoEndpoint string `json:"userinfo_endpoint"`

	EndSessionEndpoint string `json:"end_session_endpoint"`

	// A set of public RSA keys used to sign the tokens
	JSONWebKeySetURI string `json:"jwks_uri"`
}

// Discovered stores the results (well, the subset of the results that
// we're interested in) from performing OIDC Discovery.
type Discovered struct {
	Issuer                string
	AuthorizationEndpoint *url.URL
	TokenEndpoint         *url.URL
	UserInfoEndpoint      *url.URL
	EndSessionEndpoint    *url.URL
	JSONWebKeySet         jwks.JWKS
}

// Discover fetches OpenID configuration and certificate information
// from the IDP (per OIDC Discovery).
func Discover(client *http.Client, mw crd.FilterOAuth2, logger dlog.Logger) (*Discovered, error) {
	configURL, _ := url.Parse(mw.AuthorizationURL.String() + "/.well-known/openid-configuration")
	config, err := fetchOpenIDConfig(client, configURL.String())
	if err != nil {
		return nil, errors.Wrapf(err, "fetchOpenIDConfig(%q)", configURL)
	}

	var ret Discovered

	ret.Issuer = config.Issuer

	ret.AuthorizationEndpoint, err = url.Parse(config.AuthorizationEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "discovery authorization_endpoint")
	}

	ret.TokenEndpoint, err = url.Parse(config.TokenEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "discovery token_endpoint")
	}

	ret.UserInfoEndpoint, err = url.Parse(config.UserInfoEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "discovery userinfo_endpoint")
	}

	if config.EndSessionEndpoint != "" {
		ret.EndSessionEndpoint, err = url.Parse(config.EndSessionEndpoint)
		if err != nil {
			return nil, errors.Wrap(err, "discovery end_session_endpoint")
		}
	}

	ret.JSONWebKeySet, err = jwks.FetchJWKS(client, config.JSONWebKeySetURI)
	if err != nil {
		return nil, errors.Wrap(err, "discovery jwks_uri")
	}

	return &ret, nil
}

func fetchOpenIDConfig(client *http.Client, documentURL string) (OpenIDConfig, error) {
	config := OpenIDConfig{}

	sclient := &util.SimpleClient{Client: client}

	err := sclient.GetBodyJSON(documentURL, &config) // -#-n-o-s-ec G107
	if err != nil {
		// XXX: why return config here
		return config, errors.Wrap(err, "failed to fetch remote openid-configuration")
	}
	return config, nil
}
