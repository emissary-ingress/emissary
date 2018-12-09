package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
)

const (
	certFMT = "-----BEGIN CERTIFICATE-----\n%v\n-----END CERTIFICATE-----"
)

// Discovery ...
type Discovery struct {
	url   string
	cache map[string]*JWK
	mux   *sync.RWMutex
}

var instance *Discovery

// New ...
func New(cfg *config.Config) *Discovery {
	if instance == nil {
		instance = &Discovery{
			cache: make(map[string]*JWK),
			mux:   &sync.RWMutex{},
		}
		if cfg.Secure {
			instance.url = fmt.Sprintf("https://%s/.well-known/jwks.json", cfg.Domain)
		} else {
			instance.url = fmt.Sprintf("http://%s/.well-known/jwks.json", cfg.Domain)
		}
	}
	instance.fetchWebKeys()
	return instance
}

// JWK - JSON Web Key data structure
type JWK struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

// JWKSlice TODO(gsagula): comment
type JWKSlice struct {
	Keys []JWK `json:"keys"`
}

// GetPemCert TODO(gsagula): comment
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
		return fmt.Sprintf(certFMT, jwk.X5c[0])
	}
	return ""
}

func (d *Discovery) fetchWebKeys() error {
	resp, err := http.Get(d.url)
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
