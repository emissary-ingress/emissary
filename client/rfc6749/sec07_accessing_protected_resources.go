package rfc6749

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
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
// The per-client wrappers live in `sec07_accessing_protected_resources@${TYPE}.go`.
func authorizationForResourceRequest(
	registry *extensionRegistry,
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
				return nil, ErrExpiredAccessToken
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
	if typeDriver.AuthorizationNeedsBody {
		body = getBody()
	}
	return typeDriver.AuthorizationForResourceRequest(session.currentAccessToken().AccessToken, body)
}

// errorFromResourceResponse inspects a Resource Access Response from a Resource Server, and checks
// for a Token-Type-specific error response format, per §7.2.
//
// The authorization flow must have been completed in order to know what Token Type to look for; if
// the authorization flow has not been completed, then ErrNoAccessToken is returned.  If the Access
// Token Type is unsupported (i.e. it has not been registered with the Client through
// .RegisterProtocolExtensions()), then an error of type *UnsupportedTokenTypeError is returned.
// Other error indicate that there was an error inspecting the response.
func errorFromResourceResponse(
	registry *extensionRegistry,
	session clientSessionData,
	response *http.Response,
) (*ReifiedResourceAccessErrorResponse, error) {
	if session.currentAccessToken() == nil {
		return nil, ErrNoAccessToken
	}
	typeDriver, typeDriverOK := registry.getAccessTokenType(session.currentAccessToken().TokenType)
	if !typeDriverOK {
		return nil, &UnsupportedTokenTypeError{session.currentAccessToken().TokenType}
	}
	resp, err := typeDriver.ErrorFromResourceResponse(response)
	var reifiedResp *ReifiedResourceAccessErrorResponse
	if resp != nil {
		reifiedResp = &ReifiedResourceAccessErrorResponse{
			registry:                    registry,
			ResourceAccessErrorResponse: resp,
		}
	}
	return reifiedResp, err
}

// ResourceAccessErrorResponse is an interface that is implemented by Token-Type-specific error
// responses; per §7.2.
type ResourceAccessErrorResponse interface {
	error
	ErrorCode() string        // SHOULD
	ErrorDescription() string // MAY
	ErrorURI() *url.URL       // MAY
}

// ReifiedResourceAccessErrorResponse wraps a §7.2 ResourceAccessErrorResponse, to provide metadata
// around the ErrorCode, and a useful JSON serialization.
type ReifiedResourceAccessErrorResponse struct {
	registry *extensionRegistry
	ResourceAccessErrorResponse
}

// ErrorCode wraps ResourceAccessErrorResponse.ErrorCode(), returning not just the error code name,
// but also the metadata in the §11 registry for that error code.
func (e *ReifiedResourceAccessErrorResponse) ErrorCode() ExtensionError {
	name := e.ResourceAccessErrorResponse.ErrorCode()

	e.registry.ensureInitialized()
	if ee, eeOK := e.registry.resourceAccessErrors[name]; eeOK {
		return ee
	}

	return ExtensionError{Name: name}
}

// MarshalJSON implements encoding/json.Marshaler.
func (e *ReifiedResourceAccessErrorResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"message":           e.Error(),
		"error":             e.ErrorCode(),
		"error_description": e.ErrorDescription(),
		"error_uri":         e.ErrorURI(),
	})
}
