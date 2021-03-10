package v1

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/miekg/dns"
	"github.com/pkg/errors"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1client "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/datawire/apro/lib/oauth2/client/rfc6749"
)

const (
	GrantType_AuthorizationCode = "AuthorizationCode"
	GrantType_ClientCredentials = "ClientCredentials"
	GrantType_Password          = "Password"
)

var grantTypes = []string{
	GrantType_AuthorizationCode,
	GrantType_ClientCredentials,
	GrantType_Password,
}

// When adding a new ClientAuthenticationMethod, be sure to add it to
//  1. the 'clientAuthenticationMethods' array (below)
//  2. the 'clientcommon.ConfigurableClientAuthenticator.concrete()' function (elsewhere)
//  3. the 'client_credentials_clisnt.OAuth2Client.Filter' function (elsewhere)
const (
	ClientAuthenticationMethod_HeaderPassword = "HeaderPassword"
	ClientAuthenticationMethod_BodyPassword   = "BodyPassword"
	ClientAuthenticationMethod_JWTAssertion   = "JWTAssertion"
)

var clientAuthenticationMethods = []string{
	ClientAuthenticationMethod_HeaderPassword,
	ClientAuthenticationMethod_BodyPassword,
	ClientAuthenticationMethod_JWTAssertion,
}

type FilterOAuth2 struct {
	RawAuthorizationURL string   `json:"authorizationURL"` // formerly AUTH_PROVIDER_URL
	AuthorizationURL    *url.URL `json:"-"`                // calculated from RawAuthorizationURL

	// Client settings /////////////////////////////////////////////////////
	RawExpirationSafetyMargin string               `json:"expirationSafetyMargin"`
	ExpirationSafetyMargin    time.Duration        `json:"-"`
	GrantType                 string               `json:"grantType"`
	ClientAuthentication      ClientAuthentication `json:"clientAuthentication"`

	// Client settings: grantType=AuthorizationCode ////////////////////////
	UseSessionCookies            *UseSessionCookies `json:"useSessionCookies"`
	DeprecatedClientURL          string             `json:"clientURL"` // formerly tenant.tenantUrl; now replaced by protectedOrigins
	ProtectedOrigins             []Origin           `json:"protectedOrigins"`
	ExtraAuthorizationParameters map[string]string  `json:"extraAuthorizationParameters"`

	// Client settings: grantType≠ClientCredentials ////////////////////////
	ClientID        string `json:"clientID"`
	Secret          string `json:"secret"`
	SecretName      string `json:"secretName"`
	SecretNamespace string `json:"secretNamespace"`

	// Resource Server settings ////////////////////////////////////////////
	AccessTokenValidation     string             `json:"accessTokenValidation"`
	AccessTokenJWTFilter      JWTFilterReference `json:"accessTokenJWTFilter"`
	AllowMalformedAccessToken bool               `json:"allowMalformedAccessToken"`

	// HTTP settings ///////////////////////////////////////////////////////
	RawMaxStale       string                   `json:"maxStale"`
	MaxStale          time.Duration            `json:"-"` // calculated from RawMaxStale
	InsecureTLS       bool                     `json:"insecureTLS"`
	RawRenegotiateTLS string                   `json:"renegotiateTLS"`
	RenegotiateTLS    tls.RenegotiationSupport `json:"-"` // calculated from RawRenegotiateTLS

	InjectRequestHeaders []HeaderFieldTemplate `json:"injectRequestHeaders"`

	// Deprecated and entirely ignored /////////////////////////////////////
	DeprecatedStateTTL string `json:"stateTTL"`
}

type ClientAuthentication struct {
	Method       string            `json:"method"`
	JWTAssertion *JWTAssertionSpec `json:"jwtAssertion"`
}

func (m *ClientAuthentication) Validate() error {
	if m.Method == "" {
		m.Method = ClientAuthenticationMethod_HeaderPassword
	}
	if !inArray(m.Method, clientAuthenticationMethods) {
		return errors.Errorf("method=%q is invalid; valid values are %q",
			m.Method, clientAuthenticationMethods)
	}

	if m.Method != ClientAuthenticationMethod_JWTAssertion {
		if m.JWTAssertion != nil {
			return errors.New("field 'jwtAssertion' is only valid if method='JWTAssertion'")
		}
	} else {
		if m.JWTAssertion == nil {
			m.JWTAssertion = &JWTAssertionSpec{}
		}
		if err := m.JWTAssertion.Validate(); err != nil {
			return errors.Wrap(err, "jwtAssertion")
		}
	}

	return nil
}

type JWTAssertionSpec struct {
	SetClientID           bool                   `json:"setClientID"`
	Audience              string                 `json:"audience"`
	RawSigningMethod      string                 `json:"signingMethod"`
	SigningMethod         jwt.SigningMethod      `json:"-"`
	Lifetime              time.Duration          `json:"lifetime"`
	SetNBF                bool                   `json:"setNBF"`
	NBFSafetyMargin       time.Duration          `json:"nbfSafetyMargin"`
	SetIAT                bool                   `json:"setIAT"`
	OtherClaims           map[string]interface{} `json:"otherClaims"`
	OtherHeaderParameters map[string]interface{} `json:"otherHeaderParameters"`
}

func (m *JWTAssertionSpec) Validate() error {
	if m.RawSigningMethod == "" {
		m.RawSigningMethod = "RS256"
	}
	m.SigningMethod = jwt.GetSigningMethod(m.RawSigningMethod)
	if m.SigningMethod == nil {
		return errors.Errorf("invalid signingMethod=%q", m.RawSigningMethod)
	}

	if m.Lifetime == 0 {
		m.Lifetime = 1 * time.Minute
	}

	return nil
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
	Name                 string             `json:"name"`
	Namespace            string             `json:"namespace"`
	InheritScopeArgument bool               `json:"inheritScopeArgument"`
	StripInheritedScope  bool               `json:"stripInheritedScope"`
	Arguments            FilterJWTArguments `json:"arguments"`
}

type Origin struct {
	RawOrigin         string   `json:"origin"`
	Origin            *url.URL `json:"-"` // RawOrigin
	RawInternalOrigin string   `json:"internalOrigin"`
	InternalOrigin    *url.URL `json:"-"` // RawInternalOrigin
	IncludeSubdomains bool     `json:"includeSubdomains"`
}

func (m *Origin) Validate() error {
	// Handle m.Origin
	u, err := url.Parse(m.RawOrigin)
	if err != nil {
		return errors.Wrapf(err, "parsing origin: %q", m.RawOrigin)
	}
	if !u.IsAbs() {
		return errors.New("origin is not an absolute URL")
	}
	m.Origin = u

	// Handle m.InternalOrigin
	if m.RawInternalOrigin != "" {
		// A little hacky because url.Parse doesn't like scheme="*"
		if strings.HasPrefix(m.RawInternalOrigin, "*://") {
			u, err = url.Parse("https" + m.RawInternalOrigin[1:])
		} else {
			u, err = url.Parse(m.RawInternalOrigin)
		}
		if err != nil {
			return errors.Wrapf(err, "parsing internalOrigin: %q", m.RawInternalOrigin)
		}
		if !u.IsAbs() {
			return errors.New("internalOrigin is not an absolute URL")
		}
		if strings.HasPrefix(m.RawInternalOrigin, "*://") {
			u.Scheme = "*"
		}
		if m.IncludeSubdomains && (u.Scheme == "*" || u.Host == "*") {
			return errors.New("internalOrigin * wildcards cannot be used with includeSubdomains=true")
		}
		m.InternalOrigin = u
	}

	return nil
}

// Matches returns whether a given request URL matches this Origin.
func (m *Origin) Matches(u *url.URL) bool {
	// By default, we just use m.Origin.  But if m.InternalOrigin is set, then the
	// user has told us that we're behind another gateway that's rewriting the `Host:`
	// header (or something like that), and that while we should present the origin as
	// m.Origin, it will actually look to us like m.InternalOrigin on incoming
	// requests.
	origin := m.Origin
	if m.InternalOrigin != nil {
		origin = m.InternalOrigin
	}

	var matched bool
	// Both the scheme and the authority section of a URL are case-insensitive; and
	// also, m.InternalOrigin allows simple wildcards (to simplify the single-domain
	// behind-a-rewriting-gateway case; we only need to know the exact mapping of
	// external→internal origins in the multi-domain case; so only require that the
	// user provide it in the multi-domain case).
	if (origin.Scheme == "*" || strings.EqualFold(origin.Scheme, u.Scheme)) &&
		(origin.Host == "*" || strings.EqualFold(origin.Host, u.Host)) {
		matched = true
	}

	// Let's check if we allow subdomains and if this host is a subdomain
	if m.IncludeSubdomains && dns.IsSubDomain(m.Origin.Host, u.Host) {
		matched = true
	}

	return matched
}

func ParsePrivateKey(method jwt.SigningMethod, keyStr string) (interface{}, error) {
	switch method.(type) {
	case *jwt.SigningMethodRSA, *jwt.SigningMethodRSAPSS:
		key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(keyStr))
		if err != nil {
			return nil, errors.Wrap(err, "could not parse as an RSA private key")
		}
		return key, nil
	case *jwt.SigningMethodECDSA:
		key, err := jwt.ParseECPrivateKeyFromPEM([]byte(keyStr))
		if err != nil {
			return nil, errors.Wrap(err, "could not parse as an EC private key")
		}
		return key, nil
	case *jwt.SigningMethodHMAC:
		return []byte(keyStr), nil
	default:
		return nil, errors.Errorf("unsupported signing method %q", method.Alg())
	}
}

//nolint:gocyclo
func (m *FilterOAuth2) Validate(namespace string, secretsGetter coreV1client.SecretsGetter) error {
	// Global settings /////////////////////////////////////////////////////

	u, err := url.Parse(m.RawAuthorizationURL)
	if err != nil {
		return errors.Wrapf(err, "parsing authorizationURL: %q", m.RawAuthorizationURL)
	}
	if !u.IsAbs() {
		return errors.New("authorizationURL is not an absolute URL")
	}
	m.AuthorizationURL = u

	// Client settings /////////////////////////////////////////////////////

	if m.RawExpirationSafetyMargin != "" {
		d, err := time.ParseDuration(m.RawExpirationSafetyMargin)
		if err != nil {
			return errors.Wrapf(err, "parsing expirationSafetyMargin: %q", m.RawExpirationSafetyMargin)
		}
		m.ExpirationSafetyMargin = d
	}

	if m.GrantType == "" {
		m.GrantType = GrantType_AuthorizationCode
	}
	if !inArray(m.GrantType, grantTypes) {
		return errors.Errorf("grantType=%q is invalid; valid values are %q",
			m.GrantType, grantTypes)
	}

	if err := m.ClientAuthentication.Validate(); err != nil {
		return errors.Wrapf(err, "clientAuthentication")
	}

	// Client settings: grantType=AuthorizationCode ////////////////////////
	if m.GrantType == GrantType_AuthorizationCode {
		if m.UseSessionCookies == nil {
			value := false
			m.UseSessionCookies = &UseSessionCookies{
				Value: &value,
			}
		}
		if err := m.UseSessionCookies.Validate(); err != nil {
			return errors.Wrap(err, "useSessionCookies")
		}

		if m.DeprecatedClientURL != "" {
			if len(m.ProtectedOrigins) > 0 {
				return errors.New("it is invalid to set both 'clientURL' and 'protectedOrigins'; 'clientURL' is deprecated and should be replaced by 'protectedOrigins'")
			}
			m.ProtectedOrigins = []Origin{
				{
					RawOrigin:         m.DeprecatedClientURL,
					RawInternalOrigin: "*://*",
				},
			}
			m.DeprecatedClientURL = ""
		}

		if len(m.ProtectedOrigins) < 1 {
			return errors.New("must have at least one 'protectedOrigin' when 'grantType==AuthorizationCode'")
		}
		for i := range m.ProtectedOrigins {
			origin := &(m.ProtectedOrigins[i])
			if err := origin.Validate(); err != nil {
				return errors.Wrapf(err, "protectedOrigins[%d]", i)
			}
		}

		for key := range m.ExtraAuthorizationParameters {
			if _, conflicts := rfc6749.CoreAuthorizationParameters[key]; conflicts {
				return errors.Errorf("extraAuthorizationParameters: may not manually specify built-in OAuth parameter %q", key)
			}
		}
	} else {
		if m.UseSessionCookies != nil {
			return errors.New("it is invalid to set 'useSessionCookies' when 'grantType!=AuthorizationCode'")
		}
		if m.DeprecatedClientURL != "" {
			return errors.New("it is invalid to set 'clientURL' when 'grantType!=AuthorizationCode'")
		}
		if len(m.ProtectedOrigins) > 0 {
			return errors.New("it is invalid to set 'protectedOrigins' when 'grantType!=AuthorizationCode'")
		}
		if len(m.ExtraAuthorizationParameters) > 0 {
			return errors.New("it is invalid to set 'extraAuthorizationParameters' when 'grantType!=AuthorizationCode'")
		}
	}

	// Client settings: grantType≠ClientCredentials ////////////////////////
	if m.GrantType != GrantType_ClientCredentials {
		if m.ClientID == "" {
			return errors.New("it is required to set 'clientID' when 'grantType!=ClientCredentials'")
		}
		if m.SecretName != "" {
			if m.Secret != "" {
				return errors.New("it is invalid to set both 'secret' and 'secretName'")
			}
			if m.SecretNamespace == "" {
				m.SecretNamespace = namespace
			}
			secret, err := secretsGetter.Secrets(m.SecretNamespace).Get(context.TODO(), m.SecretName, metaV1.GetOptions{})
			if err != nil {
				return errors.Wrapf(err, "getting secret name=%q namespace=%q", m.SecretName, m.SecretNamespace)
			}
			secretVal, ok := secret.Data["oauth2-client-secret"]
			if !ok {
				return errors.Errorf("secret name=%q namespace=%q does not contain an 'oauth2-client-secret' field", m.SecretName, m.SecretNamespace)
			}
			m.Secret = string(secretVal)
		}
		if m.Secret == "" {
			return errors.New("it is required to set either 'secret' or 'secretName' when 'grantType!=ClientCredentials'")
		}
		if m.ClientAuthentication.Method == ClientAuthenticationMethod_JWTAssertion {
			if _, err := ParsePrivateKey(m.ClientAuthentication.JWTAssertion.SigningMethod, m.Secret); err != nil {
				return errors.Wrap(err, "client secret")
			}
		}
	} else {
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
	}

	// Resource Server settings ////////////////////////////////////////////

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
	if m.GrantType == GrantType_ClientCredentials {
		switch m.AccessTokenValidation {
		case "auto":
			m.AccessTokenValidation = "jwt"
		case "jwt":
			// do nothing
		case "userinfo":
			return errors.Errorf("accessTokenValidation=%q but grantType=%q; there will be no user to look up the user info of",
				m.AccessTokenValidation, m.GrantType)
		default:
			panic("should not happen")
		}
	}

	// HTTP settings ///////////////////////////////////////////////////////

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

	for i := range m.InjectRequestHeaders {
		hf := &(m.InjectRequestHeaders[i])
		if err := hf.Validate(); err != nil {
			return errors.Wrap(err, "injectRequestHeaders")
		}
	}

	return nil
}

func (m FilterOAuth2) CallbackURL() *url.URL {
	u, _ := m.ProtectedOrigins[0].Origin.Parse("/.ambassador/oauth2/redirection-endpoint")
	return u
}

func (m FilterOAuth2) RedirectionURL(root int) *url.URL {
	u, _ := m.ProtectedOrigins[root].Origin.Parse("/.ambassador/oauth2/multicookie")

	return u
}

func (m FilterOAuth2) GetOrigin(root int) string {
	return m.ProtectedOrigins[root].Origin.Host
}

func (m FilterOAuth2) AllowSubdomains(root int) bool {
	return m.ProtectedOrigins[root].IncludeSubdomains
}

func (m FilterOAuth2) Domain() string {
	return m.ProtectedOrigins[0].Origin.Host
}

func (m FilterOAuth2) TLS() bool {
	return m.ProtectedOrigins[0].Origin.Scheme == "https"
}

//////////////////////////////////////////////////////////////////////

type FilterOAuth2Arguments struct {
	DeprecatedScopes  []string        `json:"scopes"`
	Scope             []string        `json:"scope"`
	InsteadOfRedirect *OAuth2Redirect `json:"insteadOfRedirect"`
	SameSiteString    string          `json:"sameSite"`

	// This is the parsed value of SameSiteString. If SameSiteString was never set,
	// then this remains as the zero-value for an http.SameSite type, which ends
	// up not setting the SameSite attribute on the cookie at all. The result is that
	// the browser decides the default value for SameSite.
	//
	// If the SameSiteString was set, then it is validated against "lax", "none", and
	// "strict" and this value is set accordingly.
	SameSite http.SameSite `json:"-"`
}

type OAuth2Redirect struct {
	IfRequestHeader HeaderFieldSelector `json:"ifRequestHeader"`
	HTTPStatusCode  int                 `json:"httpStatusCode"`
	Filters         []FilterReference   `json:"filters"`
}

func (m *FilterOAuth2Arguments) Validate(namespace string) error {
	if len(m.DeprecatedScopes) > 0 {
		m.Scope = append(m.Scope, m.DeprecatedScopes...)
		m.DeprecatedScopes = nil
	}

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

	// If the SameSite attribute is supplied as an argument, attempt to parse it.
	// It is an error to set the SameSite attribute but provide an unknown value.
	if m.SameSiteString != "" {
		lowerValue := strings.ToLower(m.SameSiteString)
		switch lowerValue {
		case "strict":
			m.SameSite = http.SameSiteStrictMode
		case "lax":
			m.SameSite = http.SameSiteLaxMode
		case "none":
			m.SameSite = http.SameSiteNoneMode
		default:
			return fmt.Errorf("invalid SameSite value '%v': supported values include 'default', 'lax', 'strict', and 'none'", m.SameSiteString)
		}
	}

	return nil
}
