package oauth2handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta1"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/client"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/oauth/app/secret"
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
	Config      types.Config
	Logger      types.Logger
	Secret      *secret.Secret
	Rule        crd.Rule
	Middleware  crd.MiddlewareOAuth2
	OriginalURL *url.URL
	RedirectURL *url.URL
}

func (c *OAuth2Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	disco, err := discovery.New(c.Middleware, c.Logger)
	if err != nil {
		c.Logger.Debugf("create discovery: %v", err)
		util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
	}

	token := c.getToken(r)
	var tokenErr error
	if token == "" {
		tokenErr = errors.New("token not present in the request")
	} else {
		tokenErr = c.validateToken(token, disco)
	}
	if tokenErr == nil {
		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w.WriteHeader(http.StatusOK)
		return
	}
	c.Logger.Debug(tokenErr)

	switch c.OriginalURL.Path {
	case "/callback":
		if err := r.URL.Query().Get("error"); err != "" {
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			c.Logger.Error("check code failed")
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: "unauthorized"})
			return
		}

		res, err := client.NewRestClient(disco.AuthorizationEndpoint, disco.TokenEndpoint).Authorize(&client.AuthorizationRequest{
			GrantType:    "authorization_code", // the default grant used in for this handler
			ClientID:     c.Middleware.ClientID,
			Code:         code,
			RedirectURL:  c.Middleware.CallbackURL().String(),
			ClientSecret: c.Middleware.Secret,
		})
		if err != nil {
			c.Logger.Errorf("authorization request failed: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		c.Logger.Debug("setting authorization cookie")
		http.SetCookie(w, &http.Cookie{
			Name:     AccessTokenCookie,
			Value:    res.AccessToken,
			HttpOnly: true,
			Secure:   c.Middleware.TLS(),
			Expires:  time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
		})

		// If the user-agent request was a POST or PUT, 307 will preserve the body
		// and just follow the location header.
		// https://tools.ietf.org/html/rfc7231#section-6.4.7
		c.Logger.Debugf("redirecting user-agent to: %s", c.RedirectURL)
		http.Redirect(w, r, c.RedirectURL.String(), http.StatusTemporaryRedirect)

	default:
		redirect, _ := disco.AuthorizationEndpoint.Parse("?" + url.Values{
			"audience":      {c.Middleware.Audience},
			"response_type": {"code"},
			"redirect_uri":  {c.Middleware.CallbackURL().String()},
			"client_id":     {c.Middleware.ClientID},
			"state":         {c.signState(r)},
			"scope":         {c.Rule.Scope},
		}.Encode())

		c.Logger.Tracef("redirecting to the authorization endpoint: %s", redirect)
		http.Redirect(w, r, redirect.String(), http.StatusSeeOther)
	}
}

func (j *OAuth2Handler) validateToken(token string, disco *discovery.Discovery) error {
	// JWT validation is performed by doing the cheap operations first.
	_, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		// Validates key id header.
		if t.Header["kid"] == nil {
			return "", errors.New("missing kid")
		}

		// Get RSA certificate.
		cert, err := disco.GetPemCert(t.Header["kid"].(string))
		if err != nil {
			return "", err
		}

		// Get map of claims.
		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			return "", errors.New("failed to extract claims")
		}

		//fmt.Printf("Expected aud: %s\n", middleware.Audience)
		//fmt.Printf("Expected iss: %s\n", j.IssuerURL)
		//spew.Dump(claims)

		// Verifies 'aud' claim.
		if !claims.VerifyAudience(j.Middleware.Audience, false) {
			return "", fmt.Errorf("invalid audience %s", j.Middleware.Audience)
		}

		// Verifies 'iss' claim.
		if !claims.VerifyIssuer(disco.Issuer, false) {
			return "", errors.New("invalid issuer")
		}

		// Validates time based claims "exp, iat, nbf".
		if err := t.Claims.Valid(); err != nil {
			return "", err
		}

		// Validate scopes.
		if claims["scope"] != nil {
			for _, s := range strings.Split(claims["scope"].(string), " ") {
				j.Logger.Debugf("verifying scope %s", s)
				if !j.Rule.MatchScope(s) {
					return "", fmt.Errorf("scope %v is not in the policy", s)
				}
			}
		}

		// Validate method for last since it's the most expensive operation.
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return "", errors.New("unexpected signing method")
		}

		return jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
	})
	return err
}

func (j *OAuth2Handler) getToken(r *http.Request) string {
	cookie, _ := r.Cookie(AccessTokenCookie)
	if cookie != nil {
		return cookie.Value
	}

	j.Logger.Debugf("request has no %s cookie", AccessTokenCookie)

	bearer := strings.Split(r.Header.Get("Authorization"), " ")
	if len(bearer) != 2 && strings.ToLower(bearer[0]) != "bearer" {
		j.Logger.Debug("authorization header is not a bearer token")
		return ""
	}

	return bearer[1]
}

func (h *OAuth2Handler) signState(r *http.Request) string {
	t := jwt.New(jwt.SigningMethodRS256)
	t.Claims = jwt.MapClaims{
		"exp":          time.Now().Add(h.Middleware.StateTTL).Unix(), // time when the token will expire (10 minutes from now)
		"jti":          uuid.Must(uuid.NewV4(), nil).String(),        // a unique identifier for the token
		"iat":          time.Now().Unix(),                            // when the token was issued/created (now)
		"nbf":          0,                                            // time before which the token is not yet valid (2 minutes ago)
		"redirect_url": h.OriginalURL.String(),                       // original request url
	}

	k, err := t.SignedString(h.Secret.GetPrivateKey())
	if err != nil {
		h.Logger.Errorf("failed to sign state: %v", err)
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
