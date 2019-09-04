package rfc6749

import (
	"io"
	"net/http"
)

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
	return authorizationForResourceRequest(&client.extensionRegistry, &client.explicitClient, session, getBody)
}
