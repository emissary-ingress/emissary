package rfc6749

import (
	"net/url"
	"strconv"
	"testing"
)

func TestBuildAuthorizationRequestURI(t *testing.T) {
	testcases := []struct {
		endpoint        string
		queryParameters url.Values
		expected        string
	}{
		{"https://example.com/authz", url.Values{"foo": {"bar"}}, "https://example.com/authz?foo=bar"},
		{"https://example.com/authz?bar=baz", url.Values{"foo": {"FOO"}}, "https://example.com/authz?foo=FOO&bar=baz"},

		// request parameters MUST NOT be included more than once
		{"https://example.com/authz?foo=init", url.Values{"foo": {"FOO"}}, ""},
		{"https://example.com/authz", url.Values{"foo": {"fooa", "foob"}}, ""},

		// we have to retain application/x-www-form-urlencoded queries, but don't have to
		// retain other queries
		{"https://example.com/authz?bad%GGencoding", url.Values{"foo": {"FOO"}}, "https://example.com/authz?foo=FOO"},

		// RFC 6749 references the HTML 4.01 definition of
		// application/x-www-form-urlencoded, but lets go ahead and make sure that it
		// handles HTML-5-incorporated value-less fields.
		{"https://example.com/authz?frob", url.Values{"foo": {"FOO"}}, "https://example.com/authz?frob=&foo=FOO"},
	}
	for i, testcase := range testcases {
		t.Run(strconv.Itoa(i), func(testcase struct {
			endpoint        string
			queryParameters url.Values
			expected        string
		}) func(t *testing.T) {
			return func(t *testing.T) {
				// NB: Normalize the queries to avoid insignificant ordering issues
				endpointURL, err := url.Parse(testcase.endpoint)
				if err != nil {
					t.Errorf("could not parse testcase endpoint: %q: %v", testcase.endpoint, err)
				}
				var expectedURL *url.URL
				var expectedQuery url.Values
				if testcase.expected != "" {
					expectedURL, err = url.Parse(testcase.expected)
					if err != nil {
						t.Errorf("could not parse testcase expected result: %q: %v", testcase.expected, err)
					}
					expectedQuery, err = url.ParseQuery(expectedURL.RawQuery)
					if err != nil {
						t.Errorf("testcase expected result has bad query: %q: %v", expectedURL.RawQuery, err)
					}
					expectedURL.RawQuery = expectedQuery.Encode()
				}

				// Run it
				actualURL, actualErr := buildAuthorizationRequestURI(endpointURL, testcase.queryParameters)
				if actualErr == nil {
					actualQuery, err := url.ParseQuery(actualURL.RawQuery)
					if err != nil {
						t.Errorf("actual result has bad query: %q: %v", actualURL.RawQuery, err)
					}
					actualURL.RawQuery = actualQuery.Encode()
				}

				// Check it
				if expectedURL != nil {
					if actualErr != nil {
						t.Errorf("got unexpected error: %v", actualErr)
					}
					if actualURL.String() != expectedURL.String() {
						t.Errorf("actual URL %q did not match expected URL %q", actualURL.String(), expectedURL.String())
					}
				} else {
					if actualErr == nil {
						t.Errorf("expected an error, but got %v", actualURL)
					}
				}
			}
		}(testcase))
	}
}
