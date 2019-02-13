package discovery

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
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
	logger                types.Logger
}

var instance *Discovery

// New creates a singleton instance of the discovery client.
func New(cfg types.Config, logger types.Logger) (*Discovery, error) {
	config, err := fetchOpenIDConfig(cfg.AuthProviderURL + "/.well-known/openid-configuration")
	if err != nil {
		return nil, err
	}

	if instance == nil {
		instance = &Discovery{
			cache:  make(map[string]*JWK),
			mux:    &sync.RWMutex{},
			logger: logger,
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

	log := d.logger.WithField("KeyID", kid)
	if jwk := d.cache[kid]; jwk != nil {
		// NOTE, plombardi@datawire.io: Multiple x5c entries?
		//
		// It seems there can be multiple entries in the x5c field (at least theoretically), but I haven't seen it or
		// run into it... so let's assume the first entry is valid and use that until something breaks.
		//
		if jwk.X5c != nil && len(jwk.X5c) >= 1 {
			log.WithField("KeyFormat", "x509 certificate").Debug("JWK found")
			return fmt.Sprintf(certFMT, jwk.X5c[0])
		} else if jwk.E != "" && jwk.N != "" {
			log.WithField("KeyFormat", "public key").
				WithField("n", jwk.N).
				WithField("e", jwk.E).
				Debug("JWK found")

			rsaPubKey, err := assemblePubKeyFromNandE(jwk)
			pubKey, err := x509.MarshalPKIXPublicKey(&rsaPubKey)
			if err != nil {
				log.Error(err)
				return ""
			}

			var keyPEM = &pem.Block{
				Type:    "RSA PUBLIC KEY",
				Headers: make(map[string]string, 0),
				Bytes:   pubKey,
			}

			keyPEMString := string(pem.EncodeToMemory(keyPEM))
			return keyPEMString
		} else {
			log.Error("JWK does not have required 'x5c', or 'n' and 'e' values")
			return ""
		}
	} else {
		log.Error("JWK not found")
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
