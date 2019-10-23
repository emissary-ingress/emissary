package rfc6749

import (
	"net/url"

	"github.com/pkg/errors"

	rfc6749common "github.com/datawire/liboauth2/common/rfc6749"
)

// validateAuthorizationEndpointURI validates the requirements in §3.1.
func validateAuthorizationEndpointURI(endpoint *url.URL) error {
	if endpoint.Fragment != "" {
		return errors.Errorf("the Authorization Endpoint URI MUST NOT include a fragment component: %v", endpoint)
	}
	return nil
}

// buildAuthorizationRequestURI inserts queryParameters per §3.1.
func buildAuthorizationRequestURI(endpoint *url.URL, queryParameters url.Values) (*url.URL, error) {
	query := endpoint.Query()
	for k, vs := range queryParameters {
		if _, exists := query[k]; exists {
			return nil, errors.Errorf("cannot build Authorization Request URI: cannot insert %q parameter: Authorization Endpoint URI already includes parameter, and request parameters MUST NOT be included more than once", k)
		}
		if len(vs) > 1 {
			return nil, errors.Errorf("cannot build Authorization Request URI: request parameters MUST NOT be included more than once: %q", k)
		}
		query[k] = vs
	}
	ret := *endpoint
	ret.RawQuery = query.Encode()
	return &ret, nil
}

// validateRedirectionEndpointURI validates the requirements in §3.1.2.
func validateRedirectionEndpointURI(endpoint *url.URL) error {
	if !endpoint.IsAbs() {
		return errors.Errorf("the Redirection Endpoint MUST be an absolute URI: %v", endpoint)
	}
	if endpoint.Fragment != "" {
		return errors.Errorf("the Redirection Endpoint URI MUST NOT include a fragment component: %v", endpoint)
	}
	return nil
}

// validateTokenEndpointURI validates the requirements in §3.2.
func validateTokenEndpointURI(endpoint *url.URL) error {
	if endpoint.Fragment != "" {
		return errors.Errorf("the Token Endpoint URI MUST NOT include a fragment component: %v", endpoint)
	}
	return nil
}

// Scope represents an unordered list of scope-values as defined by §3.3.
type Scope = rfc6749common.Scope

// ParseScope de-serializes the set of scope-values from use as a parameter, per §3.3.
func ParseScope(str string) Scope {
	return rfc6749common.ParseScope(str)
}
