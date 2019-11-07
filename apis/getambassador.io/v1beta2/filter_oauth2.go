package v1

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

type FilterOAuth2 struct {
	RawAuthorizationURL string   `json:"authorizationURL"` // formerly AUTH_PROVIDER_URL
	AuthorizationURL    *url.URL `json:"-"`                // calculated from RawAuthorizationURL
	RawClientURL        string   `json:"clientURL"`        // formerly tenant.tenantUrl
	ClientURL           *url.URL `json:"-"`                // calculated from RawClientURL

	RawStateTTL     string        `json:"stateTTL"`
	StateTTL        time.Duration `json:"-"` // calculated from RawStateTTL
	Audience        string        `json:"audience"`
	ClientID        string        `json:"clientID"`
	Secret          string        `json:"secret"`
	SecretName      string        `json:"secretName"`
	SecretNamespace string        `json:"secretNamespace"`

	RawMaxStale string        `json:"maxStale"`
	MaxStale    time.Duration `json:"-"` // calculated from RawMaxStale

	InsecureTLS       bool                     `json:"insecureTLS"`
	RawRenegotiateTLS string                   `json:"renegotiateTLS"`
	RenegotiateTLS    tls.RenegotiationSupport `json:"-"`

	AccessTokenValidation string `json:"accessTokenValidation"`
}

func (m *FilterOAuth2) Validate(namespace string, secretsGetter coreV1client.SecretsGetter) error {
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

	if m.SecretName != "" {
		if m.Secret != "" {
			return errors.New("it is invalid to set both 'secret' and 'secretName'")
		}
		if m.SecretNamespace == "" {
			m.SecretNamespace = namespace
		}
		secret, err := secretsGetter.Secrets(m.SecretNamespace).Get(m.SecretName, metaV1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "getting secret name=%q namespace=%q", m.SecretName, m.SecretNamespace)
		}
		secretVal, ok := secret.Data["oauth2-client-secret"]
		if !ok {
			return errors.Errorf("secret name=%q namespace=%q does not contain an oauth2-client-secret field", m.SecretName, m.SecretNamespace)
		}
		m.Secret = string(secretVal)
	}

	switch m.AccessTokenValidation {
	case "":
		m.AccessTokenValidation = "auto"
	case "auto", "jwt", "userinfo":
		// do nothing
	default:
		return errors.Errorf("accessTokenValidation=%q is invalid; valid values are %q",
			m.AccessTokenValidation, []string{"auto", "jwt", "userinfo"})
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
	Scopes            []string        `json:"scopes"`
	InsteadOfRedirect *OAuth2Redirect `json:"insteadOfRedirect"`
}

type OAuth2Redirect struct {
	HTTPStatusCode  int                 `json:"httpStatusCode"`
	IfRequestHeader HeaderFieldSelector `json:"ifRequestHeader"`
}

func (m *FilterOAuth2Arguments) Validate() error {
	if m.InsteadOfRedirect != nil && m.InsteadOfRedirect.HTTPStatusCode == 0 {
		// The default is 403 Forbidden, and definitely not
		// 401 Unauthorized, because the User Agent is not
		// using an RFC 7235-compatible authentication scheme
		// to talk with us; 401 would be inappropriate.
		m.InsteadOfRedirect.HTTPStatusCode = http.StatusForbidden

		if err := m.InsteadOfRedirect.IfRequestHeader.Validate(); err != nil {
			err = errors.Wrap(err, "ifRequestHeader")
			err = errors.Wrap(err, "insteadOfRedirect")
			return err
		}
	}

	return nil
}
