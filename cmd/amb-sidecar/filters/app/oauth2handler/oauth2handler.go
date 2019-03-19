package oauth2handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/secret"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/util"
)

const (
	// AccessTokenCookie cookie's name
	AccessTokenCookie = "access_token"
)

// OAuth2Handler looks up the appropriate Tenant and Rule objects from
// the CRD Controller, and validates the signed JWT tokens when
// present in the request.  If the request Path is "/callback", it
// validates IDP requests and handles code exchange flow.
type OAuth2Handler struct {
	Secret          *secret.Secret
	Filter          crd.FilterOAuth2
	FilterArguments crd.FilterOAuth2Arguments
	OriginalURL     *url.URL
	RedirectURL     *url.URL
}

func (c *OAuth2Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := middleware.GetLogger(r)
	httpClient := httpclient.NewHTTPClient(logger, c.Filter.MaxStale, c.Filter.InsecureTLS)

	discovered, err := Discover(httpClient, c.Filter, logger)
	if err != nil {
		logger.Errorln("create discovery failed: %v", err)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
		return
	}

	token := getToken(r, logger)
	var tokenErr error
	if token == "" {
		tokenErr = errors.New("token not present in the request")
	} else {
		tokenErr = c.validateToken(token, discovered, logger)
	}
	if tokenErr == nil {
		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w.WriteHeader(http.StatusOK)
		return
	}
	logger.Debug(tokenErr)

	switch c.OriginalURL.Path {
	case "/callback":
		if err := r.URL.Query().Get("error"); err != "" {
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			logger.Error("check code failed")
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		res, err := Authorize(httpClient, discovered.TokenEndpoint, AuthorizationRequest{
			GrantType:    "authorization_code", // the default grant used in for this handler
			ClientID:     c.Filter.ClientID,
			Code:         code,
			RedirectURL:  c.Filter.CallbackURL().String(),
			ClientSecret: c.Filter.Secret,
		})
		if err != nil {
			logger.Errorf("authorization request failed: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		logger.Debug("setting authorization cookie")
		http.SetCookie(w, &http.Cookie{
			Name:     AccessTokenCookie,
			Value:    res.AccessToken,
			HttpOnly: true,
			Secure:   c.Filter.TLS(),
			Expires:  time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
		})

		// If the user-agent request was a POST or PUT, 307 will preserve the body
		// and just follow the location header.
		// https://tools.ietf.org/html/rfc7231#section-6.4.7
		logger.Debugf("redirecting user-agent to: %s", c.RedirectURL)
		http.Redirect(w, r, c.RedirectURL.String(), http.StatusTemporaryRedirect)

	default:
		redirect, _ := discovered.AuthorizationEndpoint.Parse("?" + url.Values{
			"audience":      {c.Filter.Audience},
			"response_type": {"code"},
			"redirect_uri":  {c.Filter.CallbackURL().String()},
			"client_id":     {c.Filter.ClientID},
			"state":         {c.signState(r, logger)},
			"scope":         {strings.Join(c.FilterArguments.Scopes, " ")},
		}.Encode())

		logger.Tracef("redirecting to the authorization endpoint: %s", redirect)
		http.Redirect(w, r, redirect.String(), http.StatusSeeOther)
	}
}

func (j *OAuth2Handler) validateToken(token string, discovered *Discovered, logger types.Logger) error {
	jwtParser := jwt.Parser{
		ValidMethods: []string{
			// Any of the RSA algs supported by jwt-go
			"RS256",
			"RS384",
			"RS512",
		},
	}

	var claims jwt.MapClaims
	_, err := jwtParser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
		// Validates key id header.
		if t.Header["kid"] == nil {
			return nil, errors.New("missing kid")
		}

		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid is not a string")
		}

		// Get RSA public key
		return discovered.JSONWebKeySet.GetKey(kid)
	})
	if err != nil {
		return err
	}

	// ParseWithClaims calls claims.Valid(), so
	// jwt.MapClaims.Valid() has already validated 'exp', 'iat',
	// and 'nbf' for us.
	//
	// We _could_ make our own implementation of the jwt.Claims
	// interface that also validates the following things when
	// ParseWithClaims calls claims.Valid(), but that seems like
	// more trouble than it's worth.

	// Verifies 'aud' claim.
	if !claims.VerifyAudience(j.Filter.Audience, false) {
		return errors.Errorf("Token has wrong audience: token=%#v expected=%q", claims["aud"], j.Filter.Audience)
	}

	// Verifies 'iss' claim.
	if !claims.VerifyIssuer(discovered.Issuer, false) {
		return errors.Errorf("Token has wrong issuer: token=%#v expected=%q", claims["iss"], discovered.Issuer)
	}

	// Validate scopes.
	if claims["scope"] != nil {
		// TODO(lukeshu): Verify that this check is
		// correct; it seems backwards to me.
		for _, s := range strings.Split(claims["scope"].(string), " ") {
			logger.Debugf("verifying scope '%s'", s)
			if s != "" && !inArray(s, j.FilterArguments.Scopes) {
				return errors.Errorf("Token scope %v is not in the policy", s)
			}
		}
	} else {
		logger.Debugf("No scopes to verify")
	}

	return nil
}

func inArray(needle string, haystack []string) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}
	return false
}

func getToken(r *http.Request, logger types.Logger) string {
	cookie, _ := r.Cookie(AccessTokenCookie)
	if cookie != nil {
		return cookie.Value
	}

	logger.Debugf("request has no %s cookie", AccessTokenCookie)

	bearer := strings.Split(r.Header.Get("Authorization"), " ")
	if len(bearer) != 2 && strings.ToLower(bearer[0]) != "bearer" {
		logger.Debug("authorization header is not a bearer token")
		return ""
	}

	return bearer[1]
}

func (h *OAuth2Handler) signState(r *http.Request, logger types.Logger) string {
	t := jwt.New(jwt.SigningMethodRS256)
	t.Claims = jwt.MapClaims{
		"exp":          time.Now().Add(h.Filter.StateTTL).Unix(), // time when the token will expire (10 minutes from now)
		"jti":          uuid.Must(uuid.NewV4(), nil).String(),    // a unique identifier for the token
		"iat":          time.Now().Unix(),                        // when the token was issued/created (now)
		"nbf":          0,                                        // time before which the token is not yet valid (2 minutes ago)
		"redirect_url": h.OriginalURL.String(),                   // original request url
	}

	k, err := t.SignedString(h.Secret.GetPrivateKey())
	if err != nil {
		logger.Errorf("failed to sign state: %v", err)
	}

	return k
}

func CheckState(r *http.Request, sec *secret.Secret) (string, error) {
	state := r.URL.Query().Get("state")
	if state == "" {
		return "", errors.New("empty state param")
	}

	token, err := jwt.Parse(state, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return "", fmt.Errorf("unexpected signing method %v", t.Header["redirect_url"])
		}
		return sec.GetPublicKey(), nil
	})

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return "", errors.New("state token validation failed")
	}

	return claims["redirect_url"].(string), nil
}
