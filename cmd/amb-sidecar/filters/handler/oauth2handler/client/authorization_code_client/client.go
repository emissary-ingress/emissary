package authorization_code_client

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// OAuth2MultiDomainInfo is a place to save information for the multidomain redirection
// part of OAuth2.
type OAuth2MultiDomainInfo struct {
	OriginalURL string
	StatusCode  int
	Cookies     []*http.Cookie
}

// Figure out which root a given scheme and authority matches -- if any.
//
// This method is a little weird because we need to return the index, not just
// the struct, so we return an int. If no root is found, we return -1.

func findMatchingOrigin(spec crd.FilterOAuth2, logger dlog.Logger, scheme string, authority string) int {
	idx := -1

	logger.Debugf("AuthCode: looking for scheme %s, authority %s", scheme, authority)

	var root crd.Origin
	var i int

	for i, root = range spec.ProtectedOrigins {
		if (root.Origin.Scheme == scheme) && (root.Origin.Host == authority) {
			idx = i
			break
		}
	}

	if idx < 0 {
		logger.Debugf("AuthCode: %s://%s matches no origin", scheme, authority)
	} else {
		logger.Debugf("AuthCode: %s://%s is origin %d (%s)", scheme, authority, i, root.Origin.String())
	}

	return idx
}

// OAuth2Client implements the OAuth Client part of the Filter.
// OAuth2Client implements the OAuth 2.0 "Client" part of the OAuth Filter.  An
// OAuth2Client has two main entry points: 'Filter' and 'ServeHTTP'.  Both are called
// by the upper-level machinery in oauth2handler/oauth2handler.go.
//
// 'Filter' gets called (at ext_authz-time) for a request that either
//  - matched a FilterPolicy that told it to use this filter for auth, or
//  - hit the special-case '/.ambassador/oauth2/redirection-endpoint' override
// either way; ruleForURL() returned a rule containing this filter.matched our
// configured FilterPolicy, or tripped one of the special cases for OAuth (i.e.,
// ruleForURL returned this filter).  The 'Filter' method gets handed the request,
// and it must return a response indicating what Ambassador should do with it
// (continue unmodiefied, continue with modification, or reject it).
//
// 'ServeHTTP' gets called (at Mapping-time) for HTTP endpoints that we're the
// Mapping for; ruleForURL is hard-coded to return nil for those endpoints; so Filter
// is never called on them.  Right now, that's just the logout endpoint
// (/.ambassador/oauth2/logout) -- anything else is a 404 error.  (For historical
// reasons, the redirection endpoint is handled in Filter(), though it would make
// more sense to handle it in ServeHTTP()).
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

// Filter is called with a request, and must return a response indicating what to
// do with the request.
func (c *OAuth2Client) Filter(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, redisClient *redis.Client, request *filterapi.FilterRequest) filterapi.FilterResponse {
	// Our filter method (lowercase) does the heavy lifting here. Call it to get a response
	// and/or some cookies.
	resp, cookies := c.filter(ctx, logger, httpClient, discovered, redisClient, request)

	// Is the response an HTTPRequestModification that allows the request
	// through, or an HTTPResponse that denies the request?
	_, allowThrough := resp.(*filterapi.HTTPRequestModification)

	logger.Debugf("AuthCode %s Filter: allowThrough %v, len(cookies) %d", c.QName, allowThrough, len(cookies))

	if allowThrough && len(cookies) > 0 {
		// Unfortunately, we have no mechanism by which we can set cookies if
		// we're allowing the request through.  So, instead we'll deny it,
		// and have the deny response (1) set the cookies, and (2) redirect
		// back to the same page.
		resp = &filterapi.HTTPResponse{
			// 307 "Temporary Redirect" (unlike other redirect codes)
			// does not allow the user-agent to change from POST to GET
			// when following the redirect--we don't want to discard any
			// form data being submitted!
			StatusCode: http.StatusTemporaryRedirect,
			Header: http.Header{
				"Location": {request.GetRequest().GetHttp().GetPath()},
			},
		}
	}

	if len(cookies) > 0 {
		// If here, then if we used to have an HTTPRequestModification, we've turned it
		// into a redirect... so we could be getting a direct response from the filter,
		// or we could be getting a redirect to the IdP, or we could've converted an
		// HTTPRequestModification into a redirect above.
		//
		// In any of these cases, it's important to understand here that we _won't_ have
		// cookies to set unless something has changed -- and if something has changed,
		// we'll need to change that something across all of our protected origins. So.
		// Do we have more than one of those?
		//
		// XXX
		// This is annoying code duplication with lowercase-f filter.

		if len(c.Spec.ProtectedOrigins) > 1 {
			// Yup, so a couple of things need to happen here. First off, generate a new
			// multidomain-session-ID...
			mdSessionID, err := randomString(sessionBits)

			if err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
					errors.Wrap(err, "failed to generate session for multidomain"), nil)
			}

			// ...and find the original URI for this request.
			//
			// XXX It annoys me to do this again when we had to do it in lowercase-filter
			// already.
			requestURL, err := filterutil.GetURL(request)

			if err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
					errors.Wrapf(err, "could not parse request URI"), nil)
			}

			// OK -- assume that we're in the direct-response case, which means we'll
			// have no Location header and we're going to issue a StatusTemporaryRedirect
			// to make sure our cookies get set _and_ that we don't throw away form data
			// by allowing the user-agent to downgrade from POST to GET.
			statusCode := http.StatusTemporaryRedirect
			originalURL := requestURL.String()

			location := resp.(*filterapi.HTTPResponse).Header.Get("Location")

			if location != "" {
				// Whoops, this isn't the direct-response case at all! This is actually
				// a redirect to the IdP or a converted HTTPRequestModification, so we
				// should save the status & location from the resp, not the stuff above.
				statusCode = resp.(*filterapi.HTTPResponse).StatusCode
				originalURL = location
			}

			// OK -- build an Oauth2MultiDomainInfo...
			mdInfo := OAuth2MultiDomainInfo{
				OriginalURL: originalURL,
				StatusCode:  statusCode,
				Cookies:     cookies,
			}

			// ...and get it saved in Redis.
			mdInfoBytes, err := json.Marshal(mdInfo)
			if err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
					errors.Wrap(err, "failed to serialize mdInfo"), nil)
			}

			logger.Debugf("AuthCode %s Filter save mdInfo: oauth2-mdinfo:%s, %#v", c.QName, mdSessionID, mdInfo)

			err = redisClient.Cmd("SET", "oauth2-mdinfo:"+mdSessionID, string(mdInfoBytes), "EX", int(sessionExpiry.Seconds())).Err
			if err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
					errors.Wrap(err, "failed to save mdInfo to Redis"), nil)
			}

			// The target URL is the redirection URL for root 0...
			targetURL := c.Spec.RedirectionURL(0)

			// ...but we need to include a query parameter with the filter's qualified name and
			// the mdSessionID.
			q := targetURL.Query() // This should always be empty, of course...
			q.Set("code", c.QName+":"+mdSessionID)
			targetURL.RawQuery = q.Encode()

			// Finally! Redirect to our first protected root for more.
			resp = &filterapi.HTTPResponse{
				// 307 "Temporary Redirect" (unlike other redirect codes)
				// does not allow the user-agent to change from POST to GET
				// when following the redirect--we don't want to discard any
				// form data being submitted!
				StatusCode: http.StatusTemporaryRedirect,
				Header: http.Header{
					"Location": {targetURL.String()},
				},
			}
		}

		// And, of course, let's make sure the cookies actually get set.
		for i, cookie := range cookies {
			// If len(cookies) > 0 and the response wasn't an HTTPResponse, then the
			// above block has already converted it to be an HTTPResponse; so no need
			// to check this type assertion.

			if i == 0 {
				logger.Debugf("AuthCode %s Filter: redirecting to %s", c.QName, resp.(*filterapi.HTTPResponse).Header.Get("Location"))
			}

			logger.Debugf("AuthCode %s Filter: setting cookie %s=%s", c.QName, cookie.Name, cookie.Value)
			resp.(*filterapi.HTTPResponse).Header.Add("Set-Cookie", cookie.String())
		}
	}

	return resp
}

// 'filter' (lowercase) is where the lion's share of the work happens: we need to figure
// out if this is a good request, or not, and tell the upper layers what to do, whatever
// is up.
func (c *OAuth2Client) filter(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, redisClient *redis.Client, request *filterapi.FilterRequest) (response filterapi.FilterResponse, cookies []*http.Cookie) {
	// Parse out the URI for this request...
	requestURL, err := filterutil.GetURL(request)

	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			errors.Wrapf(err, "could not parse request URI"), nil), nil
	}

	logger.Debugf("AuthCode %s filter firing for %s", c.QName, requestURL.String())

	// Which root is this?
	rootIndex := findMatchingOrigin(c.Spec, logger, requestURL.Scheme, requestURL.Host)

	// If there's no matching root at all... uh... that's a problem.
	if rootIndex < 0 {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.New(fmt.Sprintf("filter misconfiguration: %s://%s is not a valid origin for %s",
				requestURL.Scheme, requestURL.Host, c.QName)), nil), nil
	}

	// Start by setting up the OAuth client.  (Note that in "ClientPasswordHeader",
	// "client" refers to the OAuth client, not the end user's User-Agent -- which is
	// to say, this is Ambassador identifying itself to the IdP.)
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

	// RFC6750 specifies the "bearer token" access-token-type.  That's the
	// access-token-type that we use!
	oauthClient.RegisterProtocolExtensions(rfc6750client.OAuthProtocolExtension)

	// Load our session from Redis.
	sessionInfo := &SessionInfo{c: c}
	sessionErr := c.loadSession(sessionInfo, redisClient, logger, filterutil.GetHeader(request))

	// Whenever this method returns, we should update our session info in Redis, if
	// need be.
	defer func() {
		newCookies, err := c.saveSession(redisClient, sessionInfo, logger, filterutil.GetHeader(request), rootIndex)

		if err != nil {
			// Change the response of all of 'filter'.
			response = middleware.NewErrorResponse(ctx, http.StatusInternalServerError, err, nil)
		} else {
			cookies = append(cookies, newCookies...)
		}
	}()

	// OK, back to processing the request.  (Remeber, the last thing we did was load
	// the session info from Redis.)
	//
	// The "happy-path" is that we end up with a set of authorization headers to
	// inject in to the request before allowing it through.
	var authorization http.Header

	switch {
	case sessionErr != nil:
		// No (valid) session data. This isn't that big a deal, but log it for
		// debugging, then fall past the switch to treat this like an
		// unauthenticated session.

		logger.Debugln("session status:", errors.Wrap(sessionErr, "no session"))
	case c.readXSRFCookie(filterutil.GetHeader(request)) != sessionInfo.xsrfToken:
		// Yikes!  Someone is trying to hack our users!
		return middleware.NewErrorResponse(ctx, http.StatusForbidden,
			errors.New("XSRF protection"), nil), nil

	case sessionInfo.sessionData.CurrentAccessToken == nil:
		// This is a non-authenticated session; we've previously redirected the
		// user to the IdP, but they never completed the authorization flow.
		logger.Debugln("session status:", "non-authenticated session")

	default:
		// This is a fully-authenticated session, so try to update the access token.
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
	// If we made it this far: We haven't encountered any errors, and we
	// may-or-may-not have 'authorization' to inject in to the request.

	switch requestURL.Path {
	case "/.ambassador/oauth2/redirection-endpoint":
		// OK, this is a diversion from train-of-thought you should have when reading this
		// method--because this doesn't belong in this method, and belongs in ServeHTTP();
		// but is here for historical reasons, and the only reason for it to stay here is
		// that it would be effort to change it.

		// At the redirection-endpoint, whenever we encounter an XSRF error, we should
		// handle it not by returning an error page, but by redirecting-to-IdP (as we would
		// for an unauthenticated request; i.e. call .handleUnauthenticatedProxyRequest()).
		// https://github.com/datawire/apro/issues/999

		if sessionInfo.sessionData == nil {
			// How we "should" get here is that the IdP redirects to here after a
			// successful authentication, which of course requires a session.
			//
			// Regard lack of a session as an XSRF error.
			logger.Debugln("no sessionData for session ID", sessionInfo.sessionID)
			return sessionInfo.handleUnauthenticatedProxyRequest(ctx, logger, httpClient, oauthClient, discovered, request), nil
		}

		// Since we have non-nil sessionData, we should have an original URL.
		var originalURL *url.URL
		originalURL, err = ReadState(sessionInfo.sessionData.Request.State)

		if err != nil {
			// This should never happen--we read the state directly from what we
			// stored in Redis, so either our Redis server got hacked or we _completely_
			// botched the bookkeeping around the session and its authorization. Assuming
			// that the error is more likely, that's an internal server error.

			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrapf(err, "invalid state"), nil), nil
		}

		// Finally, we should only ever arrive here on root 0. If we got here via some other root,
		// bail -- this is almost certainly a bad configuration.

		if rootIndex != 0 {
			root0 := c.Spec.ProtectedOrigins[0].Origin

			badRootErr := errors.New(fmt.Sprintf("received redirection to %s://%s instead of %s", requestURL.Scheme, requestURL.Host, root0.String()))

			return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
				badRootErr, nil), nil
		}

		// OK. Do we already have an authorization?
		if authorization != nil {
			logger.Debugln("already logged in; original log-in-time URL %v", originalURL)

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
			// First time back here after auth! Let's see if we have an access token.
			logger.Debugln("No authorization code, trying to parse one")
			authorizationCode, err := oauthClient.ParseAuthorizationResponse(sessionInfo.sessionData, requestURL)

			if err != nil {
				if errors.As(err, new(rfc6749client.XSRFError)) {
					return sessionInfo.handleUnauthenticatedProxyRequest(ctx, logger, httpClient, oauthClient, discovered, request), nil
				}
				return middleware.NewErrorResponse(ctx, http.StatusBadRequest,
					err, nil), nil
			}

			// Given the auth code, get an access token.
			err = oauthClient.AccessToken(sessionInfo.sessionData, authorizationCode)

			if err != nil {
				return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
					err, nil), nil
			}
		}

		// OK, assume that we'll redirect with StatusSeeOther to our original URL...
		statusCode := http.StatusSeeOther
		targetURL := originalURL

		// OK, all's well, let's redirect to the target URL.
		logger.Debugf("redirecting user-agent with %d to %s", statusCode, targetURL.String())

		return &filterapi.HTTPResponse{
			StatusCode: statusCode,
			Header: http.Header{
				"Location": {targetURL.String()},
			},
			Body: "",
		}, nil

	default:
		// This is the "any path other than the redirection-endpoint" case; back to the
		// "proper" flow through this method.
		//
		// Do the right thing depending on whether we're authenticated or not.
		if authorization != nil {
			return sessionInfo.handleAuthenticatedProxyRequest(ctx, logger, httpClient, discovered, request, authorization), nil
		} else {
			return sessionInfo.handleUnauthenticatedProxyRequest(ctx, logger, httpClient, oauthClient, discovered, request), nil
		}
	}
}

// ServeHTTP gets called to handle the special `/.ambassador/oauth2/**` endpoints.  Because of
// overrides hard-coded in to ruleForURL(), no Filters have modified or intercepted these requests.
func (c *OAuth2Client) ServeHTTP(w http.ResponseWriter, r *http.Request, ctx context.Context, discovered *discovery.Discovered, redisClient *redis.Client) {
	logger := dlog.GetLogger(r.Context())

	logger.Debugf("AuthCode %s ServeHTTP firing for %s - %s", c.QName, r.Host, r.URL.Path)

	switch r.URL.Path {
	case "/.ambassador/oauth2/logout":
		// Hey look, it's the logout path! Clobber our session, but first make sure no
		// funny stuff is going on.

		sessionInfo := &SessionInfo{c: c}
		err := c.loadSession(sessionInfo, redisClient, dlog.GetLogger(r.Context()), r.Header)

		if err != nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusForbidden, // XXX: error code?
				errors.Wrap(err, "no session"), nil)
			return
		}

		if c.readXSRFCookie(r.Header) != sessionInfo.xsrfToken {
			middleware.ServeErrorResponse(w, ctx, http.StatusForbidden,
				errors.New("XSRF protection"), nil)
			return
		}

		if r.PostFormValue("_xsrf") != sessionInfo.xsrfToken {
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
		err = c.deleteSession(redisClient, sessionInfo)

		if err != nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusInternalServerError, err, nil)
			return
		}

		http.Redirect(w, r, discovered.EndSessionEndpoint.String(), http.StatusSeeOther)

	case "/.ambassador/oauth2/multicookie":
		// Hey look, it's the multidomain cookie handler! Do we have a code?

		params := r.URL.Query()
		mdState := params.Get("code")

		if mdState == "" {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.New("missing mdState"), nil)
			return
		}

		parts := strings.SplitN(mdState, ":", 2)
		if len(parts) != 2 {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.New("malformed mdState"), nil)
			return
		}

		mdSessionID := parts[1]
		logger.Debugf("Auth MultiCookie: mdSessionID %s", mdSessionID)

		// OK, we have a code -- try to find it in Redis.
		mdInfoBytes, err := redisClient.Cmd("GET", "oauth2-mdinfo:"+mdSessionID).Bytes()
		if err != nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusInternalServerError,
				errors.Wrap(err, "could not load mdinfo"), nil)
			return
		}

		// So far so good. Unmarshal the cookies...
		mdInfo := new(OAuth2MultiDomainInfo)
		err = json.Unmarshal(mdInfoBytes, mdInfo)
		if err != nil {
			middleware.ServeErrorResponse(w, ctx, http.StatusInternalServerError,
				errors.Wrap(err, "could not parse mdinfo"), nil)
			return
		}

		// Which root is this?
		//
		// XXX We can get the Host from r, but neither r nor r.URL have the scheme directly -- so
		// we look for the Envoy X-Forwarded-Proto header instead. (This might be another instance
		// of https://github.com/datawire/ambassador/issues/1581)

		scheme := r.Header.Get("X-Forwarded-Proto")
		authority := r.Host

		rootIndex := findMatchingOrigin(c.Spec, logger, scheme, authority)

		if rootIndex < 0 {
			middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
				errors.New(fmt.Sprintf("filter misconfiguration: %s://%s is not a valid origin for %s",
					scheme, authority, c.QName)), nil)
			return
		}

		// OK, all's well. Set our cookies.
		for _, cookie := range mdInfo.Cookies {
			logger.Debugf("Auth MultiCookie: set %s", cookie.String())

			// Make sure the domain matches the subdomain setting for this root.
			if c.Spec.AllowSubdomains(rootIndex) {
				cookie.Domain = c.Spec.GetOrigin(rootIndex)
			} else {
				cookie.Domain = ""
			}

			http.SetCookie(w, cookie)
		}

		// ...then, if we have more domains, redirect to the next.
		nextRoot := rootIndex + 1

		if nextRoot < len(c.Spec.ProtectedOrigins) {
			targetURL := c.Spec.RedirectionURL(nextRoot)

			q := targetURL.Query() // This should always be empty, of course...
			q.Set("code", c.QName+":"+mdSessionID)
			targetURL.RawQuery = q.Encode()

			logger.Debugf("Auth MultiCookie: redirect to %s", targetURL.String())

			// 307 "Temporary Redirect" (unlike other redirect codes)
			// does not allow the user-agent to change from POST to GET
			// when following the redirect--we don't want to discard any
			// form data being submitted!
			http.Redirect(w, r, targetURL.String(), http.StatusTemporaryRedirect)
		} else {
			// We're done. Finally. Delete the mdInfo...
			// logger.Debugf("Auth MultiCookie: delete mdInfo: %s", mdSessionID)
			// _ = redisClient.Cmd("DEL", "oauth2-mdinfo:"+mdSessionID)

			// ...then redirect to the orginal URL.
			logger.Debugf("Auth MultiCookie: redirect to %s", mdInfo.OriginalURL)

			http.Redirect(w, r, mdInfo.OriginalURL, mdInfo.StatusCode)
		}

	default:
		// Anything other than the logout URL is an error here.
		http.NotFound(w, r)
	}
}

// Why prithee isn't this up at the top of the file?
type SessionInfo struct {
	c           *OAuth2Client
	sessionID   string
	xsrfToken   string
	sessionData *rfc6749client.AuthorizationCodeClientSessionData
	activeRoot  int
}

// handleAuthenticatedProxyRequest is pretty "simple" in that we can just farm it out
// to the clientcommon package.
func (sessionInfo *SessionInfo) handleAuthenticatedProxyRequest(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, request *filterapi.FilterRequest, authorization http.Header) filterapi.FilterResponse {
	return clientcommon.HandleAuthenticatedProxyRequest(dlog.WithLogger(ctx, logger), httpClient, discovered, request, authorization, sessionInfo.sessionData.CurrentAccessToken.Scope, sessionInfo.c.ResourceServer)
}

// handleUnauthenticatedProxyRequest is less simple -- we have to do that ourselves.
func (sessionInfo *SessionInfo) handleUnauthenticatedProxyRequest(ctx context.Context, logger dlog.Logger, httpClient *http.Client, oauthClient *rfc6749client.AuthorizationCodeClient, discovered *discovery.Discovered, request *filterapi.FilterRequest) filterapi.FilterResponse {
	// Start by grabbing the original URL.
	originalURL, err := filterutil.GetURL(request)

	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
			errors.Wrap(err, "failed to construct URL"), nil)
	}

	// If we end up unauthorized at the redirection endpoint (because XSRF-protection
	// rejected the authorization), go ahead and dereference the alleged state.
	//
	// This is safe, because ruleForURL has already validated that state-derived-URL
	// as something that this filter handles, by virtue of the fact that it decided to
	// call this filter.  Cogito, ergo sum.
	//
	// If we don't dereference the alleged state here, then the alternative would be
	// to modify ruleForURL to recurse on itself; otherwise we'd get to the URL
	//
	//    redirection-endpoint?state={redirect to "redirection-endpoint?state={redirect to original URL}"}
	//
	// and because ruleForURL doesn't recurse on itself when extracting the
	// destination from `state`, ruleForURL wouldn't choose the rule for "original
	// URL", instead returning nil, so then the request falls through to the fallback
	// ServeHTTP in handler.go which would serve an HTTP 400 response.
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
		// Time for the initial redirection to OAuth.
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

func (c *OAuth2Client) loadSession(sessionInfo *SessionInfo, redisClient *redis.Client, logger dlog.Logger, requestHeader http.Header) error {
	// BS to leverage net/http's cookie-parsing
	r := &http.Request{
		Header: requestHeader,
	}

	// get the sessionID from the cookie
	cookie, err := r.Cookie(c.sessionCookieName())
	if cookie == nil {
		logger.Debugf("AuthCode loadSession: no session cookie %s", c.sessionCookieName())
		return err
	}
	sessionInfo.sessionID = cookie.Value

	logger.Debugf("AuthCode loadSession: got ID %s from cookie %s", sessionInfo.sessionID, c.sessionCookieName())

	// get the xsrf token from Redis
	sessionInfo.xsrfToken, err = redisClient.Cmd("GET", "session-xsrf:"+sessionInfo.sessionID).Str()
	if err != nil {
		return err
	}

	// get the sessionData from Redis
	sessionDataBytes, err := redisClient.Cmd("GET", "session:"+sessionInfo.sessionID).Bytes()
	if err != nil {
		return err
	}
	sessionInfo.sessionData = new(rfc6749client.AuthorizationCodeClientSessionData)
	err = json.Unmarshal(sessionDataBytes, sessionInfo.sessionData)
	if err != nil {
		return err
	}

	return nil
}

func (c *OAuth2Client) saveSession(redisClient *redis.Client, sessionInfo *SessionInfo, logger dlog.Logger, requestHeader http.Header, rootIndex int) ([]*http.Cookie, error) {
	// If we have no new session data, we can just bail here.
	if sessionInfo == nil || sessionInfo.sessionData == nil || !sessionInfo.sessionData.IsDirty() {
		logger.Debugf("AuthCode saveSession: nothing to save")
		return nil, nil
	}

	// OK, we have some new stuff. Store it in Redis, delete the old data
	// from Redis, and rev the cookies.
	//
	// Generate a new session ID...
	newSessionID, err := randomString(sessionBits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate session ID")
	}

	// ...and a new XSRF token...
	newXsrfToken, err := randomString(xsrfBits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate XSRF token")
	}

	// ...and then marshal the session data.
	sessionDataBytes, err := json.Marshal(sessionInfo.sessionData)
	if err != nil {
		return nil, errors.New("failed to serialize session information")
	}

	// Set up a pipeline of Redis updates...
	logger.Debugf("AuthCode saveSession: writing session & XSRF for ID %s", newSessionID)
	redisClient.PipeAppend("SET", "session:"+newSessionID, string(sessionDataBytes), "EX", int(sessionExpiry.Seconds()))
	redisClient.PipeAppend("SET", "session-xsrf:"+newSessionID, newXsrfToken, "EX", int(sessionExpiry.Seconds()))

	if sessionInfo.sessionID != "" {
		logger.Debugf("AuthCode saveSession: deleting old ID %s", sessionInfo.sessionID)

		redisClient.PipeAppend("DEL", "session:"+sessionInfo.sessionID)
		redisClient.PipeAppend("DEL", "session-xsrf:"+sessionInfo.sessionID)
	}

	// ...then make sure everything in the pipeline works.
	for err != redis.ErrPipelineEmpty {
		err = redisClient.PipeResp().Err
		if err != nil && err != redis.ErrPipelineEmpty {
			return nil, errors.Wrap(err, "redis failed")
		}
	}

	// Finally, update the sessionInfo with the new stuff.
	sessionInfo.sessionID = newSessionID
	sessionInfo.xsrfToken = newXsrfToken

	return c.getSessionCookies(requestHeader, sessionInfo, rootIndex), nil
}

func (c *OAuth2Client) getSessionCookies(requestHeader http.Header, sessionInfo *SessionInfo, rootIndex int) (cookies []*http.Cookie) {
	// OK, build up some cookies.
	//
	// Note that "useSessionCookies" is about whether or not we use cookies that
	// expire when the browser is closed (session cookies) or cookies that expire
	// at a particular time, irrespective of how long the browser has been open
	// (uh... non-session cookies, I guess?). It's not about whether we use cookies
	// at all -- we always set a cookie, the thing that changes is the expiry.

	useSessionCookies := *c.Spec.UseSessionCookies.Value
	if !c.Spec.UseSessionCookies.IfRequestHeader.Matches(requestHeader) {
		useSessionCookies = !useSessionCookies
	}

	maxAge := int(sessionExpiry.Seconds())

	if useSessionCookies {
		maxAge = 0 // unset
	}

	// Assume that we can do an exact domain match...
	domain := ""

	if c.Spec.AllowSubdomains(rootIndex) {
		domain = c.Spec.GetOrigin(rootIndex)
	}

	cookies = []*http.Cookie{
		&http.Cookie{
			Name:  sessionInfo.c.sessionCookieName(),
			Value: sessionInfo.sessionID,

			// Expose the cookie to all paths on this host, not just directories of {{originalURL.Path}}.
			// This is important, because `/.ambassador/oauth2/redirection-endpoint` is probably not a
			// subdirectory of originalURL.Path.
			Path: "/",

			// Strictly match {{originalURL.Hostname}}.  Explicitly setting it to originalURL.Hostname()
			// would instead also match "*.{{originalURL.Hostname}}".
			Domain: domain,

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
			Value: sessionInfo.xsrfToken,

			// Expose the cookie to all paths on this host, not just directories of {{originalURL.Path}}.
			// This is important, because `/.ambassador/oauth2/redirection-endpoint` is probably not a
			// subdirectory of originalURL.Path.
			Path: "/",

			// Strictly match {{originalURL.Hostname}}.  Explicitly setting it to originalURL.Hostname()
			// would instead also match "*.{{originalURL.Hostname}}".
			Domain: domain,

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
	}

	return cookies
}

func (c *OAuth2Client) deleteSession(redisClient *redis.Client, sessionInfo *SessionInfo) error {
	return redisClient.Cmd(
		"DEL",
		"session:"+sessionInfo.sessionID,
		"session-xsrf:"+sessionInfo.sessionID,
	).Err
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
