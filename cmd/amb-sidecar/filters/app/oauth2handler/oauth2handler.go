package oauth2handler

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/datawire/liboauth2/client/rfc6749"
	"github.com/datawire/liboauth2/client/rfc6750"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
)

const (
	// AccessTokenCookie cookie's name
	accessTokenCookie = "ambassador_session"
)

type ambassadorBearerToken struct {
	SessionData *rfc6749.AuthorizationCodeClientSessionData
}

// OAuth2Filter looks up the appropriate Tenant and Rule objects from
// the CRD Controller, and validates the signed JWT tokens when
// present in the request.  If the request Path is "/callback", it
// validates IDP requests and handles code exchange flow.
type OAuth2Filter struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	RedisPool  *pool.Pool
	Spec       crd.FilterOAuth2
	Arguments  crd.FilterOAuth2Arguments
}

func (c *OAuth2Filter) Filter(ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := middleware.GetLogger(ctx)
	httpClient := httpclient.NewHTTPClient(logger, c.Spec.MaxStale, c.Spec.InsecureTLS)

	discovered, err := Discover(httpClient, c.Spec, logger)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			errors.Wrap(err, "OIDC-discovery"), nil), nil
	}

	sessionData, err := c.getSession(request)
	if err != nil {
		logger.Debugln("session status:", errors.Wrapf(err, "proceeding with no %s", accessTokenCookie))
	} else if sessionData.CurrentAccessToken == nil {
		logger.Debugln("session status:", "proceeding with non-authenticated session")
	} else if err := c.validateAccessToken(sessionData.CurrentAccessToken.AccessToken, discovered, httpClient, logger); err != nil {
		logger.Debugln("session status:", errors.Wrap(err, "invalid access token"))
		return middleware.NewErrorResponse(ctx, http.StatusBadRequest, err, nil), nil
	} else {
		logger.Debugln("session status:", "valid access token")
		header := make(http.Header)
		rfc6750.AddToHeader(sessionData.CurrentAccessToken.AccessToken, header)
		ret := &filterapi.HTTPRequestModification{}
		for k, vs := range header {
			for _, v := range vs {
				ret.Header = append(ret.Header, &filterapi.HTTPHeaderReplaceValue{
					Key:   k,
					Value: v,
				})
			}
		}
		return ret, nil
	}
	logger.Debugf("session data: %#v", sessionData)

	oauthClient, err := rfc6749.NewAuthorizationCodeClient(
		c.Spec.ClientID,
		discovered.AuthorizationEndpoint,
		discovered.TokenEndpoint,
		rfc6749.ClientPasswordHeader(c.Spec.ClientID, c.Spec.Secret),
		httpClient,
	)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			err, nil), nil
	}

	u, err := url.ParseRequestURI(request.GetRequest().GetHttp().GetPath())
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Wrapf(err, "could not parse URI: %q", request.GetRequest().GetHttp().GetPath()), nil), nil
	}
	switch u.Path {
	case "/callback":
		authorizationCode, err := oauthClient.ParseAuthorizationResponse(sessionData, u)
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusBadRequest,
				err, nil), nil
		}
		originalURL, err := checkState(sessionData.Request.State, c.PublicKey)
		if err != nil {
			// This should never happen--the state matched
			// what we stored in Redis.  For this to
			// happen, either (1) our Redis server was
			// cracked, or (2) we generated an invalid
			// state when we submitted the authorization
			// request.  Assuming that (2) is more likely,
			// that's an internal server issue.
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrapf(err, "invalid state"), nil), nil
		}
		if err := oauthClient.AccessToken(sessionData, authorizationCode); err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
				err, nil), nil
		}
		logger.Debug("setting authorization cookie")
		cookie, err := c.setSession(sessionData)
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				err, nil), nil
		}
		logger.Debugf("redirecting user-agent to: %s", originalURL)
		return &filterapi.HTTPResponse{
			StatusCode: http.StatusSeeOther,
			Header: http.Header{
				"Set-Cookie": {cookie.String()},
				"Location":   {originalURL},
			},
			Body: "",
		}, nil
	default:
		// https://github.com/datawire/ambassador/issues/1581
		originalURL, err := url.ParseRequestURI(request.GetRequest().GetHttp().GetHeaders()["x-forwarded-proto"] + "://" + request.GetRequest().GetHttp().GetHost() + request.GetRequest().GetHttp().GetPath())
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "failed to construct URL"), nil), nil
		}
		scope := make(rfc6749.Scope, len(c.Arguments.Scopes))
		for _, s := range c.Arguments.Scopes {
			scope[s] = struct{}{}
		}
		scope["openid"] = struct{}{}
		authorizationRequestURI, sessionData, err := oauthClient.AuthorizationRequest(c.Spec.CallbackURL(), scope, c.signState(originalURL, logger))
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				err, nil), nil
		}
		logger.Debug("setting authorization cookie")
		cookie, err := c.setSession(sessionData)
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				err, nil), nil
		}
		return &filterapi.HTTPResponse{
			// A 302 "Found" may or may not convert POST->GET.  We want
			// the UA to GET the Authorization URI, so we shouldn't use
			// 302 which may or may not do the right thing, but use 303
			// "See Other" which MUST convert to GET.
			StatusCode: http.StatusSeeOther,
			Header: http.Header{
				"Set-Cookie": {cookie.String()},
				"Location":   {authorizationRequestURI.String()},
			},
			Body: "",
		}, nil
	}
}

func (j *OAuth2Filter) validateAccessToken(token string, discovered *Discovered, httpClient *http.Client, logger types.Logger) error {
	req, err := http.NewRequest("GET", discovered.UserInfoEndpoint.String(), nil)
	if err != nil {
		return err
	}
	rfc6750.AddToHeader(token, req.Header)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return errors.Errorf("token validation through userinfo endpoint failed: HTTP %d", res.StatusCode)
	}
	return nil

	// jwtParser := jwt.Parser{
	// 	ValidMethods: []string{
	// 		// Any of the RSA algs supported by jwt-go
	// 		"RS256",
	// 		"RS384",
	// 		"RS512",
	// 	},
	// }

	// var claims jwt.MapClaims
	// _, err := jwtParser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
	// 	// Validates key id header.
	// 	if t.Header["kid"] == nil {
	// 		return nil, errors.New("missing kid")
	// 	}

	// 	kid, ok := t.Header["kid"].(string)
	// 	if !ok {
	// 		return nil, errors.New("kid is not a string")
	// 	}

	// 	// Get RSA public key
	// 	return discovered.JSONWebKeySet.GetKey(kid)
	// })
	// if err != nil {
	// 	return err
	// }

	// // ParseWithClaims calls claims.Valid(), so
	// // jwt.MapClaims.Valid() has already validated 'exp', 'iat',
	// // and 'nbf' for us.
	// //
	// // We _could_ make our own implementation of the jwt.Claims
	// // interface that also validates the following things when
	// // ParseWithClaims calls claims.Valid(), but that seems like
	// // more trouble than it's worth.

	// // Verifies 'aud' claim.
	// if !claims.VerifyAudience(j.Spec.Audience, false) {
	// 	return errors.Errorf("Token has wrong audience: token=%#v expected=%q", claims["aud"], j.Spec.Audience)
	// }

	// // Verifies 'iss' claim.
	// if !claims.VerifyIssuer(discovered.Issuer, false) {
	// 	return errors.Errorf("Token has wrong issuer: token=%#v expected=%q", claims["iss"], discovered.Issuer)
	// }

	// // Validate scopes.
	// if claims["scope"] != nil {
	// 	var scopes []string
	// 	switch scope := claims["scope"].(type) {
	// 	case string:
	// 		for _, s := range strings.Split(scope, " ") {
	// 			if s == "" {
	// 				continue
	// 			}
	// 			scopes = append(scopes, s)
	// 		}
	// 	case []interface{}: // this seems to be out-of-spec, but UAA does it
	// 		for _, _s := range scope {
	// 			s, ok := _s.(string)
	// 			if !ok {
	// 				logger.Warningf("Unexpected scope[n] type: %T", _s)
	// 				continue
	// 			}
	// 			scopes = append(scopes, s)
	// 		}
	// 	default:
	// 		logger.Warningf("Unexpected scope type: %T", scope)
	// 	}
	// 	// TODO(lukeshu): Verify that this check is
	// 	// correct; it seems backwards to me.
	// 	for _, s := range scopes {
	// 		logger.Debugf("verifying scope '%s'", s)
	// 		if !inArray(s, j.Arguments.Scopes) {
	// 			return errors.Errorf("Token scope %v is not in the policy", s)
	// 		}
	// 	}
	// } else {
	// 	logger.Debugf("No scopes to verify")
	// }

	// return nil
}

// func inArray(needle string, haystack []string) bool {
// 	for _, straw := range haystack {
// 		if straw == needle {
// 			return true
// 		}
// 	}
// 	return false
// }

func (c *OAuth2Filter) getSession(request *filterapi.FilterRequest) (*rfc6749.AuthorizationCodeClientSessionData, error) {
	// BS to leverage net/http's cookie-parsing
	r := &http.Request{
		Header: make(http.Header),
	}
	for k, v := range request.GetRequest().GetHttp().GetHeaders() {
		r.Header.Set(k, v)
	}

	cookie, err := r.Cookie(accessTokenCookie)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, err
	}
	//cleartext, err := c.cryptoDecryptAndVerify(ciphertext, []byte(accessTokenCookie))
	//if err != nil {
	//	return nil, err
	//}
	cleartext := ciphertext // TODO(lukeshu): Begone with cleartext := ciphertext
	var token ambassadorBearerToken
	err = json.Unmarshal(cleartext, &token)
	if err != nil {
		return nil, err
	}
	return token.SessionData, nil
}

func (c *OAuth2Filter) setSession(session *rfc6749.AuthorizationCodeClientSessionData) (*http.Cookie, error) {
	cleartext, err := json.Marshal(ambassadorBearerToken{session})
	if err != nil {
		return nil, err
	}
	//ciphertext, err := c.cryptoSignAndEncrypt(cleartext, []byte(accessTokenCookie))
	//if err != nil {
	//	return nil, err
	//}
	ciphertext := cleartext // TODO(lukeshu): Begone with ciphertext := cleartext
	return &http.Cookie{
		Name:  accessTokenCookie,
		Value: base64.RawURLEncoding.EncodeToString(ciphertext),
		// TODO(lukeshu): Verify that these are sane cookie parameters
		HttpOnly: true,
		Secure:   c.Spec.TLS(),
	}, nil
}

func (h *OAuth2Filter) signState(originalURL *url.URL, logger types.Logger) string {
	t := jwt.New(jwt.SigningMethodRS256)
	t.Claims = jwt.MapClaims{
		"exp":          time.Now().Add(h.Spec.StateTTL).Unix(), // time when the token will expire (10 minutes from now)
		"jti":          uuid.Must(uuid.NewV4(), nil).String(),  // a unique identifier for the token
		"iat":          time.Now().Unix(),                      // when the token was issued/created (now)
		"nbf":          0,                                      // time before which the token is not yet valid (2 minutes ago)
		"redirect_url": originalURL.String(),                   // original request url
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
