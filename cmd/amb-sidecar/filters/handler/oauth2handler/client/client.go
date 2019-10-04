package client

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
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	rfc6749client "github.com/datawire/liboauth2/client/rfc6749"
	rfc6750client "github.com/datawire/liboauth2/client/rfc6750"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/resourceserver"
	"github.com/datawire/apro/cmd/amb-sidecar/types"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwtsupport"
)

const (
	// How long Redis should remember sessions for, since "last use".
	sessionExpiry = 365 * 24 * time.Hour
)

// OAuth2Client implements the OAuth Client part of the Filter.
type OAuth2Client struct {
	QName     string
	Spec      crd.FilterOAuth2
	Arguments crd.FilterOAuth2Arguments

	ResourceServer *resourceserver.OAuth2ResourceServer

	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func (c *OAuth2Client) sessionCookieName() string {
	return "ambassador_session." + c.QName
}

func (c *OAuth2Client) Filter(ctx context.Context, logger types.Logger, httpClient *http.Client, discovered *discovery.Discovered, redisClient *redis.Client, request *filterapi.FilterRequest) filterapi.FilterResponse {
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

	sessionInfo := &SessionInfo{c: c}
	var sessionErr error
	sessionInfo.sessionID, sessionInfo.sessionData, sessionErr = c.loadSession(redisClient, request)
	defer func() {
		if sessionInfo.sessionData != nil {
			err := c.saveSession(redisClient, sessionInfo.sessionID, sessionInfo.sessionData)
			if err != nil {
				// TODO(lukeshu): Letting FilterMux recover() this panic() and generate an error message
				// isn't the *worst* way of handling this error.
				panic(err)
			}
		}
	}()
	logger.Debugf("session data: %#v", sessionInfo.sessionData)
	switch {
	case sessionErr != nil:
		logger.Debugln("session status:", errors.Wrap(sessionErr, "no session"))
	case sessionInfo.sessionData.CurrentAccessToken == nil:
		logger.Debugln("session status:", "non-authenticated session")
	default:
		logger.Debugln("session status:", "authenticated session")
		authorization, err := oauthClient.AuthorizationForResourceRequest(sessionInfo.sessionData, func() io.Reader {
			return strings.NewReader(request.GetRequest().GetHttp().GetBody())
		})
		if err == nil {
			return sessionInfo.handleAuthenticatedProxyRequest(ctx, logger, httpClient, discovered, request, authorization)
		} else if err == rfc6749client.ErrNoAccessToken {
			// This indicates a programming error; we've already checked that there is an access token.
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
		if sessionInfo.sessionData == nil {
			return middleware.NewErrorResponse(ctx, http.StatusForbidden,
				errors.Errorf("no %q cookie", c.sessionCookieName()), nil)
		}
		authorizationCode, err := oauthClient.ParseAuthorizationResponse(sessionInfo.sessionData, u)
		if err != nil {
			return middleware.NewErrorResponse(ctx, http.StatusBadRequest,
				err, nil)
		}
		originalURL, err := checkState(sessionInfo.sessionData.Request.State, c.PublicKey)
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
		if err := oauthClient.AccessToken(sessionInfo.sessionData, authorizationCode); err != nil {
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
		return sessionInfo.handleUnauthenticatedProxyRequest(ctx, logger, httpClient, oauthClient, discovered, request)
	}
}

type SessionInfo struct {
	c           *OAuth2Client
	sessionID   string
	sessionData *rfc6749client.AuthorizationCodeClientSessionData
}

func (sessionInfo *SessionInfo) handleAuthenticatedProxyRequest(ctx context.Context, logger types.Logger, httpClient *http.Client, discovered *discovery.Discovered, request *filterapi.FilterRequest, authorization http.Header) filterapi.FilterResponse {
	addAuthorization := &filterapi.HTTPRequestModification{}
	for k, vs := range authorization {
		for _, v := range vs {
			addAuthorization.Header = append(addAuthorization.Header, &filterapi.HTTPHeaderReplaceValue{
				Key:   k,
				Value: v,
			})
		}
	}
	filterutil.ApplyRequestModification(request, addAuthorization)

	resourceResponse := sessionInfo.c.ResourceServer.Filter(ctx, logger, httpClient, discovered, request, sessionInfo.sessionData.CurrentAccessToken.Scope)
	if resourceResponse == nil {
		// nil means to send the same request+authorization to the upstream service, so tell
		// Envoy to add the authorization to the request.
		return addAuthorization
	} else if resourceResponse, typeOK := resourceResponse.(*filterapi.HTTPResponse); typeOK && resourceResponse.StatusCode == http.StatusUnauthorized {
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
		)
	} else {
		// Otherwise, just return the upstream resource server's response
		return resourceResponse
	}
}

func (sessionInfo *SessionInfo) handleUnauthenticatedProxyRequest(ctx context.Context, logger types.Logger, httpClient *http.Client, oauthClient *rfc6749client.AuthorizationCodeClient, discovered *discovery.Discovered, request *filterapi.FilterRequest) filterapi.FilterResponse {
	// Use X-Forwarded-Proto instead of .GetScheme() to build the URL.
	// https://github.com/datawire/ambassador/issues/1581
	originalURL, err := url.ParseRequestURI(filterutil.GetHeader(request).Get("X-Forwarded-Proto") + "://" + request.GetRequest().GetHttp().GetHost() + request.GetRequest().GetHttp().GetPath())
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Wrap(err, "failed to construct URL"), nil)
	}

	// Build the scope
	scope := make(rfc6749client.Scope, len(sessionInfo.c.Arguments.Scopes))
	for _, s := range sessionInfo.c.Arguments.Scopes {
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
	sessionInfo.sessionID, err = randomString(256)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Wrap(err, "failed to generate session ID"), nil)
	}
	cookie := &http.Cookie{
		Name:  sessionInfo.c.sessionCookieName(),
		Value: sessionInfo.sessionID,

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
		Secure: sessionInfo.c.Spec.TLS(),

		// Don't expose the cookie to JavaScript.
		HttpOnly: true,
	}

	// Build the full request
	var authorizationRequestURI *url.URL
	authorizationRequestURI, sessionInfo.sessionData, err = oauthClient.AuthorizationRequest(
		sessionInfo.c.Spec.CallbackURL(),
		scope,
		sessionInfo.c.signState(originalURL, logger),
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

func randomString(bits int) (string, error) {
	buf := make([]byte, (bits+1)/8)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (c *OAuth2Client) loadSession(redisClient *redis.Client, request *filterapi.FilterRequest) (sessionID string, sessionData *rfc6749client.AuthorizationCodeClientSessionData, err error) {
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

func (c *OAuth2Client) saveSession(redisClient *redis.Client, sessionID string, sessionData *rfc6749client.AuthorizationCodeClientSessionData) error {
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

func (c *OAuth2Client) signState(originalURL *url.URL, logger types.Logger) string {
	t := jwt.New(jwt.SigningMethodRS256)
	t.Claims = jwt.MapClaims{
		"exp":          time.Now().Add(c.Spec.StateTTL).Unix(), // time when the token will expire (10 minutes from now)
		"jti":          uuid.Must(uuid.NewV4(), nil).String(),  // a unique identifier for the token
		"iat":          time.Now().Unix(),                      // when the token was issued/created (now)
		"nbf":          0,                                      // time before which the token is not yet valid (2 minutes ago)
		"redirect_url": originalURL.String(),                   // original request url
	}

	k, err := t.SignedString(c.PrivateKey)
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
