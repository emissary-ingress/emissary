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

	"github.com/datawire/ambassador/pkg/dlog"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	rfc6749client "github.com/datawire/apro/client/rfc6749"
	rfc6750client "github.com/datawire/apro/client/rfc6750"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/client/clientcommon"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/resourceserver"
	"github.com/datawire/apro/lib/filterapi"
	"github.com/datawire/apro/lib/filterapi/filterutil"
	"github.com/datawire/apro/lib/jwtsupport"
)

const (
	sessionBits = 256
	xsrfBits    = 256
	stateBits   = 256
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

func (c *OAuth2Client) Filter(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, redisClient *redis.Client, request *filterapi.FilterRequest) filterapi.FilterResponse {
	resp, cookies := c.filter(ctx, logger, httpClient, discovered, redisClient, request)
	if _, allowThrough := resp.(*filterapi.HTTPRequestModification); allowThrough && len(cookies) > 0 {
		resp = &filterapi.HTTPResponse{
			StatusCode: http.StatusTemporaryRedirect,
			Header: http.Header{
				"Location": {request.GetRequest().GetHttp().GetPath()},
			},
		}
	}
	for _, cookie := range cookies {
		// If len(cookies) > 0 and the response wasn't an HTTPResponse, then the
		// above block has already converted it to be an HTTPResponse; so no need
		// to check this type assertion.
		resp.(*filterapi.HTTPResponse).Header.Add("Set-Cookie", cookie.String())
	}
	return resp
}

func (c *OAuth2Client) filter(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, redisClient *redis.Client, request *filterapi.FilterRequest) (response filterapi.FilterResponse, cookies []*http.Cookie) {
	oauthClient, err := rfc6749client.NewAuthorizationCodeClient(
		c.Spec.ClientID,
		discovered.AuthorizationEndpoint,
		discovered.TokenEndpoint,
		rfc6749client.ClientPasswordHeader(c.Spec.ClientID, c.Spec.Secret),
		httpClient,
	)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			err, nil), nil
	}
	oauthClient.RegisterProtocolExtensions(rfc6750client.OAuthProtocolExtension)

	sessionInfo := &SessionInfo{c: c}
	var sessionErr error
	sessionInfo.sessionID, sessionInfo.xsrfToken, sessionInfo.sessionData, sessionErr = c.loadSession(redisClient, filterutil.GetHeader(request))
	defer func() {
		// If we have new session data, store it in to Redis, delete the old data
		// from Redis, and rev the cookies.
		if sessionInfo.sessionData == nil || !sessionInfo.sessionData.IsDirty() {
			return
		}
		newSessionID, err := randomString(sessionBits)
		if err != nil {
			response = middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "failed to generate session ID"), nil)
			return
		}
		newXsrfToken, err := randomString(xsrfBits)
		if err != nil {
			response = middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "failed to generate XSRF token"), nil)
			return
		}
		sessionDataBytes, err := json.Marshal(sessionInfo.sessionData)
		if err != nil {
			response = middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.New("failed to serialize session information"), nil)
			return
		}
		redisClient.PipeAppend("SET", "session:"+newSessionID, string(sessionDataBytes), "EX", int(sessionExpiry.Seconds()))
		redisClient.PipeAppend("SET", "session-xsrf:"+newSessionID, newXsrfToken, "EX", int(sessionExpiry.Seconds()))
		if sessionInfo.sessionID != "" {
			redisClient.PipeAppend("DEL", "session:"+sessionInfo.sessionID)
			redisClient.PipeAppend("DEL", "session-xsrf:"+sessionInfo.sessionID)
		}
		for err != redis.ErrPipelineEmpty {
			err = redisClient.PipeResp().Err
			if err != nil && err != redis.ErrPipelineEmpty {
				response = middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
					errors.Wrap(err, "redis"), nil)
				return
			}
		}
		useSessionCookies := *c.Spec.UseSessionCookies.Value
		if !c.Spec.UseSessionCookies.IfRequestHeader.Matches(filterutil.GetHeader(request)) {
			useSessionCookies = !useSessionCookies
		}
		maxAge := int(sessionExpiry.Seconds())
		if useSessionCookies {
			maxAge = 0 // unset
		}
		cookies = append(cookies,
			&http.Cookie{
				Name:  sessionInfo.c.sessionCookieName(),
				Value: newSessionID,

				// Expose the cookie to all paths on this host, not just directories of {{originalURL.Path}}.
				// This is important, because `/.ambassador/oauth2/redirection-endpoint` is probably not a
				// subdirectory of originalURL.Path.
				Path: "/",

				// Strictly match {{originalURL.Hostname}}.  Explicitly setting it to originalURL.Hostname()
				// would instead also match "*.{{originalURL.Hostname}}".
				Domain: "",

				// How long should the User-Agent retain the cookie?  If unset, it will expire at the end of the
				// "session" (when they close their browser).
				Expires: time.Time{}, // as a time (low precedence)
				MaxAge:  maxAge,      // as a duration (high precedence)

				// Whether to send the cookie for non-TLS requests.
				// TODO(lukeshu): consider using originalURL.Scheme
				Secure: sessionInfo.c.Spec.TLS(),

				// Do NOT expose the session cookie to JavaScript.
				HttpOnly: true,
			},
			&http.Cookie{
				Name:  sessionInfo.c.xsrfCookieName(),
				Value: newXsrfToken,

				// Expose the cookie to all paths on this host, not just directories of {{originalURL.Path}}.
				// This is important, because `/.ambassador/oauth2/redirection-endpoint` is probably not a
				// subdirectory of originalURL.Path.
				Path: "/",

				// Strictly match {{originalURL.Hostname}}.  Explicitly setting it to originalURL.Hostname()
				// would instead also match "*.{{originalURL.Hostname}}".
				Domain: "",

				// How long should the User-Agent retain the cookie?  If unset, it will expire at the end of the
				// "session" (when they close their browser).
				Expires: time.Time{}, // as a time (low precedence)
				MaxAge:  maxAge,      // as a duration (high precedence)

				// Whether to send the cookie for non-TLS requests.
				// TODO(lukeshu): consider using originalURL.Scheme
				Secure: sessionInfo.c.Spec.TLS(),

				// DO expose the XSRF cookie to JavaScript.
				HttpOnly: false,
			},
		)
	}()
	logger.Debugf("session data: %#v", sessionInfo.sessionData)
	var authorization http.Header
	switch {
	case sessionErr != nil:
		logger.Debugln("session status:", errors.Wrap(sessionErr, "no session"))
	case c.readXSRFCookie(filterutil.GetHeader(request)) != sessionInfo.xsrfToken:
		return middleware.NewErrorResponse(ctx, http.StatusForbidden,
			errors.New("XSRF protection"), nil), nil
	case sessionInfo.sessionData.CurrentAccessToken == nil:
		logger.Debugln("session status:", "non-authenticated session")
	default:
		logger.Debugln("session status:", "authenticated session")
		authorization, err = oauthClient.AuthorizationForResourceRequest(sessionInfo.sessionData, func() io.Reader {
			return strings.NewReader(request.GetRequest().GetHttp().GetBody())
		})
		if err == nil {
			// continue with (authorization != nil)
		} else if err == rfc6749client.ErrNoAccessToken {
			// This indicates a programming error; we've already checked that there is an access token.
			panic(err)
		} else if err == rfc6749client.ErrExpiredAccessToken {
			logger.Debugln("access token expired; continuing as if non-authenticated session")
			// continue with (authorization == nil); as if this `.CurrentAccessToken == nil`
		} else if _, ok := err.(*rfc6749client.UnsupportedTokenTypeError); ok {
			return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
				err, nil), nil
		} else {
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "unknown error"), nil), nil
		}
	}

	u, err := url.ParseRequestURI(request.GetRequest().GetHttp().GetPath())
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Wrapf(err, "could not parse URI: %q", request.GetRequest().GetHttp().GetPath()), nil), nil
	}
	switch u.Path {
	case "/.ambassador/oauth2/redirection-endpoint":
		if sessionInfo.sessionData == nil {
			// Regard this as an XSRF error
			return sessionInfo.handleUnauthenticatedProxyRequest(ctx, logger, httpClient, oauthClient, discovered, request), nil
		}
		var originalURL *url.URL
		if authorization != nil {
			logger.Debugln("already logged in; redirecting to original log-in-time URL")
			originalURL, err = ReadState(sessionInfo.sessionData.Request.State)
			if err != nil {
				// This should never happen--we read the state directly from what we
				// stored in Redis.
				return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
					errors.Wrapf(err, "invalid state"), nil), nil
			}
			if originalURL.Path == "/.ambassador/oauth2/redirection-endpoint" {
				// Avoid a redirect loop.  This "shouldn't" happen; we "shouldn't"
				// have generated (in .handleUnauthenticatedProxyRequest) a
				// sessionData.Request.State with this URL with this path.  However,
				// APro 0.8.0 and older could be tricked in to generating such a
				// .State.  So turn it in to an error page.  Of course, APro 0.8.0
				// used "/callback" instead of
				// "/.ambassador/oauth2/redirection-endpoint", so this even more
				// "shouldn't" happen now.
				return middleware.NewErrorResponse(ctx, http.StatusNotAcceptable,
					errors.New("no representation of /.ambassador/oauth2/redirection-endpoint resource"), nil), nil
			}
		} else {
			authorizationCode, err := oauthClient.ParseAuthorizationResponse(sessionInfo.sessionData, u)
			if err != nil {
				if errors.Is(err, rfc6749client.XSRFError("")) {
					return sessionInfo.handleUnauthenticatedProxyRequest(ctx, logger, httpClient, oauthClient, discovered, request), nil
				}
				return middleware.NewErrorResponse(ctx, http.StatusBadRequest,
					err, nil), nil
			}
			originalURL, err = ReadState(sessionInfo.sessionData.Request.State)
			if err != nil {
				// This should never happen--the state matched what we stored in Redis
				// (validated in .ParseAuthorizationResponse()).  For this to happen, either
				// (1) our Redis server was cracked, or (2) we generated an invalid state
				// when we submitted the authorization request.  Assuming that (2) is more
				// likely, that's an internal server issue.
				return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
					errors.Wrapf(err, "invalid state"), nil), nil
			}
			if err := oauthClient.AccessToken(sessionInfo.sessionData, authorizationCode); err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
					err, nil), nil
			}
		}
		logger.Debugf("redirecting user-agent to: %s", originalURL)
		return &filterapi.HTTPResponse{
			StatusCode: http.StatusSeeOther,
			Header: http.Header{
				"Location": {originalURL.String()},
			},
			Body: "",
		}, nil
	default:
		if authorization != nil {
			return sessionInfo.handleAuthenticatedProxyRequest(ctx, logger, httpClient, discovered, request, authorization), nil
		} else {
			return sessionInfo.handleUnauthenticatedProxyRequest(ctx, logger, httpClient, oauthClient, discovered, request), nil
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

func (sessionInfo *SessionInfo) handleAuthenticatedProxyRequest(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, request *filterapi.FilterRequest, authorization http.Header) filterapi.FilterResponse {
	return clientcommon.HandleAuthenticatedProxyRequest(dlog.WithLogger(ctx, logger), httpClient, discovered, request, authorization, sessionInfo.sessionData.CurrentAccessToken.Scope, sessionInfo.c.ResourceServer)
}

func (sessionInfo *SessionInfo) handleUnauthenticatedProxyRequest(ctx context.Context, logger dlog.Logger, httpClient *http.Client, oauthClient *rfc6749client.AuthorizationCodeClient, discovered *discovery.Discovered, request *filterapi.FilterRequest) filterapi.FilterResponse {
	// Use X-Forwarded-Proto instead of .GetScheme() to build the URL.
	// https://github.com/datawire/ambassador/issues/1581
	originalURL, err := url.ParseRequestURI(filterutil.GetHeader(request).Get("X-Forwarded-Proto") + "://" + request.GetRequest().GetHttp().GetHost() + request.GetRequest().GetHttp().GetPath())
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Wrap(err, "failed to construct URL"), nil)
	}

	// If we end up unauthorized at the redirection endpoint (because XSRF-protection
	// rejected the authorization), go ahead and dereference the alleged state.  This
	// is safe, because ruleForURL has already validated that state-derived-URL as
	// something that this filter handles.  This way we don't have to deal with the
	// redirection endpoint being bounced to multiple times.
	if originalURL.Path == "/.ambassador/oauth2/redirection-endpoint" {
		u, err := ReadState(originalURL.Query().Get("state"))
		if err == nil {
			originalURL = u
		}
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

	// Build the full request
	state, err := genState(originalURL)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			err, nil)
	}
	var authorizationRequestURI *url.URL
	authorizationRequestURI, sessionInfo.sessionData, err = oauthClient.AuthorizationRequest(
		sessionInfo.c.Spec.CallbackURL(),
		scope,
		state,
		sessionInfo.c.Spec.ExtraAuthorizationParameters,
	)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			err, nil)
	}

	if sessionInfo.c.Arguments.InsteadOfRedirect != nil && sessionInfo.c.Arguments.InsteadOfRedirect.IfRequestHeader.Matches(filterutil.GetHeader(request)) {
		sessionInfo.sessionData = nil // avoid setting pre-redirect session cookies
		if sessionInfo.c.Arguments.InsteadOfRedirect.HTTPStatusCode != 0 {
			return middleware.NewErrorResponse(ctx, sessionInfo.c.Arguments.InsteadOfRedirect.HTTPStatusCode,
				errors.New("session cookie is either missing, or refers to an expired or non-authenticated session"),
				nil)
		} else {
			ret, err := sessionInfo.c.RunFilters(sessionInfo.c.Arguments.InsteadOfRedirect.Filters, dlog.WithLogger(ctx, logger), request)
			if err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
					errors.Wrap(err, "insteadOfRedirect.filters"), nil)
			}
			return ret
		}
	} else {
		return &filterapi.HTTPResponse{
			// A 302 "Found" may or may not convert POST->GET.  We want
			// the UA to GET the Authorization URI, so we shouldn't use
			// 302 which may or may not do the right thing, but use 303
			// "See Other" which MUST convert to GET.
			StatusCode: http.StatusSeeOther,
			Header: http.Header{
				"Location": {authorizationRequestURI.String()},
			},
			Body: "",
		}
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

func genState(originalURL *url.URL) (string, error) {
	securePart, err := randomString(stateBits)
	if err != nil {
		return "", err
	}
	practicalPart := originalURL.String()
	return securePart + ":" + practicalPart, nil
}

func ReadState(state string) (*url.URL, error) {
	v0str, v0err := readStateV0(state)
	v1str, v1err := readStateV1(state)

	var urlStr string
	switch {
	case v0err == nil:
		urlStr = v0str
	case v1err == nil:
		urlStr = v1str
	default:
		return nil, v1err
	}
	return url.Parse(urlStr)
}

// readState for states created by AES <1.1.1
func readStateV0(state string) (string, error) {
	// Don't bother doing crypto validation on the JWT--it's
	// validated by doing a string-compare with the state stored
	// in Redis.
	claims := jwt.MapClaims{}
	if _, _, err := jwtsupport.SanitizeParseUnverified(new(jwt.Parser).ParseUnverified(state, &claims)); err != nil {
		return "", err
	}
	redirectURL, ok := claims["redirect_url"].(string)
	if !ok {
		return "", errors.New("malformed state")
	}
	return redirectURL, nil
}

// readState for states created by AES >=1.1.1
func readStateV1(state string) (string, error) {
	// then try parsing it as an AES 1.1.1+ state
	parts := strings.SplitN(state, ":", 2)
	if len(parts) != 2 {
		return "", errors.New("malformed state")
	}
	return parts[1], nil
}
