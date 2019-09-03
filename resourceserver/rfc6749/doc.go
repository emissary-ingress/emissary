// Package rfc6749 implements the "Resource Server" role of the OAuth 2.0 Framework.
//
// A Resource Server application (that is, an application that services requests from a Client,
// subject to validating the Access Token-based authorization submitted with the request) will make
// use of this package by... TODO.
package rfc6749

// A ResourceServer implements the Resource Server role in the OAuth 2.0 framework.
type ResourceServer struct {
	extensionRegistry // See sec11_0_iana_considerations.go
}
