package client_credentials_client

import (
	"context"
	"encoding/json"
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

func (c *OAuth2Client) Filter(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, redisClient *redis.Client, request *filterapi.FilterRequest) filterapi.FilterResponse {
	// Build the scope
	scope := make(rfc6749client.Scope, len(c.Arguments.Scopes))
	for _, s := range c.Arguments.Scopes {
		scope[s] = struct{}{}
	}

	// Get the credentials from the request
	clientID := filterutil.GetHeader(request).Get("X-Ambassador-Client-ID")
	clientSecret := filterutil.GetHeader(request).Get("X-Ambassador-Client-Secret")

	oauthClient, err := rfc6749client.NewClientCredentialsClient(
		discovered.TokenEndpoint,
		rfc6749client.ClientPasswordHeader(clientID, clientSecret),
		httpClient,
	)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			err, nil)
	}
	oauthClient.RegisterProtocolExtensions(rfc6750client.OAuthProtocolExtension)

	sessionData, err := c.loadSession(redisClient, clientID, clientSecret)
	if err != nil {
		logger.Debugln("session status:", errors.Wrap(err, "no session"))
	}
	defer func() {
		if sessionData != nil {
			err := c.saveSession(redisClient, clientID, clientSecret, sessionData)
			if err != nil {
				// TODO(lukeshu): Letting FilterMux recover() this panic() and generate an error message
				// isn't the *worst* way of handling this error.
				panic(err)
			}
		}
	}()

	for {
		if sessionData == nil {
			sessionData, err = oauthClient.AuthorizationRequest(scope)
			if err != nil {
				if _, isAuthErr := err.(rfc6749client.TokenErrorResponse); isAuthErr {
					return middleware.NewErrorResponse(ctx, http.StatusForbidden,
						err, nil)
				}
				return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
					err, nil)
			}
		}

		authorization, err := oauthClient.AuthorizationForResourceRequest(sessionData, func() io.Reader {
			return strings.NewReader(request.GetRequest().GetHttp().GetBody())
		})
		switch err {
		case nil:
			return c.handleAuthenticatedProxyRequest(ctx, logger, httpClient, discovered, request, authorization, sessionData)
		case rfc6749client.ErrNoAccessToken, rfc6749client.ErrExpiredAccessToken:
			sessionData = nil
			continue
		default:
			if _, ok := err.(*rfc6749client.UnsupportedTokenTypeError); ok {
				return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
					err, nil)
			}
			return middleware.NewErrorResponse(ctx, http.StatusInternalServerError,
				errors.Wrap(err, "unknown error"), nil)
		}
	}
}

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

func (c *OAuth2Client) handleAuthenticatedProxyRequest(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, request *filterapi.FilterRequest, authorization http.Header, sessionData *rfc6749client.ClientCredentialsClientSessionData) filterapi.FilterResponse {
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

	resourceResponse := c.ResourceServer.Filter(ctx, logger, httpClient, discovered, request, sessionData.CurrentAccessToken.Scope)
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

func (c *OAuth2Client) loadSession(redisClient *redis.Client, clientID, clientSecret string) (*rfc6749client.ClientCredentialsClientSessionData, error) {
	sessionID := url.QueryEscape(clientID) + ":" + url.QueryEscape(clientSecret)

	sessionDataBytes, err := redisClient.Cmd("GET", "session:"+sessionID).Bytes()
	if err != nil {
		return nil, err
	}
	sessionData := new(rfc6749client.ClientCredentialsClientSessionData)
	if err := json.Unmarshal(sessionDataBytes, sessionData); err != nil {
		return nil, err
	}
	return sessionData, nil
}

func (c *OAuth2Client) saveSession(redisClient *redis.Client, clientID, clientSecret string, sessionData *rfc6749client.ClientCredentialsClientSessionData) error {
	sessionID := url.QueryEscape(clientID) + ":" + url.QueryEscape(clientSecret)
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
