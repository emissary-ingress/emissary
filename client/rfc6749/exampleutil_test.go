package rfc6749_test

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
)

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func randomToken() string {
	d := make([]byte, 128)
	if _, err := rand.Read(d); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(d)
}
