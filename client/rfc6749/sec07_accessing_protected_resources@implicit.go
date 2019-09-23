package rfc6749

import (
	"io"
	"net/http"
)

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
	return authorizationForResourceRequest(&client.extensionRegistry, nil, session, getBody)
}

// ErrorFromResourceResponse inspects a Resource Access Response from a Resource Server, and checks
// for a Token-Type-specific error response format, per §7.2.
//
// The authorization flow must have been completed in order to know what Token Type to look for; if
// the authorization flow has not been completed, then ErrNoAccessToken is returned.  If the Access
// Token Type is unsupported (i.e. it has not been registered with the Client through
// .RegisterProtocolExtensions()), then an error of type *UnsupportedTokenTypeError is returned.
// Other error indicate that there was an error inspecting the response.
func (client *ImplicitClient) ErrorFromResourceResponse(
	session *ImplicitClientSessionData,
	response *http.Response,
) (*ReifiedResourceAccessErrorResponse, error) {
	return errorFromResourceResponse(&client.extensionRegistry, session, response)
}
