package oauth2handler

import (
	"context"
	"crypto/rsa"
	"net/http"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/pkg/errors"

	crd "github.com/datawire/apro/apis/getambassador.io/v1beta2"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/httpclient"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/middleware"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/client/authorization_code_client"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/client/client_credentials_client"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/client/header_credentials_client"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/discovery"
	"github.com/datawire/apro/cmd/amb-sidecar/filters/handler/oauth2handler/resourceserver"
	"github.com/datawire/apro/lib/filterapi"
)

// OAuth2Filter implements (part of?) an OAuth 2 Client, and the top
// half of an OAuth 2 Resource Server.
//
// Background:
//
//   An OAuth Client application is an application that receives an
//   Access Token from an Authorization Server, and uses that Access
//   Token to access resources on a Resource Server.
//
//   An OAuth Resource Server is an HTTP service that is in cahoots
//   with an Authorization Server, such that access to the resources
//   that it exposes is protected via an Access Token.
//
//   As an example, a Client might be some application that has
//   sign-in-with-GitHub (like ZenHub), and then can talk to GitHub's
//   Resource Server to access your data and provide functionality on
//   top of that (like a view that aggregates issues from multiple
//   repositories).
//
// The OAuth Client that is the OAuth2Filter is super dumb.  It isn't
// building any features on top of the ResourceServer.  The
// functionality that it has is "proxy requests (near-)verbatim to the
// Resource Server, and proxy responses verbatim back to the
// initiating HTTP client".  It's an OAuth Client in the way that UDP
// is an L4 protocol.
//
// Anyway, the Client-part takes whatever the initiating HTTP client
// (web-browser?)  submitted, and talks with the Authorization Server
// to exchange that for an Access Token.  It injects that Access Token
// in to the request such that the Resource Server can see it.
// Exactly what it's expecting from the initiating HTTP client, and
// how it goes about talking to the Authorization server, depends on
// the configured `grantType`.  The different grantTypes have entirely
// separate Client implementations that live in sub-packages.
//
// The Resource-Server-Part takes that request that now has the Access
// Token injected, and validates the Access Token, ensuring that this
// user has access to (and has granted this Client access to) the
// specific resource (by checking the token's "scope").  If the Access
// Token has insufficient privilege, we return a permission-denied
// response.  If it does have sufficient privilege, then we instruct
// Envoy to pass the (modified) request along to the upstream backend
// service (the "bottom half" of the Resource Server).
//
// Everything described above happens in the `Filter()` method (which
// calls out to a more specific `Filter()` method based on the
// grantType).  Some of the clients for different grantTypes require
// having their own helper HTTP endpoints that we don't proxy to the
// ResourceServer.  Serving those endpoints happens in the
// `ServeHTTP()` method (which calls out to a more specific
// `ServeHTTP()` method based on the grantType).
type OAuth2Filter struct {
	PrivateKey   *rsa.PrivateKey
	PublicKey    *rsa.PublicKey
	RedisPool    *pool.Pool
	QName        string
	Spec         crd.FilterOAuth2
	Arguments    crd.FilterOAuth2Arguments
	RunFilters   func(filters []crd.FilterReference, ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error)
	RunJWTFilter func(filterRef crd.JWTFilterReference, ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error)
}

// OAuth2Client is the common interface implemented by the OAuth
// Clients for different grantTypes.
//
// FIXME(lukeshu): You have my sincerest apologies for the arguments
// lists of these functions.
type OAuth2Client interface {
	Filter(ctx context.Context, logger dlog.Logger, httpClient *http.Client, discovered *discovery.Discovered, redisClient *redis.Client, request *filterapi.FilterRequest) filterapi.FilterResponse
	ServeHTTP(w http.ResponseWriter, r *http.Request, ctx context.Context, discovered *discovery.Discovered, redisClient *redis.Client)
}

func (f *OAuth2Filter) Filter(ctx context.Context, request *filterapi.FilterRequest) (filterapi.FilterResponse, error) {
	logger := dlog.GetLogger(ctx)
	httpClient := httpclient.NewHTTPClient(logger, f.Spec.MaxStale, f.Spec.InsecureTLS, f.Spec.RenegotiateTLS)

	discovered, err := discovery.Discover(httpClient, f.Spec, logger)
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			errors.Wrap(err, "OIDC-discovery"), nil), nil
	}

	redisClient, err := f.RedisPool.Get()
	if err != nil {
		return middleware.NewErrorResponse(ctx, http.StatusBadGateway,
			errors.Wrap(err, "Redis"), nil), nil
	}
	defer f.RedisPool.Put(redisClient)

	var oauth2client OAuth2Client
	switch f.Spec.GrantType {
	case crd.GrantType_AuthorizationCode:
		oauth2client = &authorization_code_client.OAuth2Client{
			QName:     f.QName,
			Spec:      f.Spec,
			Arguments: f.Arguments,

			ResourceServer: &resourceserver.OAuth2ResourceServer{
				QName:        f.QName,
				Spec:         f.Spec,
				Arguments:    f.Arguments,
				RunJWTFilter: f.RunJWTFilter,
			},

			PrivateKey: f.PrivateKey,
			PublicKey:  f.PublicKey,

			RunFilters: f.RunFilters,
		}
	case crd.GrantType_ClientCredentials:
		oauth2client = &client_credentials_client.OAuth2Client{
			QName:     f.QName,
			Spec:      f.Spec,
			Arguments: f.Arguments,

			ResourceServer: &resourceserver.OAuth2ResourceServer{
				QName:        f.QName,
				Spec:         f.Spec,
				Arguments:    f.Arguments,
				RunJWTFilter: f.RunJWTFilter,
			},
		}
	case crd.GrantType_HeaderCredentials:
		oauth2client = &header_credentials_client.OAuth2Client{
			QName:     f.QName,
			Spec:      f.Spec,
			Arguments: f.Arguments,

			ResourceServer: &resourceserver.OAuth2ResourceServer{
				QName:        f.QName,
				Spec:         f.Spec,
				Arguments:    f.Arguments,
				RunJWTFilter: f.RunJWTFilter,
			},
		}
	default:
		panic(errors.Errorf("unrecognized grantType=%#v", f.Spec.GrantType))
	}

	return oauth2client.Filter(ctx, logger, httpClient, discovered, redisClient, request), nil
}

func (f *OAuth2Filter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := dlog.GetLogger(ctx)
	httpClient := httpclient.NewHTTPClient(logger, f.Spec.MaxStale, f.Spec.InsecureTLS, f.Spec.RenegotiateTLS)

	discovered, err := discovery.Discover(httpClient, f.Spec, logger)
	if err != nil {
		middleware.ServeErrorResponse(w, ctx, http.StatusBadGateway,
			errors.Wrap(err, "OIDC-discovery"), nil)
		return
	}

	redisClient, err := f.RedisPool.Get()
	if err != nil {
		middleware.ServeErrorResponse(w, ctx, http.StatusBadGateway,
			errors.Wrap(err, "Redis"), nil)
		return
	}
	defer f.RedisPool.Put(redisClient)

	var oauth2client OAuth2Client
	switch f.Spec.GrantType {
	case crd.GrantType_AuthorizationCode:
		oauth2client = &authorization_code_client.OAuth2Client{
			QName:     f.QName,
			Spec:      f.Spec,
			Arguments: f.Arguments,

			PrivateKey: f.PrivateKey,
			PublicKey:  f.PublicKey,
		}
	case crd.GrantType_ClientCredentials:
		oauth2client = &client_credentials_client.OAuth2Client{
			QName:     f.QName,
			Spec:      f.Spec,
			Arguments: f.Arguments,
		}
	default:
		panic(errors.Errorf("unrecognized grantType=%#v", f.Spec.GrantType))
	}

	oauth2client.ServeHTTP(w, r, ctx, discovered, redisClient)
}
