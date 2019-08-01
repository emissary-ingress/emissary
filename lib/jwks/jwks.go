package jwks

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

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

const (
	certFMT = "-----BEGIN CERTIFICATE-----\n%v\n-----END CERTIFICATE-----"
)

// JWK - JSON Web Key structure.
type JWK struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

// jwkSlice contains a collection of JSON WEB Keys.
type jwkSlice struct {
	Keys []JWK `json:"keys"`
}

// JWKS - JWK Set
type JWKS map[string]JWK

func FetchJWKS(client *http.Client, jwksURI string) (JWKS, error) {
	resp, err := client.Get(jwksURI)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", jwksURI)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", jwksURI)
	}
	jwks := jwkSlice{}
	err = json.Unmarshal(body, &jwks)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", jwksURI)
	}

	ret := make(JWKS, len(jwks.Keys))
	for _, k := range jwks.Keys {
		ret[k.Kid] = k
	}
	return ret, nil
}

func (jwks JWKS) GetKey(kid string) (*rsa.PublicKey, error) {
	jwk, jwkOK := jwks[kid]
	if !jwkOK {
		return nil, errors.Errorf("KeyID=%q: JWK not found", kid)
	}
	// NOTE, plombardi@datawire.io: Multiple x5c entries?
	//
	// It seems there can be multiple entries in the x5c field (at least theoretically), but I haven't seen it or
	// run into it... so let's assume the first entry is valid and use that until something breaks.
	switch {
	case jwk.X5c != nil && len(jwk.X5c) >= 1:
		cert := fmt.Sprintf(certFMT, jwk.X5c[0])
		rsaPubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
		if err != nil {
			return nil, errors.Wrapf(err, "KeyID=%q: x509 certificate", kid)
		}
		return rsaPubKey, nil
	case jwk.E != "" && jwk.N != "":
		rsaPubKey, err := assemblePubKeyFromNandE(jwk)
		if err != nil {
			return nil, errors.Wrapf(err, "KeyID=%q: public key", kid)
		}
		return &rsaPubKey, nil
	default:
		return nil, errors.Errorf("KeyID=%q: invalid", kid)
	}
}

func assemblePubKeyFromNandE(jwk JWK) (rsa.PublicKey, error) {
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
