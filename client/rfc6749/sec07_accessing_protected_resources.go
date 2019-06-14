package rfc6749

import (
	"io"
	"net/http"
	"time"
)

// authorizationForResourceRequest returns a set of HTTP header fields to inject in to HTTP requests
// to the Resource Server, in order to authorize the requests, per §7.1.
//
// If the Access Token is known to be expired and a Refresh Token was granted (and explicitClient is
// non-nil), then AuthorizationForResourceRequest will attempt to refresh it.
//
// This should be called separately for each outgoing request.
//
// ErrNoAccessToken is returned if the authorization flow has not yet been completed.
// ErrExpiredAccessToken is returned if the the Access Token is expired, and could not be refreshed.
// If the Access Token Type is unsupported (i.e. it has not been registered with the Client through
// .RegisterProtocolExtensions()), then an error of type *UnsupportedTokenTypeError is returned.
// Other errors indicate an Token-Type-specific error condition.
//
// This method is unexported, and accepts an interface, so that the implementation can be shared.
// An exported wrapper around it for each client type takes a concrete type instead of an interface.
func authorizationForResourceRequest(
	registry extensionRegistry,
	explicitClient *explicitClient, // optional
	session clientSessionData,
	getBody func() io.Reader,
) (http.Header, error) {
	if session.currentAccessToken() == nil {
		return nil, ErrNoAccessToken
	}
	if !session.currentAccessToken().ExpiresAt.IsZero() && session.currentAccessToken().ExpiresAt.Before(time.Now()) {
		if session.currentAccessToken().RefreshToken != nil && explicitClient != nil {
			if err := explicitClient.refresh(session, nil); err != nil {
				return nil, err
			}
		} else {
			return nil, ErrExpiredAccessToken
		}
	}
	typeDriver, typeDriverOK := registry.getAccessTokenType(session.currentAccessToken().TokenType)
	if !typeDriverOK {
		return nil, &UnsupportedTokenTypeError{session.currentAccessToken().TokenType}
	}
	var body io.Reader
	if typeDriver.NeedsBody {
		body = getBody()
	}
	return typeDriver.AuthorizationForResourceRequest(session.currentAccessToken().AccessToken, body)
}

// Yes, the following is gross.  It's the cost of having clear godocs with concrete types in the
// signatures.

// AuthorizationForResourceRequest returns a set of HTTP header fields to inject in to HTTP requests
// to the Resource Server, in order to authorize the requests, per §7.1.
//
// If the Access Token is known to be expired and a Refresh Token was granted, then
// AuthorizationForResourceRequest will attempt to refresh it.
//
// ErrNoAccessToken is returned if the authorization flow has not yet been completed.
// ErrExpiredAccessToken is returned if the the Access Token is expired, and could not be refreshed.
// If the Access Token Type is unsupported (i.e. it has not been registered with the Client through
// .RegisterProtocolExtensions()), then an error of type *UnsupportedTokenTypeError is returned.
// Other errors indicate an Token-Type-specific error condition.
//
// This should be called separately for each outgoing request.
func (client *AuthorizationCodeClient) AuthorizationForResourceRequest(
	session *AuthorizationCodeClientSessionData,
	getBody func() io.Reader,
) (http.Header, error) {
	return authorizationForResourceRequest(client.extensionRegistry, &client.explicitClient, session, getBody)
}

// AuthorizationForResourceRequest returns a set of HTTP header fields to inject in to HTTP requests
// to the Resource Server, in order to authorize the requests, per §7.1.
//
// ErrNoAccessToken is returned if the authorization flow has not yet been completed.
// ErrExpiredAccessToken is returned if the the Access Token is expired.  If the Access Token Type
// is unsupported (i.e. it has not been registered with the Client through
// .RegisterProtocolExtensions()), then an error of type *UnsupportedTokenTypeError is returned.
// Other errors indicate an Token-Type-specific error condition.
//
// This should be called separately for each outgoing request.
func (client *ImplicitClient) AuthorizationForResourceRequest(
	session *ImplicitClientSessionData,
	getBody func() io.Reader,
) (http.Header, error) {
	return authorizationForResourceRequest(client.extensionRegistry, nil, session, getBody)
}

// AuthorizationForResourceRequest returns a set of HTTP header fields to inject in to HTTP requests
// to the Resource Server, in order to authorize the requests, per §7.1.
//
// If the Access Token is known to be expired and a Refresh Token was granted, then
// AuthorizationForResourceRequest will attempt to refresh it.
//
// ErrNoAccessToken is returned if the authorization flow has not yet been completed.
// ErrExpiredAccessToken is returned if the the Access Token is expired, and could not be refreshed.
// If the Access Token Type is unsupported (i.e. it has not been registered with the Client through
// .RegisterProtocolExtensions()), then an error of type *UnsupportedTokenTypeError is returned.
// Other errors indicate an Token-Type-specific error condition.
//
// This should be called separately for each outgoing request.
func (client *ResourceOwnerPasswordCredentialsClient) AuthorizationForResourceRequest(
	session *ResourceOwnerPasswordCredentialsClientSessionData,
	getBody func() io.Reader,
) (http.Header, error) {
	return authorizationForResourceRequest(client.extensionRegistry, &client.explicitClient, session, getBody)
}

// AuthorizationForResourceRequest returns a set of HTTP header fields to inject in to HTTP requests
// to the Resource Server, in order to authorize the requests, per §7.1.
//
// If the Access Token is known to be expired and a Refresh Token was granted, then
// AuthorizationForResourceRequest will attempt to refresh it.
//
// ErrNoAccessToken is returned if the authorization flow has not yet been completed.
// ErrExpiredAccessToken is returned if the the Access Token is expired, and could not be refreshed.
// If the Access Token Type is unsupported (i.e. it has not been registered with the Client through
// .RegisterProtocolExtensions()), then an error of type *UnsupportedTokenTypeError is returned.
// Other errors indicate an Token-Type-specific error condition.
//
// This should be called separately for each outgoing request.
func (client *ClientCredentialsClient) AuthorizationForResourceRequest(
	session *ClientCredentialsClientSessionData,
	getBody func() io.Reader,
) (http.Header, error) {
	return authorizationForResourceRequest(client.extensionRegistry, &client.explicitClient, session, getBody)
}
