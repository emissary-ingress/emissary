package authorization_code_client

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

	rfc6749client "github.com/datawire/apro/client/rfc6749"
	rfc6750client "github.com/datawire/apro/client/rfc6750"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/client/clientcommon"
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

	RunFilters func(filters []crd.FilterReference, ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error)
}

func (c *OAuth2Client) sessionCookieName() string {
	return "ambassador_session." + c.QName
}

func (c *OAuth2Client) xsrfCookieName() string {
	return "ambassador_xsrf." + c.QName
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
	sessionInfo.sessionID, sessionInfo.xsrfToken, sessionInfo.sessionData, sessionErr = c.loadSession(redisClient, filterutil.GetHeader(request))
	defer func() {
		if sessionInfo.sessionData != nil {
			err := c.saveSession(redisClient, sessionInfo.sessionID, sessionInfo.xsrfToken, sessionInfo.sessionData)
			if err != nil {
				// TODO(lukeshu): Letting FilterMux recover() this panic() and generate an error message
				// isn't the *worst* way of handling this error.
				panic(err)
			}
		}
	}()
	logger.Debugf("session data: %#v", sessionInfo.sessionData)
	var authorization http.Header
	switch {
	case sessionErr != nil:
		logger.Debugln("session status:", errors.Wrap(sessionErr, "no session"))
	case c.readXSRFCookie(filterutil.GetHeader(request)) != sessionInfo.xsrfToken:
		return middleware.NewErrorResponse(ctx, http.StatusForbidden,
			errors.New("XSRF protection"), nil)
	case sessionInfo.sessionData.CurrentAccessToken == nil:
		logger.Debugln("session status:", "non-authenticated session")
	default:
		logger.Debugln("session status:", "authenticated session")
		authorization, err = oauthClient.AuthorizationForResourceRequest(sessionInfo.sessionData, func() io.Reader {
			return strings.NewReader(request.GetRequest().GetHttp().GetBody())
		})
		if err == nil {
			// continue with (authrorization != nil)
		} else if err == rfc6749client.ErrNoAccessToken {
			// This indicates a programming error; we've already checked that there is an access token.
			panic(err)
		} else if err == rfc6749client.ErrExpiredAccessToken {
			logger.Debugln("access token expired; continuing as if non-authenticated session")
			// continue with (authorization == nil); as if this `.CurrentAccessToken == nil`
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
		var originalURL string
		if authorization != nil {
			logger.Debugln("already logged in; redirecting to original log-in-time URL")
			originalURL, err = checkState(sessionInfo.sessionData.Request.State, c.PublicKey)
			if err != nil {
				// This should never happen--we read the state directly from what we
				// stored into Redis.
				return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
					errors.Wrapf(err, "invalid state"), nil)
			}
			u, _ := url.Parse(originalURL)
			if u.Path == "/callback" {
				// Avoid a redirect loop.  This "shouldn't" happen; we "shouldn't"
				// have generated (in .handleUnauthenticatedProxyRequest) a
				// sessionData.Request.State with this URL with this path.  However,
				// APro 0.8.0 and older could be tricked in to generating such a
				// .State.  So turn it in to an error page.
				return middleware.NewErrorResponse(ctx, http.StatusNotAcceptable,
					errors.New("no representation of /callback resource"), nil)
			}
		} else {
			authorizationCode, err := oauthClient.ParseAuthorizationResponse(sessionInfo.sessionData, u)
			if err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusBadRequest,
					err, nil)
			}
			originalURL, err = checkState(sessionInfo.sessionData.Request.State, c.PublicKey)
			if err != nil {
				// This should never happen--the state matched what we stored in Redis
				// (validated in .ParseAuthorizationResponse()).  For this to happen, either
				// (1) our Redis server was cracked, or (2) we generated an invalid state
				// when we submitted the authorization request.  Assuming that (2) is more
				// likely, that's an internal server issue.
				return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
					errors.Wrapf(err, "invalid state"), nil)
			}
			if err := oauthClient.AccessToken(sessionInfo.sessionData, authorizationCode); err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
					err, nil)
			}
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
		if authorization != nil {
			return sessionInfo.handleAuthenticatedProxyRequest(ctx, logger, httpClient, discovered, request, authorization)
		} else {
			return sessionInfo.handleUnauthenticatedProxyRequest(ctx, logger, httpClient, oauthClient, discovered, request)
		}
	}
}

func (c *OAuth2Client) ServeHTTP(w http.ResponseWriter, r *http.Request, ctx context.Context, discovered *discovery.Discovered, redisClient *redis.Client) {
	switch r.URL.Path {
	case "/.ambassador/oauth2/logout":
		sessionID, xsrfToken, _, err := c.loadSession(redisClient, r.Header)
		if err != nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusForbidden, // XXX: error code?
				errors.Wrap(err, "no session"), nil)
			return
		}

		if c.readXSRFCookie(r.Header) != xsrfToken {
			middleware.ServeErrorResponse(w, ctx, http.StatusForbidden,
				errors.New("XSRF protection"), nil)
			return
		}

		if r.PostFormValue("_xsrf") != xsrfToken {
			middleware.ServeErrorResponse(w, ctx, http.StatusForbidden,
				errors.New("XSRF protection"), nil)
			return
		}

		if discovered.EndSessionEndpoint == nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusNotImplemented,
				errors.Errorf("identify provider %q does not support OIDC-session section 5 logout", c.Spec.AuthorizationURL), nil)
			return
		}

		//query := discovered.EndSessionEndpoint.Query()
		//query.Set("id_token_hint", "TODO") // TODO: only RECOMMENDED; would require us to track ID Tokens better
		//query.Set("post_logout_redirect_uri", "TODO") // TODO: only OPTIONAL; having a good UX around this probably requires us to support OIDC-registration
		//query.Set("state", "TODO") // TODO: only OPTIONAL; only does something if "post_logout_redirect_uri"

		// TODO: Don't do the delete until the post_logout_redirect_uri is hit?
		if err := redisClient.Cmd("DEL", "session:"+sessionID, "session-xsrf:"+sessionID).Err; err != nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusInternalServerError,
				err, nil)
			return
		}

		http.Redirect(w, r, discovered.EndSessionEndpoint.String(), http.StatusSeeOther)
	default:
		http.NotFound(w, r)
	}
}

type SessionInfo struct {
	c           *OAuth2Client
	sessionID   string
	xsrfToken   string
	sessionData *rfc6749client.AuthorizationCodeClientSessionData
}

func (sessionInfo *SessionInfo) handleAuthenticatedProxyRequest(ctx context.Context, logger types.Logger, httpClient *http.Client, discovered *discovery.Discovered, request *filterapi.FilterRequest, authorization http.Header) filterapi.FilterResponse {
	return clientcommon.HandleAuthenticatedProxyRequest(middleware.WithLogger(ctx, logger), httpClient, discovered, request, authorization, sessionInfo.sessionData.CurrentAccessToken.Scope, sessionInfo.c.ResourceServer)
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
	sessionCookie := &http.Cookie{
		Name:  sessionInfo.c.sessionCookieName(),
		Value: sessionInfo.sessionID,

		// Expose the cookie to all paths on this host, not just directories of {{originalURL.Path}}.
		// This is important, because `/callback` is probably not a subdirectory of originalURL.Path.
		Path: "/",

		// Strictly match {{originalURL.Hostname}}.  Explicitly setting it to originalURL.Hostname()
		// would instead also match "*.{{originalURL.Hostname}}".
		Domain: "",

		// How long should the User-Agent retain the cookie?  If unset, it will expire at the end of the
		// "session" (when they close their browser).
		Expires: time.Time{},                                // as a time (low precedence)
		MaxAge:  int((10 * 365 * 24 * time.Hour).Seconds()), // as a duration (high precedence)

		// Whether to send the cookie for non-TLS requests.
		// TODO(lukeshu): consider using originalURL.Scheme
		Secure: sessionInfo.c.Spec.TLS(),

		// Do NOT expose the session cookie to JavaScript.
		HttpOnly: true,
	}

	sessionInfo.xsrfToken, err = randomString(256)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Wrap(err, "failed to generate XSRF token"), nil)
	}
	xsrfCookie := &http.Cookie{
		Name:  sessionInfo.c.xsrfCookieName(),
		Value: sessionInfo.xsrfToken,

		// Expose the cookie to all paths on this host, not just directories of {{originalURL.Path}}.
		// This is important, because `/callback` is probably not a subdirectory of originalURL.Path.
		Path: "/",

		// Strictly match {{originalURL.Hostname}}.  Explicitly setting it to originalURL.Hostname()
		// would instead also match "*.{{originalURL.Hostname}}".
		Domain: "",

		// How long should the User-Agent retain the cookie?  If unset, it will expire at the end of the
		// "session" (when they close their browser).
		Expires: time.Time{},                                // as a time (low precedence)
		MaxAge:  int((10 * 365 * 24 * time.Hour).Seconds()), // as a duration (high precedence)

		// Whether to send the cookie for non-TLS requests.
		// TODO(lukeshu): consider using originalURL.Scheme
		Secure: sessionInfo.c.Spec.TLS(),

		// DO expose the XSRF cookie to JavaScript.
		HttpOnly: false,
	}

	// Build the full request
	var authorizationRequestURI *url.URL
	authorizationRequestURI, sessionInfo.sessionData, err = oauthClient.AuthorizationRequest(
		sessionInfo.c.Spec.CallbackURL(),
		scope,
		sessionInfo.c.signState(originalURL, logger),
		sessionInfo.c.Spec.ExtraAuthorizationParameters,
	)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			err, nil)
	}

	if sessionInfo.c.Arguments.InsteadOfRedirect != nil {
		noRedirect := sessionInfo.c.Arguments.InsteadOfRedirect.IfRequestHeader.Matches(filterutil.GetHeader(request))
		if noRedirect {
			if sessionInfo.c.Arguments.InsteadOfRedirect.HTTPStatusCode != 0 {
				ret := middleware.NewErrorResponse(ctx, sessionInfo.c.Arguments.InsteadOfRedirect.HTTPStatusCode,
					errors.New("session cookie is either missing, or refers to an expired or non-authenticated session"),
					nil)
				ret.Header["Set-Cookie"] = []string{
					sessionCookie.String(),
					xsrfCookie.String(),
				}
				return ret
			} else {
				ret, err := sessionInfo.c.RunFilters(sessionInfo.c.Arguments.InsteadOfRedirect.Filters, middleware.WithLogger(ctx, logger), request)
				if err != nil {
					return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
						errors.Wrap(err, "insteadOfRedirect.filters"), nil)
				}
				return ret
			}
		}
	}

	return &filterapi.HTTPResponse{
		// A 302 "Found" may or may not convert POST->GET.  We want
		// the UA to GET the Authorization URI, so we shouldn't use
		// 302 which may or may not do the right thing, but use 303
		// "See Other" which MUST convert to GET.
		StatusCode: http.StatusSeeOther,
		Header: http.Header{
			"Set-Cookie": {
				sessionCookie.String(),
				xsrfCookie.String(),
			},
			"Location": {authorizationRequestURI.String()},
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

func (c *OAuth2Client) readXSRFCookie(requestHeader http.Header) string {
	// BS to leverage net/http's cookie-parsing
	r := &http.Request{
		Header: requestHeader,
	}

	cookie, err := r.Cookie(c.xsrfCookieName())
	if cookie == nil || err != nil {
		return ""
	}
	return cookie.Value
}

func (c *OAuth2Client) loadSession(redisClient *redis.Client, requestHeader http.Header) (sessionID, xsrfToken string, sessionData *rfc6749client.AuthorizationCodeClientSessionData, err error) {
	// BS to leverage net/http's cookie-parsing
	r := &http.Request{
		Header: requestHeader,
	}

	// get the sessionID from the cookie
	cookie, err := r.Cookie(c.sessionCookieName())
	if cookie == nil {
		return "", "", nil, err
	}
	sessionID = cookie.Value

	// get the xsrf token from Redis
	xsrfToken, err = redisClient.Cmd("GET", "session-xsrf:"+sessionID).Str()
	if err != nil {
		return "", "", nil, err
	}

	// get the sessionData from Redis
	sessionDataBytes, err := redisClient.Cmd("GET", "session:"+sessionID).Bytes()
	if err != nil {
		return "", "", nil, err
	}
	sessionData = new(rfc6749client.AuthorizationCodeClientSessionData)
	if err := json.Unmarshal(sessionDataBytes, sessionData); err != nil {
		return "", "", nil, err
	}

	return sessionID, xsrfToken, sessionData, nil
}

func (c *OAuth2Client) saveSession(redisClient *redis.Client, sessionID, xsrfToken string, sessionData *rfc6749client.AuthorizationCodeClientSessionData) error {
	if sessionData.IsDirty() {
		sessionDataBytes, err := json.Marshal(sessionData)
		if err != nil {
			return err
		}
		if err := redisClient.Cmd("SET", "session:"+sessionID, string(sessionDataBytes)).Err; err != nil {
			return err
		}
		if err := redisClient.Cmd("SET", "session-xsrf:"+sessionID, xsrfToken).Err; err != nil {
			return err
		}
	}
	if err := redisClient.Cmd("EXPIRE", "session:"+sessionID, int64(sessionExpiry.Seconds())).Err; err != nil {
		return err
	}
	if err := redisClient.Cmd("EXPIRE", "session-xsrf:"+sessionID, int64(sessionExpiry.Seconds())).Err; err != nil {
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
