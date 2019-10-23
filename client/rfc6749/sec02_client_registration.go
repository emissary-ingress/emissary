package rfc6749

import (
	"net/http"
	"net/url"
)

// A ClientAuthenticationMethod modifies an the HTTP header and/or
// "application/x-www-form-url-encoded" HTTP body before an HTTP request to the Authorization Server
// is sent, such that it includes appropriate Client Authentication as defined in §2.3.
//
// BUG(lukeshu): It is not possible to have a ClientAuthenticationMethod that requires multiple
// round-trips.  That's a "limitation", but it's probably a good thing.
type ClientAuthenticationMethod func(header http.Header, body url.Values)

// ClientPasswordHeader implements HTTP Basic authentication of the Client, as specified in §2.3.1.
func ClientPasswordHeader(clientID, clientPassword string) ClientAuthenticationMethod {
	return func(header http.Header, _ url.Values) {
		r := &http.Request{Header: header}
		r.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(clientPassword))
	}
}

// ClientPasswordBody implements request-body authentication of the Client, as specified in §2.3.1.
// This NOT RECOMMENDED, you should use `ClientPasswordHeader` instead.
func ClientPasswordBody(clientID, clientPassword string) ClientAuthenticationMethod {
	return func(_ http.Header, body url.Values) {
		body.Set("client_id", clientID)
		body.Set("client_secret", clientPassword)
	}
}
