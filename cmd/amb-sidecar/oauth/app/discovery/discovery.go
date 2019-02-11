package discovery

import (
	"bytes"
	"crypto/rsa"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"io/ioutil"
	"math/big"
	"net/http"
	"sync"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

const (
	certFMT = "-----BEGIN CERTIFICATE-----\n%v\n-----END CERTIFICATE-----"
)

type openIDConfig struct {
	// Issuer is signing authority for the tokens
	Issuer string `json:"issuer"`

	// Endpoint used to perform token authorization
	AuthorizationEndpoint string `json:"authorization_endpoint"`

	// Endpoint used to perform token authorization
	TokenEndpoint string `json:"token_endpoint"`

	// A set of public RSA keys used to sign the tokens
	JSONWebKeyURI string `json:"jwks_uri"`
}

// Discovery is used to fetch the certificate information from the IDP.
type Discovery struct {
	Issuer                string
	AuthorizationEndpoint string
	TokenEndpoint         string
	JSONWebKeysURI        string
	cache                 map[string]*JWK
	mux                   *sync.RWMutex
}

var instance *Discovery

// New creates a singleton instance of the discovery client.
func New(cfg types.Config) (*Discovery, error) {
	config, err := fetchOpenIDConfig(cfg.AuthProviderURL + "/.well-known/openid-configuration")
	if err != nil {
		return nil, err
	}

	if instance == nil {
		instance = &Discovery{
			cache: make(map[string]*JWK),
			mux:   &sync.RWMutex{},
		}

		instance.Issuer = config.Issuer
		instance.AuthorizationEndpoint = config.AuthorizationEndpoint
		instance.TokenEndpoint = config.TokenEndpoint
		instance.JSONWebKeysURI = config.JSONWebKeyURI
	}

	err = instance.fetchWebKeys()
	if err != nil {
		return nil, err
	}

	return instance, nil
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

// GetPemCert fetches the certificate from the IDP. It returns a cert string or
// error if a problem occurs.
func (d *Discovery) GetPemCert(kid string) (string, error) {
	if cert := d.getCert(kid); cert != "" {
		return cert, nil
	}

	if err := d.fetchWebKeys(); err != nil {
		return "", err
	}

	if cert := d.getCert(kid); cert != "" {
		return cert, nil
	}

	return "", errors.New("certificate not found")
}

func (d *Discovery) getCert(kid string) string {
	d.mux.RLock()
	defer d.mux.RUnlock()
	if jwk := d.cache[kid]; jwk != nil {
		if jwk.X5c != nil {
			return fmt.Sprintf(certFMT, jwk.X5c)
		} else if jwk.E != "" && jwk.N != "" {
			pubKey, err := assemblePubKeyFromNandE(jwk)

			fmt.Println(pubKey.Size())
			spew.Dump(pubKey)

			if err != nil {
				return "" // TODO: err?
			}

			asn1Bytes, err := asn1.Marshal(pubKey)
			var pemkey = &pem.Block{
				Type:  "RSA PUBLIC KEY",
				Bytes: asn1Bytes,
			}

			key := pem.EncodeToMemory(pemkey)
			fmt.Println(string(key))
			return string(key)
		} else {
			return "" // TODO: err?
		}
	}
	return ""
}

func (d *Discovery) fetchWebKeys() error {
	resp, err := http.Get(d.JSONWebKeysURI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	jwks := JWKSlice{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)
	if err != nil {
		return err
	}

	d.mux.Lock()
	defer d.mux.Unlock()
	for _, k := range jwks.Keys {
		d.cache[k.Kid] = &k
	}

	return nil
}

func assemblePubKeyFromNandE(jwk *JWK) (rsa.PublicKey, error) {
	// Extract n and e values from the jwk
	nStr := jwk.N
	eStr := jwk.E

	key := rsa.PublicKey{}

	// Base64URL Decode the strings
	decN, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return key, err
	}
	n := big.NewInt(0)
	n.SetBytes(decN)

	decE, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
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
		return key, err
	}

	return rsa.PublicKey{N: n, E: int(e)}, nil
}

func fetchOpenIDConfig(documentURL string) (openIDConfig, error) {
	config := openIDConfig{}

	res, err := http.Get(documentURL)
	if err != nil {

		return config, errors.Wrap(err, "failed to fetch remote openid-configuration")
	}

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
