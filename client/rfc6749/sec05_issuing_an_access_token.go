package rfc6749

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

// parseTokenResponse parses a response from a Token Endpoint, per §5.
//
// This will NOT close the response Body for you.
//
// If the Authorization Server sent a semantically valid error response, the returned error is of
// type TokenErrorResponse.  On protocol errors, a different error type is returned.
func parseTokenResponse(res *http.Response) (TokenResponse, error) {
	switch {
	case res.StatusCode == http.StatusOK:
		mediatype, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
		if err != nil {
			return TokenResponse{}, err
		}
		if mediatype != "application/json" {
			return TokenResponse{}, errors.Errorf("expected \"application/json\" media type, got %q", mediatype)
		}
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return TokenResponse{}, err
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
			return TokenResponse{}, err
		}
		if rawResponse.AccessToken == nil {
			return TokenResponse{}, errors.New("parameter \"access_token\" is missing")
		}
		if rawResponse.TokenType == nil {
			return TokenResponse{}, errors.New("parameter \"token_type\" is missing")
		}
		ret := TokenResponse{
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
			ret.Scope = ParseScope(*rawResponse.Scope)
		}
		return ret, nil
	case res.StatusCode/100 == 4:
		// The spec says "400, unless otherwise specified".  This package doesn't (yet?)
		// keep track of HTTP statuses associated with different error codes.  Even if it
		// did, Auth0 returns 403 for error=invalid_grant, when the spec is clear that it
		// should be using 400 for that.  Assuming that anything in the 4XX range suggests
		// an Error Response seams reasonable.
		mediatype, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
		if err != nil {
			return TokenResponse{}, err
		}
		if mediatype != "application/json" {
			return TokenResponse{}, errors.Errorf("expected \"application/json\" media type, got %q", mediatype)
		}
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return TokenResponse{}, err
		}
		var ret TokenErrorResponse
		err = json.Unmarshal(bodyBytes, &ret)
		if err != nil {
			return TokenResponse{}, err
		}
		return TokenResponse{}, ret
	default:
		return TokenResponse{}, errors.Errorf("unexpected response code: %v", res.Status)
	}
}

// TokenResponse stores a successful response containing a token, as
// specified in §5.1.
type TokenResponse struct {
	AccessToken  string    // REQUIRED.
	TokenType    string    // REQUIRED.
	ExpiresAt    time.Time // RECOMMENDED.
	RefreshToken *string   // OPTIONAL.
	Scope        Scope     // OPTIONAL if identical to scope requested by the client; otherwise REQUIRED.
}

// TokenErrorResponse stores an error response, as specified in §5.2.
type TokenErrorResponse struct {
	ErrorCode        string
	ErrorDescription string
	ErrorURI         *url.URL
}

type rawTokenErrorResponse struct {
	ErrorCode        *string `json:"error"`
	ErrorDescription *string `json:"error_description,omitempty"`
	ErrorURI         *string `json:"error_uri,omitempty"`
}

func (r TokenErrorResponse) Error() string {
	ret := fmt.Sprintf("token error response: error=%q", r.ErrorCode)
	if r.ErrorDescription != "" {
		ret = fmt.Sprintf("%s error_description=%q", ret, r.ErrorDescription)
	}
	if r.ErrorURI != nil {
		ret = fmt.Sprintf("%s error_uri=%q", ret, r.ErrorURI.String())
	}
	return ret
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *TokenErrorResponse) UnmarshalJSON(bodyBytes []byte) error {
	var rawResponse rawTokenErrorResponse
	err := json.Unmarshal(bodyBytes, &rawResponse)
	if err != nil {
		return err
	}
	if rawResponse.ErrorCode == nil {
		return errors.New("parameter \"error\" is missing")
	}
	r.ErrorCode = *rawResponse.ErrorCode
	if rawResponse.ErrorDescription != nil {
		r.ErrorDescription = *rawResponse.ErrorDescription
	}
	if rawResponse.ErrorURI != nil {
		r.ErrorURI, err = url.Parse(*rawResponse.ErrorURI)
		if err != nil {
			return err
		}
	}
	return nil
}

// MarshalJSON implements json.Marshaler.
func (r TokenErrorResponse) MarshalJSON() ([]byte, error) {
	var rawResponse rawTokenErrorResponse
	rawResponse.ErrorCode = &r.ErrorCode
	if r.ErrorDescription != "" {
		rawResponse.ErrorDescription = &r.ErrorDescription
	}
	if r.ErrorURI != nil {
		str := r.ErrorURI.String()
		rawResponse.ErrorURI = &str
	}
	return json.Marshal(rawResponse)
}

// These are the error codes that may be present in a TokenErrorResponse, as enumerated in §5.2.
// This set may be extended by the extensions error registry.
func newBuiltInTokenErrors() map[string]ExtensionError {
	ret := make(map[string]ExtensionError)
	add := func(name, meaning string) {
		if _, set := ret[name]; set {
			panic(errors.Errorf("token error=%q already registered", name))
		}
		ret[name] = ExtensionError{
			Name:                   name,
			UsageLocations:         []ErrorUsageLocation{LocationTokenErrorResponse},
			RelatedExtension:       "(built-in)",
			ChangeController:       "IETF",
			SpecificationDocuments: []string{"RFC 6749"},

			Meaning: meaning,
		}
	}

	add("invalid_request", ""+
		"The request is missing a required parameter, includes an "+
		"unsupported parameter value (other than grant type), "+
		"repeats a parameter, includes multiple credentials, "+
		"utilizes more than one mechanism for authenticating the "+
		"client, or is otherwise malformed.")

	add("invalid_client", ""+
		"Client authentication failed (e.g., unknown client, no "+
		"client authentication included, or unsupported "+
		"authentication method).  The authorization server MAY "+
		"return an HTTP 401 (Unauthorized) status code to indicate "+
		"which HTTP authentication schemes are supported.  If the "+
		"client attempted to authenticate via the \"Authorization\" "+
		"request header field, the authorization server MUST "+
		"respond with an HTTP 401 (Unauthorized) status code and "+
		"include the \"WWW-Authenticate\" response header field "+
		"matching the authentication scheme used by the client. ")

	add("invalid_grant", ""+
		"The provided authorization grant (e.g., authorization "+
		"code, resource owner credentials) or refresh token is "+
		"invalid, expired, revoked, does not match the redirection "+
		"URI used in the authorization request, or was issued to "+
		"another client.")

	add("unauthorized_client", ""+
		"The authenticated client is not authorized to use this "+
		"authorization grant type.")

	add("unsupported_grant_type", ""+
		"The authorization grant type is not supported by the "+
		"authorization server.")

	add("invalid_scope", ""+
		"The requested scope is invalid, unknown, malformed, or "+
		"exceeds the scope granted by the resource owner.")

	return ret
}
