package v1

import (
	"net/url"

	"github.com/pkg/errors"
)

type TenantSpec struct {
	AmbassadorID AmbassadorID   `json:"ambassador_id"`
	Tenants      []TenantObject `json:"tenants"`
}

type TenantObject struct {
	RawTenantURL string   `json:"tenantUrl"`
	TenantURL    *url.URL `json:"-"` // is calculated from RawTenantURL
	Audience     string   `json:"audience"`
	ClientID     string   `json:"clientId"`
	Secret       string   `json:"secret"`
}

func (t *TenantObject) Validate() error {
	u, err := url.Parse(t.RawTenantURL)
	if err != nil {
		return errors.Wrap(err, "parsing tenant url")
	}
	if !u.IsAbs() {
		return errors.New("tenantUrl needs to be an absolute url: {scheme}://{host}:{port}")
	}
	t.TenantURL = u
	return nil
}

func (t TenantObject) CallbackURL() *url.URL {
	u, _ := t.TenantURL.Parse("/callback")
	return u
}

func (t TenantObject) TLS() bool {
	return t.TenantURL.Scheme == "https"
}

func (t TenantObject) Domain() string {
	return t.TenantURL.Host
}
