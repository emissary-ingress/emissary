package ropc_client

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/datawire/ambassador/pkg/dlog"
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
)

const (
	// How long Redis should remember sessions for, since "last use".
	sessionExpiry = 7 * 24 * time.Hour
)

// OAuth2Client implements the OAuth Client part of the Filter.
type OAuth2Client struct {
	QName     string
	Spec      crd.FilterOAuth2
	Arguments crd.FilterOAuth2Arguments

	ResourceServer *resourceserver.OAuth2ResourceServer
}

// An OAuth2Client has two main entry points. Both are called by the upper-level
// machinery in oauth2handler/oauth2handler.go.
//
// Filter gets called for a request that matched an override rule in the upper-level
// machinery (ie, ruleForURL returned this filter). In practice, for us, this means
// that it's called when this filter is configured to do auth, and a request for
// a resource with a "normal" URL (not /.ambassador/) arrives. The Filter method gets
// handed the request, and it must return a response indicating what Ambassador should
// do with it (let it continue, modify it, reject it, etc).
//
// ServeHTTP gets called for requests that do _not_ match an override rule (ie,
// ruleForURL returned nil). In practice, for us, that's always an error.

// Filter is called with with a request, and must return a response indicating what to
// do with the request.
func (c *OAuth2Client) Filter(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, redisClient *redis.Client, request *filterapi.FilterRequest) filterapi.FilterResponse {
	// Build the scope that the user wants...
	scope := make(rfc6749client.Scope, len(c.Arguments.Scopes))
	for _, s := range c.Arguments.Scopes {
		scope[s] = struct{}{}
	}

	// ...get the credentials from the request...
	username := filterutil.GetHeader(request).Get("X-Ambassador-Username")
	password := filterutil.GetHeader(request).Get("X-Ambassador-Password")

	if (username == "") || (password == "") {
		// C'mon, people.
		return middleware.NewErrorResponse(ctx, http.StatusForbidden,
			errors.Errorf("username and password are required"), nil)
	}

	// ...and set up the OAuth2 client. (Note that in "ClientPasswordHeader",
	// "client" refers to the client of the IdP -- which is to say, us. So this
	// is us given our password to the IdP.
	oauthClient, err := rfc6749client.NewResourceOwnerPasswordCredentialsClient(
		discovered.TokenEndpoint,
		rfc6749client.ClientPasswordHeader(c.Spec.ClientID, c.Spec.Secret),
		httpClient,
	)

	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			err, nil)
	}

	// The RFC6750 client is all about using a Bearer token when correctly
	// authenticated. We want that.
	oauthClient.RegisterProtocolExtensions(rfc6750client.OAuthProtocolExtension)

	// OK. For our session key, hash the username and the password.
	sessionHash := sha256.New()
	_, _ = sessionHash.Write([]byte(username))
	_, _ = sessionHash.Write([]byte("--"))
	_, _ = sessionHash.Write([]byte(password))

	sessionKey := fmt.Sprintf("%x", sessionHash.Sum(nil))

	// ...then set up session info.
	sessionData, err := c.loadSession(redisClient, sessionKey)

	if err != nil {
		logger.Debugln("session status:", errors.Wrap(err, "no session"))
	}

	// Whenever we leave the scope of this method, we should update our session info
	// in Redis, if need be.
	defer func() {
		if sessionData != nil {
			err := c.saveSession(redisClient, sessionKey, sessionData)
			if err != nil {
				// TODO(lukeshu): Letting FilterMux recover() this panic() and generate an error message
				// isn't the *worst* way of handling this error.
				panic(err)
			}
		}
	}()

	// Loop forever, in cause we find that there's no access token and we need to cycle to
	// reset the session data.
	for {
		// OK, if we have no session data...
		if sessionData == nil {
			// ...then go make a new authorization request.
			sessionData, err = oauthClient.AuthorizationRequest(username, password, scope)

			if err != nil {
				// Bzzzt. So what went wrong?
				_, isAuthErr := err.(rfc6749client.TokenErrorResponse)

				if isAuthErr {
					// They supplied bad credentials. Forbidden!!
					return middleware.NewErrorResponse(ctx, http.StatusForbidden,
						err, nil)
				}

				// Something else went wrong. Bad gateway! No token for you!
				return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
					err, nil)
			}
		}

		// Here we already have a session (possibly one we just created), and we need
		// to make sure it's properly authorized.
		authorization, err := oauthClient.AuthorizationForResourceRequest(sessionData, func() io.Reader {
			return strings.NewReader(request.GetRequest().GetHttp().GetBody())
		})

		switch err {
		case nil:
			// All is well. Handle this as an authenticated request.
			return c.handleAuthenticatedProxyRequest(ctx, logger, httpClient, discovered, request, authorization, sessionData)

		case rfc6749client.ErrNoAccessToken, rfc6749client.ErrExpiredAccessToken:
			// Bzzt. Wipe the session and try again.
			sessionData = nil
			continue

		default:
			// This "shouldn't happen", but it did. Hand back an error.
			if _, ok := err.(*rfc6749client.UnsupportedTokenTypeError); ok {
				return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
					err, nil)
			}
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "unknown error"), nil)
		}
	}
}

// ServeHTTP gets called for requests that do _not_ match an override rule (ie,
// ruleForURL returned nil). In practice, for us, that's always an error, but we
// can at least be slightly polite with the answer to the logout endpoint.
func (c *OAuth2Client) ServeHTTP(w http.ResponseWriter, r *http.Request, ctx context.Context, discovered *discovery.Discovered, redisClient *redis.Client) {
	switch r.URL.Path {
	case "/.ambassador/oauth2/logout":
		middleware.ServeErrorResponse(w, ctx, http.StatusBadRequest,
			errors.New("realm does not support logout"),
			nil)
	default:
		http.NotFound(w, r)
	}
}

// handleAuthenticatedProxyRequest is a utility method to update the bearer token for
// a successfully-authenticated request.
func (c *OAuth2Client) handleAuthenticatedProxyRequest(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, request *filterapi.FilterRequest, authorization http.Header, sessionData *rfc6749client.ResourceOwnerPasswordCredentialsClientSessionData) filterapi.FilterResponse {
	return clientcommon.HandleAuthenticatedProxyRequest(dlog.WithLogger(ctx, logger), httpClient, discovered, request, authorization, sessionData.CurrentAccessToken.Scope, c.ResourceServer)
}

// Load our session from Redis.
func (c *OAuth2Client) loadSession(redisClient *redis.Client, sessionKey string) (*rfc6749client.ResourceOwnerPasswordCredentialsClientSessionData, error) {
	sessionID := "ropc-" + url.QueryEscape(sessionKey)

	sessionDataBytes, err := redisClient.Cmd("GET", "session:"+sessionID).Bytes()
	if err != nil {
		return nil, err
	}
	sessionData := new(rfc6749client.ResourceOwnerPasswordCredentialsClientSessionData)
	if err := json.Unmarshal(sessionDataBytes, sessionData); err != nil {
		return nil, err
	}
	return sessionData, nil
}

// Save our session to Redis.
func (c *OAuth2Client) saveSession(redisClient *redis.Client, sessionKey string, sessionData *rfc6749client.ResourceOwnerPasswordCredentialsClientSessionData) error {
	sessionID := "ropc-" + url.QueryEscape(sessionKey)

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
