package oauth2handler

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/datawire/liboauth2/rfc6749/rfc6749client"
	"github.com/datawire/liboauth2/rfc6750"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/util"
)

const (
	// AccessTokenCookie cookie's name
	accessTokenCookie = "ambassador_bearer_token"
)

type ambassadorBearerToken struct {
	UpstreamResponse rfc6749client.TokenSuccessResponse
}

// OAuth2Handler looks up the appropriate Tenant and Rule objects from
// the CRD Controller, and validates the signed JWT tokens when
// present in the request.  If the request Path is "/callback", it
// validates IDP requests and handles code exchange flow.
type OAuth2Handler struct {
	PrivateKey      *rsa.PrivateKey
	PublicKey       *rsa.PublicKey
	Filter          crd.FilterOAuth2
	FilterArguments crd.FilterOAuth2Arguments
}

func (c *OAuth2Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	originalURL := util.OriginalURL(r)
	logger := middleware.GetLogger(r.Context())
	httpClient := httpclient.NewHTTPClient(logger, c.Filter.MaxStale, c.Filter.InsecureTLS)

	discovered, err := Discover(httpClient, c.Filter, logger)
	if err != nil {
		err = errors.Wrap(err, "OIDC-discovery")
		logger.Errorln(err)
		util.ToJSONResponse(w, http.StatusBadGateway, &util.Error{Message: err.Error()})
		return
	}

	token, err := c.getToken(r)
	if err != nil {
		logger.Infoln(errors.Wrapf(err, "proceeding with no %s", accessTokenCookie))
	} else {
		err = c.validateAccessToken(token.UpstreamResponse.AccessToken, discovered, logger)
		if err != nil {
			logger.Debug(err)
			util.ToJSONResponse(w, http.StatusBadRequest, &util.Error{Message: err.Error()})
			return
		} else {
			rfc6750.AddToHeader(token.UpstreamResponse.AccessToken, w.Header())
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	oauthClient, err := rfc6749client.NewAuthorizationCodeClient(
		c.Filter.ClientID,
		discovered.AuthorizationEndpoint,
		discovered.TokenEndpoint,
		rfc6749client.ClientPasswordHeader(c.Filter.ClientID, c.Filter.Secret),
	)
	if err != nil {
		logger.Errorln(err)
		util.ToJSONResponse(w, http.StatusBadGateway, &util.Error{Message: err.Error()})
		return
	}

	switch originalURL.Path {
	case "/callback":
		authorizationResponse, err := oauthClient.ParseAuthorizationResponse(r)
		if err != nil {
			util.ToJSONResponse(w, http.StatusBadRequest, &util.Error{Message: err.Error()})
			return
		}
		redirectURLstr, err := checkState(authorizationResponse.GetState(), c.PublicKey)
		if err != nil {
			// This mostly indicates an XSRF-type attack.
			// The request wasn't malformed, but one of
			// the credentials in it (the 'state'
			// parameter) was.
			err = errors.Wrap(err, "state parameter")
			logger.Errorln(err)
			util.ToJSONResponse(w, http.StatusUnauthorized, &util.Error{Message: err.Error()})
			return
		}
		redirectURL, err := url.Parse(redirectURLstr)
		if err != nil {
			// this should never happen -- the state was
			// signed as valid; for this to happen, either
			// (1) the crypto apocalypse has come, or (2)
			// we generated an invalid state when we
			// submitted the authorization request.
			// Assuming that (2) is more likely, that's an
			// internal server issue.
			err = errors.Wrapf(err, "state parameter: redirect_url: %q", redirectURLstr)
			logger.Errorln(err)
			util.ToJSONResponse(w, http.StatusInternalServerError, &util.Error{Message: err.Error()})
			return
		}
		switch authorizationResponse := authorizationResponse.(type) {
		case rfc6749client.AuthorizationCodeAuthorizationErrorResponse:
			util.ToJSONResponse(w, http.StatusUnauthorized, map[string]interface{}{
				"message":           "unauthorized: authorization request failed",
				"upstream_response": authorizationResponse,
				"error_meaning":     authorizationResponse.ErrorMeaning(),
			})
			return
		case rfc6749client.AuthorizationCodeAuthorizationSuccessResponse:
			tokenResponse, err := oauthClient.AccessToken(httpClient, authorizationResponse.Code, c.Filter.CallbackURL())
			if err != nil {
				logger.Errorln(err)
				util.ToJSONResponse(w, http.StatusBadGateway, &util.Error{Message: err.Error()})
				return
			}
			switch tokenResponse := tokenResponse.(type) {
			case rfc6749client.TokenErrorResponse:
				util.ToJSONResponse(w, http.StatusInternalServerError, map[string]interface{}{
					"message":           "token request failed",
					"upstream_response": tokenResponse,
					"error_meaning":     tokenResponse.ErrorMeaning(),
				})
				return
			case rfc6749client.TokenSuccessResponse:
				logger.Debug("setting authorization cookie")
				err = c.setToken(w, ambassadorBearerToken{
					UpstreamResponse: tokenResponse,
				})
				if err != nil {
					logger.Errorln(err)
					util.ToJSONResponse(w, http.StatusInternalServerError, &util.Error{Message: err.Error()})
					return
				}
				// If the user-agent request was a POST or PUT, 307 will preserve the body
				// and just follow the location header.
				// https://tools.ietf.org/html/rfc7231#section-6.4.7
				logger.Debugf("redirecting user-agent to: %s", redirectURL)
				http.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
				return
			}
		}
	default:
		scope := make(rfc6749client.Scope, len(c.FilterArguments.Scopes))
		for _, s := range c.FilterArguments.Scopes {
			scope[s] = struct{}{}
		}
		oauthClient.AuthorizationRequest(w, r,
			c.Filter.CallbackURL(), scope, c.signState(originalURL, logger))
	}
}

func (j *OAuth2Handler) validateAccessToken(token string, discovered *Discovered, logger types.Logger) error {
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
		var scopes []string
		switch scope := claims["scope"].(type) {
		case string:
			for _, s := range strings.Split(scope, " ") {
				if s == "" {
					continue
				}
				scopes = append(scopes, s)
			}
		case []interface{}: // this seems to be out-of-spec, but UAA does it
			for _, _s := range scope {
				s, ok := _s.(string)
				if !ok {
					logger.Warningf("Unexpected scope[n] type: %T", _s)
					continue
				}
				scopes = append(scopes, s)
			}
		default:
			logger.Warningf("Unexpected scope type: %T", scope)
		}
		// TODO(lukeshu): Verify that this check is
		// correct; it seems backwards to me.
		for _, s := range scopes {
			logger.Debugf("verifying scope '%s'", s)
			if !inArray(s, j.FilterArguments.Scopes) {
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

func (c *OAuth2Handler) getToken(r *http.Request) (ambassadorBearerToken, error) {
	cookie, err := r.Cookie(accessTokenCookie)
	if err != nil {
		return ambassadorBearerToken{}, err
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return ambassadorBearerToken{}, err
	}
	cleartext, err := c.cryptoDecryptAndVerify(ciphertext, []byte(accessTokenCookie))
	if err != nil {
		return ambassadorBearerToken{}, err
	}
	var token ambassadorBearerToken
	err = json.Unmarshal(cleartext, &token)
	if err != nil {
		return ambassadorBearerToken{}, err
	}
	return token, nil
}

func (c *OAuth2Handler) setToken(w http.ResponseWriter, token ambassadorBearerToken) error {
	cleartext, err := json.Marshal(token)
	if err != nil {
		return err
	}
	ciphertext, err := c.cryptoSignAndEncrypt(cleartext, []byte(accessTokenCookie))
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:  accessTokenCookie,
		Value: base64.RawURLEncoding.EncodeToString(ciphertext),
		// TODO(lukeshu): Verify that these are sane cookie parameters
		HttpOnly: true,
		Secure:   c.Filter.TLS(),
	})
	return nil
}

func (h *OAuth2Handler) signState(originalURL *url.URL, logger types.Logger) string {
	t := jwt.New(jwt.SigningMethodRS256)
	t.Claims = jwt.MapClaims{
		"exp":          time.Now().Add(h.Filter.StateTTL).Unix(), // time when the token will expire (10 minutes from now)
		"jti":          uuid.Must(uuid.NewV4(), nil).String(),    // a unique identifier for the token
		"iat":          time.Now().Unix(),                        // when the token was issued/created (now)
		"nbf":          0,                                        // time before which the token is not yet valid (2 minutes ago)
		"redirect_url": originalURL.String(),                     // original request url
	}

	k, err := t.SignedString(h.PrivateKey)
	if err != nil {
		logger.Errorf("failed to sign state: %v", err)
	}

	return k
}

func checkState(state string, pubkey *rsa.PublicKey) (string, error) {
	if state == "" {
		return "", errors.New("empty state param")
	}

	token, err := jwt.Parse(state, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return "", errors.Errorf("unexpected signing method %v", t.Header["redirect_url"])
		}
		return pubkey, nil
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
