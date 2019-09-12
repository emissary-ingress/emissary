package oauth2handler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	rfc6749client "github.com/datawire/liboauth2/client/rfc6749"
	rfc6750client "github.com/datawire/liboauth2/client/rfc6750"
	rfc6750resourceserver "github.com/datawire/liboauth2/resourceserver/rfc6750"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/app/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwtsupport"
)

const (
	// How long Redis should remember sessions for, since "last use".
	sessionExpiry = 365 * 24 * time.Hour
)

// OAuth2Filter looks up the appropriate Tenant and Rule objects from
// the CRD Controller, and validates the signed JWT tokens when
// present in the request.  If the request Path is "/callback", it
// validates IDP requests and handles code exchange flow.
type OAuth2Filter struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	RedisPool  *pool.Pool
	QName      string
	Spec       crd.FilterOAuth2
	Arguments  crd.FilterOAuth2Arguments
}

func (c *OAuth2Filter) sessionCookieName() string {
	return "ambassador_session." + c.QName
}

func (c *OAuth2Filter) Filter(ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := middleware.GetLogger(ctx)
	httpClient := httpclient.NewHTTPClient(logger, c.Spec.MaxStale, c.Spec.InsecureTLS)

	discovered, err := Discover(httpClient, c.Spec, logger)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			errors.Wrap(err, "OIDC-discovery"), nil), nil
	}

	clientResponse := c.filterClient(ctx, logger, httpClient, discovered, request)
	switch clientResponse := clientResponse.(type) {
	case *filterapi.HTTPResponse:
		return clientResponse, nil
	case *filterapi.HTTPRequestModification:
		filterutil.ApplyRequestModification(request, clientResponse)
	default:
		panic(errors.Errorf("unexpexted filter response type %T", clientResponse))
	}

	resourceResponse := c.filterResourceServer(ctx, logger, httpClient, discovered, request)
	switch resourceResponse := resourceResponse.(type) {
	case *filterapi.HTTPResponse:
		if resourceResponse.StatusCode == http.StatusUnauthorized {
			// The upstream Resource Server returns 401 Unauthorized to the Client--the Client does NOT pass
			// 401 along to the User Agent; the User Agent is NOT using an RFC 7235-compatible
			// authentication scheme to talk to the Client; 401 would be inappropriate.
			//
			// Instead, wrap the 401 response in a 403 Forbidden response.
			return middleware.NewErrorResponse(ctx, http.StatusForbidden,
				errors.New("authorization rejected"),
				map[string]interface{}{
					"synthesized_upstream_response": resourceResponse,
				},
			), nil
		}
		return resourceResponse, nil
	case nil:
		// do nothing
	default:
		panic(errors.Errorf("unexpexted filter response type %T", resourceResponse))
	}

	return clientResponse, nil
}

// filterClient implements the OAuth Client part of the Filter.
func (c *OAuth2Filter) filterClient(ctx context.Context, logger types.Logger, httpClient *http.Client, discovered *Discovered, request *filterapi.FilterRequest) filterapi.FilterResponse {
	redisClient, err := c.RedisPool.Get()
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			errors.Wrap(err, "Redis"), nil)
	}

	oauthClient, err := rfc6749client.NewAuthorizationCodeClient(
		c.Spec.ClientID,
		discovered.AuthorizationEndpoint,
		discovered.TokenEndpoint,
		rfc6749client.ClientPasswordHeader(c.Spec.ClientID, c.Spec.Secret),
		httpClient,
	)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			err, nil)
	}
	oauthClient.RegisterProtocolExtensions(rfc6750client.OAuthProtocolExtension)

	sessionID, sessionData, sessionErr := c.loadSession(redisClient, request)
	defer func() {
		if sessionData != nil {
			err := c.saveSession(redisClient, sessionID, sessionData)
			if err != nil {
				// TODO(lukeshu): Letting FilterMux recover() this panic() and generate an error message
				// isn't the *worst* way of handling this error.
				panic(err)
			}
		}
		c.RedisPool.Put(redisClient)
	}()
	logger.Debugf("session data: %#v", sessionData)
	switch {
	case sessionErr != nil:
		logger.Debugln("session status:", errors.Wrap(sessionErr, "no session"))
	case sessionData.CurrentAccessToken == nil:
		logger.Debugln("session status:", "non-authenticated session")
	default:
		logger.Debugln("session status:", "authenticated session")
		authorization, err := oauthClient.AuthorizationForResourceRequest(sessionData, func() io.Reader {
			return strings.NewReader(request.GetRequest().GetHttp().GetBody())
		})
		if err == nil {
			// Validate the scope values we were granted.  This really belongs in
			// .filterResourceServer(), but our content-agnostic half-Resource-Server
			// doesn't have a good way of verifying that it was granted arbitrary scope
			// values.
			if err := c.validateScope(sessionData.CurrentAccessToken.Scope); err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusForbidden,
					errors.Wrap(err, "insufficient privilege scope"), nil)
			}
			// OK, the scope check passed, inject the Authorization header, and continue
			// to the Resource Server half.
			ret := &filterapi.HTTPRequestModification{}
			for k, vs := range authorization {
				for _, v := range vs {
					ret.Header = append(ret.Header, &filterapi.HTTPHeaderReplaceValue{
						Key:   k,
						Value: v,
					})
				}
			}
			return ret
		} else if err == rfc6749client.ErrNoAccessToken {
			// This indicates a programming error; we've check that there is an access token.
			panic(err)
		} else if err == rfc6749client.ErrExpiredAccessToken {
			logger.Debugln("access token expired; continuing as if non-authenticated session")
			// continue as if this `.CurrentAccessToken == nil`
		} else if _, ok := err.(*rfc6749client.UnsupportedTokenTypeError); ok {
			return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
				err, nil)
		} else {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "unknown error"), nil)
		}
	}

	u, err := url.ParseRequestURI(request.GetRequest().GetHttp().GetPath())
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Wrapf(err, "could not parse URI: %q", request.GetRequest().GetHttp().GetPath()), nil)
	}
	switch u.Path {
	case "/callback":
		if sessionData == nil {
			return middleware.NewErrorResponse(ctx, http.StatusForbidden,
				errors.Errorf("no %q cookie", c.sessionCookieName()), nil)
		}
		authorizationCode, err := oauthClient.ParseAuthorizationResponse(sessionData, u)
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusBadRequest,
				err, nil)
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
				errors.Wrapf(err, "invalid state"), nil)
		}
		if err := oauthClient.AccessToken(sessionData, authorizationCode); err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
				err, nil)
		}
		logger.Debugf("redirecting user-agent to: %s", originalURL)
		return &filterapi.HTTPResponse{
			StatusCode: http.StatusSeeOther,
			Header: http.Header{
				"Location": {originalURL},
			},
			Body: "",
		}
	default:
		// Use X-Forwarded-Proto instead of .GetScheme() to build the URL.
		// https://github.com/datawire/ambassador/issues/1581
		originalURL, err := url.ParseRequestURI(filterutil.GetHeader(request).Get("X-Forwarded-Proto") + "://" + request.GetRequest().GetHttp().GetHost() + request.GetRequest().GetHttp().GetPath())
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "failed to construct URL"), nil)
		}

		// Build the scope
		scope := make(rfc6749client.Scope, len(c.Arguments.Scopes))
		for _, s := range c.Arguments.Scopes {
			scope[s] = struct{}{}
		}

		scope["openid"] = struct{}{} // TODO(lukeshu): More carefully consider always asserting OIDC

		// You'd be tempted to include "offline_access" here, but if offline access isn't
		// allowed, some Authorization Servers (including Google, Keycloak, and UAA) will
		// reject the entire authorization request, instead of simply issuing an Access
		// Token without a Refresh Token.
		//
		//scope["offline_access"] = struct{}{}

		// Build the sessionID and the associated cookie
		sessionID, err = randomString(256) // NB: Do NOT re-declare sessionID
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "failed to generate session ID"), nil)
		}
		cookie := &http.Cookie{
			Name:  c.sessionCookieName(),
			Value: sessionID,

			// Expose the cookie to all paths on this host, not just directories of {{originalURL.Path}}.
			// This is important, because `/callback` is probably not a subdirectory of originalURL.Path.
			Path: "/",

			// Strictly match {{originalURL.Hostname}}.  Explicitly setting it to originalURL.Hostname()
			// would instead also "*.{{originalURL.Hostname}}".
			Domain: "",

			// How long should the User-Agent retain the cookie?  If unset, it will expire at the end of the
			// "session" (when they close their browser).
			Expires: time.Time{},                                // as a time (low precedence)
			MaxAge:  int((10 * 365 * 24 * time.Hour).Seconds()), // as a duration (high precedence)

			// Whether to send the cookie for non-TLS requests.
			// TODO(lukeshu): consider using originalURL.Scheme
			Secure: c.Spec.TLS(),

			// Don't expose the cookie to JavaScript.
			HttpOnly: true,
		}

		// Build the full request
		var authorizationRequestURI *url.URL
		authorizationRequestURI, sessionData, err = oauthClient.AuthorizationRequest( // NB: Do NOT re-declare sessionData
			c.Spec.CallbackURL(),
			scope,
			c.signState(originalURL, logger),
		)
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				err, nil)
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
		}
	}
}

// filterResourceServer implements the OAuth Resource Server part of the Filter.
func (c *OAuth2Filter) filterResourceServer(ctx context.Context, logger types.Logger, httpClient *http.Client, discovered *Discovered, request *filterapi.FilterRequest) filterapi.FilterResponse {
	token := rfc6750resourceserver.GetFromHeader(filterutil.GetHeader(request))
	if err := c.validateAccessToken(token, discovered, httpClient, logger); err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadRequest, err, nil)
	}
	return nil
}

func (j *OAuth2Filter) validateAccessToken(token string, discovered *Discovered, httpClient *http.Client, logger types.Logger) error {
	switch j.Spec.AccessTokenValidation {
	case "auto":
		claims, err := j.parseJWT(token, discovered)
		if err == nil {
			return j.validateJWT(claims, discovered, logger)
		}
		logger.Debugln("rejecting JWT validation; falling back to UserInfo Endpoint validation:", err)
		fallthrough
	case "userinfo":
		return j.validateAccessTokenUserinfo(token, discovered, httpClient, logger)
	case "jwt":
		claims, err := j.parseJWT(token, discovered)
		if err != nil {
			return err
		}
		return j.validateJWT(claims, discovered, logger)
	}
	panic("not reached")
}

func (j *OAuth2Filter) validateAccessTokenUserinfo(token string, discovered *Discovered, httpClient *http.Client, logger types.Logger) error {
	req, err := http.NewRequest("GET", discovered.UserInfoEndpoint.String(), nil)
	if err != nil {
		return err
	}
	rfc6750client.AddToHeader(token, req.Header)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return errors.Errorf("token validation through userinfo endpoint failed: HTTP %d", res.StatusCode)
	}
	return nil
}

func (j *OAuth2Filter) parseJWT(token string, discovered *Discovered) (jwt.MapClaims, error) {
	jwtParser := jwt.Parser{
		ValidMethods: []string{
			// Any of the RSA algs supported by jwt-go
			"RS256",
			"RS384",
			"RS512",
		},
		SkipClaimsValidation: true,
	}

	var claims jwt.MapClaims
	_, err := jwtsupport.SanitizeParse(jwtParser.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
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
	}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (j *OAuth2Filter) validateScope(actual rfc6749client.Scope) error {
	desired := make(rfc6749client.Scope, len(j.Arguments.Scopes))
	for _, s := range j.Arguments.Scopes {
		desired[s] = struct{}{}
	}
	var missing []string
	for scopeValue := range desired {
		if scopeValue == "offline_access" {
			continue
		}
		if _, ok := actual[scopeValue]; !ok {
			missing = append(missing, scopeValue)
		}
	}
	switch len(missing) {
	case 0:
		return nil
	case 1:
		return errors.Errorf("missing required scope value: %q", missing[0])
	default:
		return errors.Errorf("missing required scope values: %q", missing)
	}
}

func (j *OAuth2Filter) validateJWT(claims jwt.MapClaims, discovered *Discovered, logger types.Logger) error {
	// Validate 'exp', 'iat', and 'nbf' claims.
	if err := claims.Valid(); err != nil {
		return err
	}

	// Validate 'aud' claim.
	//if !claims.VerifyAudience(j.Spec.Audience, false) {
	//	return errors.Errorf("token has wrong audience: token=%#v expected=%q", claims["aud"], j.Spec.Audience)
	//}

	// Validate 'iss' claim.
	if !claims.VerifyIssuer(discovered.Issuer, false) {
		return errors.Errorf("token has wrong issuer: token=%#v expected=%q", claims["iss"], discovered.Issuer)
	}

	// Validate 'scopes' claim (draft standard).
	// https://www.iana.org/assignments/jwt/jwt.xhtml
	// https://tools.ietf.org/html/draft-ietf-oauth-token-exchange-16#section-4.2
	switch scopeClaim := claims["scope"].(type) {
	case nil:
		logger.Debugf("No scope to verify")
	case string: // proposed standard; most Authorization Servers do this
		if err := j.validateScope(rfc6749client.ParseScope(scopeClaim)); err != nil {
			return errors.Wrap(err, "token has wrong scope")
		}
	case []interface{}: // UAA does this
		actual := make(rfc6749client.Scope, len(scopeClaim))
		for _, scopeValue := range scopeClaim {
			switch scopeValue := scopeValue.(type) {
			case string:
				actual[scopeValue] = struct{}{}
			default:
				logger.Warningf("Unexpected scope[n] type: %T", scopeValue)
			}
		}
		if err := j.validateScope(actual); err != nil {
			return errors.Wrap(err, "token has wrong scope")
		}
	default:
		logger.Warningf("Unexpected scope type: %T", scopeClaim)
	}

	return nil
}

func randomString(bits int) (string, error) {
	buf := make([]byte, (bits+1)/8)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (c *OAuth2Filter) loadSession(redisClient *redis.Client, request *filterapi.FilterRequest) (sessionID string, sessionData *rfc6749client.AuthorizationCodeClientSessionData, err error) {
	// BS to leverage net/http's cookie-parsing
	r := &http.Request{
		Header: filterutil.GetHeader(request),
	}

	// get the sessionID from the cookie
	cookie, err := r.Cookie(c.sessionCookieName())
	if cookie == nil {
		return "", nil, err
	}
	sessionID = cookie.Value

	// get the sessionData from Redis
	sessionDataBytes, err := redisClient.Cmd("GET", "session:"+sessionID).Bytes()
	if err != nil {
		return "", nil, err
	}
	sessionData = new(rfc6749client.AuthorizationCodeClientSessionData)
	if err := json.Unmarshal(sessionDataBytes, sessionData); err != nil {
		return "", nil, err
	}

	return sessionID, sessionData, nil
}

func (c *OAuth2Filter) saveSession(redisClient *redis.Client, sessionID string, sessionData *rfc6749client.AuthorizationCodeClientSessionData) error {
	if sessionData.IsDirty() {
		sessionDataBytes, err := json.Marshal(sessionData)
		if err != nil {
			return err
		}
		if err := redisClient.Cmd("SET", "session:"+sessionID, string(sessionDataBytes)).Err; err != nil {
			return err
		}
	}
	if err := redisClient.Cmd("EXPIRE", "session:"+sessionID, int64(sessionExpiry.Seconds())).Err; err != nil {
		return err
	}
	return nil
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

	token, err := jwtsupport.SanitizeParse(jwt.Parse(state, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return "", errors.Errorf("unexpected signing method %v", t.Header["redirect_url"])
		}
		return pubkey, nil
	}))

	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return "", errors.New("state token validation failed")
	}

	return claims["redirect_url"].(string), nil
}
