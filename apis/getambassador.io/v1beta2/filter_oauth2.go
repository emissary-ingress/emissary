package v1

import (
	"net/url"
	"time"

	"github.com/pkg/errors"
)

type FilterOAuth2 struct {
	RawAuthorizationURL string   `json:"authorizationURL"` // formerly AUTH_PROVIDER_URL
	AuthorizationURL    *url.URL `json:"-"`                // calculated from RawAuthorizationURL
	RawClientURL        string   `json:"clientURL"`        // formerly tenant.tenantUrl
	ClientURL           *url.URL `json:"-"`                // calculated from RawClientURL

	RawStateTTL string        `json:"stateTTL"`
	StateTTL    time.Duration `json:"-"` // calculated from RawStateTTL
	Audience    string        `json:"audience"`
	ClientID    string        `json:"clientID"`
	Secret      string        `json:"secret"`

	RawMaxStale string        `json:"maxStale"`
	MaxStale    time.Duration `json:"-"` // calculated from RawMaxStale

	InsecureTLS bool `json:"insecureTLS"`
}

func (m *FilterOAuth2) Validate() error {
	u, err := url.Parse(m.RawAuthorizationURL)
	if err != nil {
		return errors.Wrapf(err, "parsing authorizationURL: %q", m.RawAuthorizationURL)
	}
	if !u.IsAbs() {
		return errors.New("authorizationURL is not an absolute URL")
	}
	m.AuthorizationURL = u

	u, err = url.Parse(m.RawClientURL)
	if err != nil {
		return errors.Wrapf(err, "parsing clientURL: %q", m.RawClientURL)
	}
	if !u.IsAbs() {
		return errors.New("clientURL is not an absolute URL")
	}
	m.ClientURL = u

	if m.RawStateTTL == "" {
		m.StateTTL = 5 * time.Minute
	} else {
		d, err := time.ParseDuration(m.RawStateTTL)
		if err != nil {
			return errors.Wrapf(err, "parsing stateTTL: %q", m.RawStateTTL)
		}
		m.StateTTL = d
	}

	if m.RawMaxStale != "" {
		d, err := time.ParseDuration(m.RawMaxStale)
		if err != nil {
			return errors.Wrapf(err, "parsing maxStale: %q", m.RawMaxStale)
		}
		m.MaxStale = d
	}

	return nil
}

func (m FilterOAuth2) CallbackURL() *url.URL {
	u, _ := m.ClientURL.Parse("/callback")
	return u
}

func (m FilterOAuth2) Domain() string {
	return m.ClientURL.Host
}

func (m FilterOAuth2) TLS() bool {
	return m.ClientURL.Scheme == "https"
}

//////////////////////////////////////////////////////////////////////

type FilterOAuth2Arguments struct {
	Scopes []string `json:"scopes"`
}
