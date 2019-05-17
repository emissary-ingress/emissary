package rfc6749client

import (
	"net/url"
	"net/http"
)

// A ClientAuthenticationMethod modifies an *http.Request before it is
// sent, such that it includes appropriate Client Authentication as
// defined in §2.3.
//
// BUG(lukeshu): It is not possible to have a
// ClientAuthenticationMethod that requires multiple round-trips.
// That's a "limitation", but it's probably a good thing.
type ClientAuthenticationMethod func(*http.Request)

// ClientPassword implements HTTP Basic authentication as specified in
// §2.3.1.
func ClientPassword(clientID, clientPassword string) ClientAuthenticationMethod {
	return func(r *http.Client) {
		r.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(clientPassword))
	}
}
