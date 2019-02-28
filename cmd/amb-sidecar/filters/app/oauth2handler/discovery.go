package oauth2handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/jwks"
)

type OpenIDConfig struct {
	// Issuer is signing authority for the tokens
	Issuer string `json:"issuer"`

	// Endpoint used to perform token authorization
	AuthorizationEndpoint string `json:"authorization_endpoint"`

	// Endpoint used to perform token authorization
	TokenEndpoint string `json:"token_endpoint"`

	// A set of public RSA keys used to sign the tokens
	JSONWebKeySetURI string `json:"jwks_uri"`
}

// Discovered stors the results (well, the subset of the results that
// we're interested in) from performing OIDC Discovery.
type Discovered struct {
	Issuer                string
	AuthorizationEndpoint *url.URL
	TokenEndpoint         *url.URL
	JSONWebKeySet         jwks.JWKS
}

// Discover fetches OpenID configuration and certificate information
// from the IDP (per OIDC Discovery).
func Discover(client *http.Client, mw crd.FilterOAuth2, logger types.Logger) (*Discovered, error) {
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

	ret.JSONWebKeySet, err = jwks.FetchJWKS(client, config.JSONWebKeySetURI)
	if err != nil {
		return nil, errors.Wrap(err, "discovery jwks_uri")
	}

	return &ret, nil
}

func fetchOpenIDConfig(client *http.Client, documentURL string) (OpenIDConfig, error) {
	config := OpenIDConfig{}

	res, err := client.Get(documentURL) // #nosec G107
	if err != nil {
		return config, errors.Wrap(err, "failed to fetch remote openid-configuration")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return config, errors.New("failed to fetch remote openid-configuration (status != 200)")
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return config, errors.Wrap(err, "failed to read openid-configuration HTTP response body")
	}

	err = json.Unmarshal(buf, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
