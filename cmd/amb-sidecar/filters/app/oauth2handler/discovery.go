package oauth2handler

import (
	"bytes"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

const (
	certFMT = "-----BEGIN CERTIFICATE-----\n%v\n-----END CERTIFICATE-----"
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
	JSONWebKeySet         map[string]*JWK
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

	ret.JSONWebKeySet, err = fetchWebKeys(client, config.JSONWebKeySetURI)
	if err != nil {
		return nil, errors.Wrap(err, "discovery jwks_uri")
	}

	return &ret, nil
}

// JWK - JSON Web Key structure.
type JWK struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

// JWKSlice contains a collection of JSON WEB Keys.
type JWKSlice struct {
	Keys []JWK `json:"keys"`
}

// GetKey fetches the certificate from the IDP.  It returns an RSA
// public key or error if a problem occurs.
func (d *Discovered) GetKey(kid string, logger types.Logger) (*rsa.PublicKey, error) {
	log := logger.WithField("KeyID", kid)
	jwk := d.JSONWebKeySet[kid]
	if jwk == nil {
		return nil, errors.Errorf("JWK for KeyID=%q not found", kid)
	}
	// NOTE, plombardi@datawire.io: Multiple x5c entries?
	//
	// It seems there can be multiple entries in the x5c field (at least theoretically), but I haven't seen it or
	// run into it... so let's assume the first entry is valid and use that until something breaks.
	switch {
	case jwk.X5c != nil && len(jwk.X5c) >= 1:
		log.WithField("KeyFormat", "x509 certificate").Debug("JWK found")
		cert := fmt.Sprintf(certFMT, jwk.X5c[0])
		return jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
	case jwk.E != "" && jwk.N != "":
		log.WithField("KeyFormat", "public key").
			WithField("n", jwk.N).
			WithField("e", jwk.E).
			Debug("JWK found")

		rsaPubKey, err := assemblePubKeyFromNandE(jwk)
		if err != nil {
			return nil, err
		}
		return &rsaPubKey, nil
	default:
		return nil, errors.Errorf("JWK for KeyID=%q not found", kid)
	}
}

func fetchWebKeys(client *http.Client, jwksURI string) (map[string]*JWK, error) {
	resp, err := client.Get(jwksURI)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", jwksURI)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", jwksURI)
	}
	jwks := JWKSlice{}
	err = json.Unmarshal(body, &jwks)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", jwksURI)
	}

	ret := make(map[string]*JWK, len(jwks.Keys))
	for _, k := range jwks.Keys {
		ret[k.Kid] = &k
	}
	return ret, nil
}

func assemblePubKeyFromNandE(jwk *JWK) (rsa.PublicKey, error) {
	// Extract n and e values from the jwk
	nStr := jwk.N
	eStr := jwk.E

	key := rsa.PublicKey{}

	// Base64URL Decode the strings
	decN, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		fmt.Printf("Error %v\n", err)
		return key, err
	}
	n := big.NewInt(0)
	n.SetBytes(decN)

	decE, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		fmt.Printf("Error %v\n", err)
		return key, err
	}

	var eBytes []byte
	if len(decE) < 8 {
		eBytes = make([]byte, 8-len(decE), 8)
		eBytes = append(eBytes, decE...)
	} else {
		eBytes = decE
	}
	eReader := bytes.NewReader(eBytes)
	var e uint64
	err = binary.Read(eReader, binary.BigEndian, &e)
	if err != nil {
		fmt.Printf("Error %v\n", err)
		return key, err
	}

	return rsa.PublicKey{N: n, E: int(e)}, nil
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
