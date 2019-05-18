package rfc6749client

import (
	"encoding/json"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

// parseTokenResponse parses a response from a Token Endpoint, per §5.
//
// The returned response is either a TokenSuccessResponse or a
// TokenErrorResponse.
//
// This will NOT close the response Body for you.
func parseTokenResponse(res *http.Response) (TokenResponse, error) {
	switch res.StatusCode {
	case http.StatusOK:
		mediatype, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		if mediatype != "application/json" {
			return nil, errors.Errorf("expected \"application/json\" media type, got %q", mediatype)
		}
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		var rawResponse struct {
			AccessToken  *string  `json:"access_token"`
			TokenType    *string  `json:"token_type"`
			ExpiresIn    *float64 `json:"expires_in"`
			RefreshToken *string  `json:"refresh_token"`
			Scope        *string  `json:"scope"`
		}
		err = json.Unmarshal(bodyBytes, &rawResponse)
		if err != nil {
			return nil, err
		}
		if rawResponse.AccessToken == nil {
			return nil, errors.New("parameter \"access_token\" is missing")
		}
		if rawResponse.TokenType == nil {
			return nil, errors.New("parameter \"token_type\" is missing")
		}
		ret := TokenSuccessResponse{
			AccessToken: *rawResponse.AccessToken,
			TokenType:   *rawResponse.TokenType,
		}
		if rawResponse.ExpiresIn != nil {
			ret.ExpiresAt = time.Now().Add(time.Duration(*rawResponse.ExpiresIn * float64(time.Second)))
		}
		if rawResponse.RefreshToken != nil {
			ret.RefreshToken = rawResponse.RefreshToken
		}
		if rawResponse.Scope != nil {
			ret.Scope = ParseScopes(*rawResponse.Scope)
		}
		return ret, nil
	case http.StatusBadRequest, http.StatusUnauthorized:
		mediatype, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
		if err != nil {
			return nil, err
		}
		if mediatype != "application/json" {
			return nil, errors.Errorf("expected \"application/json\" media type, got %q", mediatype)
		}
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		var rawResponse struct {
			Error            *string `json:"error"`
			ErrorDescription *string `json:"error_description"`
			ErrorURI         *string `json:"error_uri"`
		}
		err = json.Unmarshal(bodyBytes, &rawResponse)
		if err != nil {
			return nil, err
		}
		if rawResponse.Error == nil {
			return nil, errors.New("parameter \"error\" is missing")
		}
		ecodeData, ok := tokenErrorCodeData[*rawResponse.Error]
		if !ok {
			return nil, errors.Errorf("invalid error code: %q", *rawResponse.Error)
		}
		ret := TokenErrorResponse{
			Error: ecodeData.Self,
		}
		if rawResponse.ErrorDescription != nil {
			ret.ErrorDescription = *rawResponse.ErrorDescription
		}
		if rawResponse.ErrorURI != nil {
			ret.ErrorURI, err = url.Parse(*rawResponse.ErrorURI)
			if err != nil {
				return nil, err
			}
		}
		return ret, nil
	default:
		return nil, errors.Errorf("unexpected response code: %v", res.Status)
	}
}

type TokenResponse interface {
	isTokenResponse()
}

// TokenSuccessResponse stores a successful response containing a
// token, as specified in §5.1.
type TokenSuccessResponse struct {
	AccessToken  string    // REQUIRED
	TokenType    string    // REQUIRED
	ExpiresAt    time.Time // RECOMMENDED
	RefreshToken *string   // OPTIONAL
	Scope        Scopes    // OPTIONAL if identical to scope requiested by the client; otherwise REQUIRED.
}

func (r TokenSuccessResponse) isTokenResponse() {}

// TokenSuccessResponse stores an error response, as specified in
// §5.1.
type TokenErrorResponse struct {
	Error            TokenErrorCode
	ErrorDescription string
	ErrorURI         *url.URL
}

func (r TokenErrorResponse) isTokenResponse() {}

type TokenErrorCode interface {
	isTokenErrorCode()
	String() string
	Description() string
}

type tokenErrorCode string

func (ecode tokenErrorCode) isTokenErrorCode()   {}
func (ecode tokenErrorCode) String() string      { return string(ecode) }
func (ecode tokenErrorCode) Description() string { return tokenErrorCodeData[string(ecode)].Description }

var (
	TokenErrorCodeInvalidRequest       TokenErrorCode = tokenErrorCode("invalid_request")
	TokenErrorCodeInvalidClient        TokenErrorCode = tokenErrorCode("invalid_client")
	TokenErrorCodeInvalidGrant         TokenErrorCode = tokenErrorCode("invalid_grant")
	TokenErrorCodeUnauthorizedClient   TokenErrorCode = tokenErrorCode("unauthorized_client")
	TokenErrorCodeUnsupportedGrantType TokenErrorCode = tokenErrorCode("unsupported_grant_type")
	TokenErrorCodeInvalidScope         TokenErrorCode = tokenErrorCode("invalid_scope")
)

var tokenErrorCodeData = map[string]struct {
	Self        TokenErrorCode
	Description string
}{
	"invalid_request": {
		Self: TokenErrorCodeInvalidRequest,
		Description: "The request is missing a required parameter, includes an " +
			"unsupported parameter value (other than grant type), " +
			"repeats a parameter, includes multiple credentials, " +
			"utilizes more than one mechanism for authenticating the " +
			"client, or is otherwise malformed."},

	"invalid_client": {
		Self: TokenErrorCodeInvalidClient,
		Description: "Client authentication failed (e.g., unknown client, no " +
			"client authentication included, or unsupported " +
			"authentication method).  The authorization server MAY " +
			"return an HTTP 401 (Unauthorized) status code to indicate " +
			"which HTTP authentication schemes are supported.  If the " +
			"client attempted to authenticate via the \"Authorization\" " +
			"request header field, the authorization server MUST " +
			"respond with an HTTP 401 (Unauthorized) status code and " +
			"include the \"WWW-Authenticate\" response header field " +
			"matching the authentication scheme used by the client. "},

	"invalid_grant": {
		Self: TokenErrorCodeInvalidGrant,
		Description: "The provided authorization grant (e.g., authorization " +
			"code, resource owner credentials) or refresh token is " +
			"invalid, expired, revoked, does not match the redirection " +
			"URI used in the authorization request, or was issued to " +
			"another client."},

	"unauthorized_client": {
		Self: TokenErrorCodeUnauthorizedClient,
		Description: "The authenticated client is not authorized to use this " +
			"authorization grant type."},

	"unsupported_grant_type": {
		Self: TokenErrorCodeUnsupportedGrantType,
		Description: "The authorization grant type is not supported by the " +
			"authorization server."},

	"invalid_scope": {
		Self: TokenErrorCodeInvalidScope,
		Description: "The requested scope is invalid, unknown, malformed, or " +
			"exceeds the scope granted by the resource owner.",
	},
}
