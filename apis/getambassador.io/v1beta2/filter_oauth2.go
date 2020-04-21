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

const (
	GrantType_AuthorizationCode = "AuthorizationCode"
	GrantType_ClientCredentials = "ClientCredentials"
	GrantType_Password          = "Password"
)

type FilterOAuth2 struct {
	RawAuthorizationURL string   `json:"authorizationURL"` // formerly AUTH_PROVIDER_URL
	AuthorizationURL    *url.URL `json:"-"`                // calculated from RawAuthorizationURL

	GrantType string `json:"grantType"`

	// grantType=AuthorizationCode
	DeprecatedClientURL string             `json:"clientURL"` // formerly tenant.tenantUrl
	DeprecatedStateTTL  string             `json:"stateTTL"`
	ClientID            string             `json:"clientID"`
	Secret              string             `json:"secret"`
	SecretName          string             `json:"secretName"`
	SecretNamespace     string             `json:"secretNamespace"`
	UseSessionCookies   *UseSessionCookies `json:"useSessionCookies"`
	ProtectedOrigins    []Origin           `json:"protectedOrigins"`

	RawMaxStale string        `json:"maxStale"`
	MaxStale    time.Duration `json:"-"` // calculated from RawMaxStale

	InsecureTLS       bool                     `json:"insecureTLS"`
	RawRenegotiateTLS string                   `json:"renegotiateTLS"`
	RenegotiateTLS    tls.RenegotiationSupport `json:"-"`

	ExtraAuthorizationParameters map[string]string `json:"extraAuthorizationParameters"`

	AccessTokenValidation string             `json:"accessTokenValidation"`
	AccessTokenJWTFilter  JWTFilterReference `json:"accessTokenJWTFilter"`
}

type UseSessionCookies struct {
	Value           *bool               `json:"value"`
	IfRequestHeader HeaderFieldSelector `json:"ifRequestHeader"`
}

func (m *UseSessionCookies) Validate() error {
	if m.Value == nil {
		value := true
		m.Value = &value
	}
	return m.IfRequestHeader.Validate()
}

type JWTFilterReference struct {
	Name      string             `json:"name"`
	Namespace string             `json:"namespace"`
	Arguments FilterJWTArguments `json:"arguments"`
}

type Origin struct {
	RawOrigin         string   `json:"origin"`
	Origin            *url.URL `json:"-"` // RawOrigin
	IncludeSubdomains bool     `json:"includeSubdomains"`
}

func (m *Origin) Validate() error {
	u, err := url.Parse(m.RawOrigin)
	if err != nil {
		return errors.Wrapf(err, "parsing origin: %q", m.RawOrigin)
	}
	if !u.IsAbs() {
		return errors.New("origin is not an absolute URL")
	}
	m.Origin = u
	return nil
}

//nolint:gocyclo
func (m *FilterOAuth2) Validate(namespace string, secretsGetter coreV1client.SecretsGetter) error {
	// Convert deprecated fields
	if m.DeprecatedClientURL != "" {
		if len(m.ProtectedOrigins) > 0 {
			return errors.New("it is invalid to set both \"clientURL\" and \"protectedOrigins\"; \"clientURL\" is deprecated and should be replaced by \"protectedOrigins\"")
		}
		m.ProtectedOrigins = []Origin{
			{RawOrigin: m.DeprecatedClientURL},
		}
		m.DeprecatedClientURL = ""
	}

	// Main
	u, err := url.Parse(m.RawAuthorizationURL)
	if err != nil {
		return errors.Wrapf(err, "parsing authorizationURL: %q", m.RawAuthorizationURL)
	}
	if !u.IsAbs() {
		return errors.New("authorizationURL is not an absolute URL")
	}
	m.AuthorizationURL = u

	if m.GrantType == "" {
		m.GrantType = GrantType_AuthorizationCode
	}
	switch m.GrantType {
	case GrantType_AuthorizationCode:
		if len(m.ProtectedOrigins) < 1 {
			return errors.New("must have at least one 'protectedOrigin' when 'grantType==AuthorizationCode'")
		}
		for i := range m.ProtectedOrigins {
			origin := &(m.ProtectedOrigins[i])
			if err := origin.Validate(); err != nil {
				return errors.Wrapf(err, "protectedOrigins[%d]", i)
			}
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

		if m.UseSessionCookies == nil {
			value := false
			m.UseSessionCookies = &UseSessionCookies{
				Value: &value,
			}
		}
		if err := m.UseSessionCookies.Validate(); err != nil {
			return errors.Wrap(err, "useSessionCookies")
		}
	case GrantType_ClientCredentials:
		if m.DeprecatedClientURL != "" {
			return errors.New("it is invalid to set 'clientURL' when 'grantType==ClientCredentials'")
		}
		if len(m.ProtectedOrigins) > 0 {
			return errors.New("it is invalid to set 'protectedOrigins' when 'grantType==ClientCredentials'")
		}
		if m.ClientID != "" {
			return errors.New("it is invalid to set 'clientID' when 'grantType==ClientCredentials'")
		}
		if m.Secret != "" {
			return errors.New("it is invalid to set 'secret' when 'grantType==ClientCredentials'")
		}
		if m.SecretName != "" {
			return errors.New("it is invalid to set 'secretName' when 'grantType==ClientCredentials'")
		}
		if m.SecretNamespace != "" {
			return errors.New("it is invalid to set 'secretNamespace' when 'grantType==ClientCredentials'")
		}

	case GrantType_Password:
		if m.DeprecatedClientURL != "" {
			return errors.New("it is invalid to set 'clientURL' when 'grantType==HeaderCrentials'")
		}
		if len(m.ProtectedOrigins) > 0 {
			return errors.New("it is invalid to set 'protectedOrigins' when 'grantType==HeaderCrentials'")
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

	default:
		return errors.Errorf("grantType=%q is invalid; valid values are %q",
			m.GrantType,
			[]string{GrantType_AuthorizationCode,
				GrantType_ClientCredentials,
				GrantType_Password})
	}

	if m.RawMaxStale != "" {
		d, err := time.ParseDuration(m.RawMaxStale)
		if err != nil {
			return errors.Wrapf(err, "parsing maxStale: %q", m.RawMaxStale)
		}
		m.MaxStale = d
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

	for key := range m.ExtraAuthorizationParameters {
		_, conflict := map[string]struct{}{
			"response_type": {},
			"client_id":     {},
			"redirect_uri":  {},
			"scope":         {},
			"state":         {},
		}[key]
		if conflict {
			return errors.Errorf("extraAuthorizationParameters: may not manually specify built-in OAuth parameter %q", key)
		}
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

	if m.AccessTokenJWTFilter.Name != "" {
		switch m.AccessTokenValidation {
		case "auto":
			m.AccessTokenValidation = "jwt"
		case "jwt":
			// do nothing
		case "userinfo":
			return errors.Errorf("accessTokenValidation=%q does not do JWT validation, but accessTokenJWTFilter is set",
				m.AccessTokenValidation)
		default:
			panic("should not happen")
		}
		if m.AccessTokenJWTFilter.Namespace == "" {
			m.AccessTokenJWTFilter.Namespace = namespace
		}
	}

	return nil
}

func (m FilterOAuth2) CallbackURL() *url.URL {
	u, _ := m.ProtectedOrigins[0].Origin.Parse("/.ambassador/oauth2/redirection-endpoint")
	return u
}

func (m FilterOAuth2) RedirectionURL(root int) *url.URL {
    u, _ := m.ProtectedOrigins[root].Origin.ClientURL.Parse("/.ambassador/oauth2/multicookie")

	return u
}

func (m FilterOAuth2) Domain() string {
	return m.ProtectedOrigins[0].Origin.Host
}

func (m FilterOAuth2) TLS() bool {
	return m.ProtectedOrigins[0].Origin.Scheme == "https"
}

//////////////////////////////////////////////////////////////////////

type FilterOAuth2Arguments struct {
	Scopes            []string        `json:"scopes"`
	InsteadOfRedirect *OAuth2Redirect `json:"insteadOfRedirect"`
}

type OAuth2Redirect struct {
	IfRequestHeader HeaderFieldSelector `json:"ifRequestHeader"`
	HTTPStatusCode  int                 `json:"httpStatusCode"`
	Filters         []FilterReference   `json:"filters"`
}

func (m *FilterOAuth2Arguments) Validate(namespace string) error {
	if m.InsteadOfRedirect != nil {
		if m.InsteadOfRedirect.Filters == nil && m.InsteadOfRedirect.HTTPStatusCode == 0 {
			// The default is 403 Forbidden, and definitely not
			// 401 Unauthorized, because the User Agent is not
			// using an RFC 7235-compatible authentication scheme
			// to talk with us; 401 would be inappropriate.
			m.InsteadOfRedirect.HTTPStatusCode = http.StatusForbidden
		}

		if (m.InsteadOfRedirect.HTTPStatusCode == 0) == (m.InsteadOfRedirect.Filters == nil) {
			err := errors.New("must set either 'httpStatusCode' or 'filters'; not both")
			err = errors.Wrap(err, "insteadOfRedirect")
			return err
		}

		if err := validateFilters(m.InsteadOfRedirect.Filters, namespace); err != nil {
			err = errors.Wrap(err, "filters")
			err = errors.Wrap(err, "insteadOfRedirect")
			return err
		}

		if err := m.InsteadOfRedirect.IfRequestHeader.Validate(); err != nil {
			err = errors.Wrap(err, "ifRequestHeader")
			err = errors.Wrap(err, "insteadOfRedirect")
			return err
		}
	}

	return nil
}
