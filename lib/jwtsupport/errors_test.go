package jwtsupport_test

import (
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/datawire/apro/lib/jwtsupport"
	"github.com/datawire/apro/lib/testutil"

	jwt "github.com/dgrijalva/jwt-go"
)

func TestSanitizeParse(t *testing.T) {
	assert := &testutil.Assert{T: t}

	// This JWT should be rejected because it is expired
	jwtStr := "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6InU0T2ZORlBId0VCb3NIanRyYXVPYlY4NExuWSIsImtpZCI6InU0T2ZORlBId0VCb3NIanRyYXVPYlY4NExuWSJ9.eyJhdWQiOiI3NzVkYjgwNi0wNDkzLTRhZWQtODRmOS05ZDU5ZjA1NzI3NGQiLCJpc3MiOiJodHRwczovL3N0cy53aW5kb3dzLm5ldC9hODdiNmQzZC1kODVlLTRkOWItODcwNC02YWVkNzZhNDk0NDQvIiwiaWF0IjoxNTY0NDA1NjcwLCJuYmYiOjE1NjQ0MDU2NzAsImV4cCI6MTU2NDQwOTU3MCwiYWNyIjoiMSIsImFpbyI6IkFUUUF5LzhNQUFBQUpaaVVlTm5mSGpqTU92YmFEc2hyTEk2dUZsSHNud1gvSms0YXpGTDlCZUdkZ2wyTmN2S3I3bFFZeDFEb1JhdTAiLCJhbXIiOlsicHdkIl0sImFwcGlkIjoiNDM4MzQ0MGUtZjgwNy00Y2RmLTlhNTAtNmMxOWRhNWY3MzQzIiwiYXBwaWRhY3IiOiIxIiwiZmFtaWx5X25hbWUiOiJ2YW4gZGVyIE1lZXIiLCJnaXZlbl9uYW1lIjoiQWtlIiwiaXBhZGRyIjoiMTQ1LjY0LjEzNC4yNDUiLCJuYW1lIjoidmFuIGRlciBNZWVyIEFrZSIsIm9pZCI6IjI2NTc2OWE2LWM4ZDctNDg3OC04ZDJjLTQyZjhmMDhjMDhkZSIsIm9ucHJlbV9zaWQiOiJTLTEtNS0yMS0xMjI5MjcyODIxLTYwMjE2MjM1OC04Mzk1MjIxMTUtNjIwODMiLCJzY3AiOiJ0ZXN0LXNjb3BlIiwic3ViIjoiSUxGcHZzOXA3QlVxcDFRZ2dmNW1ySThIQlJ0cFRiNXZsX0FvbnZpMmt1YyIsInRpZCI6ImE4N2I2ZDNkLWQ4NWUtNGQ5Yi04NzA0LTZhZWQ3NmE0OTQ0NCIsInVuaXF1ZV9uYW1lIjoiYXZkbWVlci5leHRlcm5hbEBlcG8ub3JnIiwidXBuIjoiYXZkbWVlci5leHRlcm5hbEBlcG8ub3JnIiwidXRpIjoiVGdxT2Z0TjM4MDZaRjI1NF9uVV9BQSIsInZlciI6IjEuMCJ9.D8-t4m35cw59Rv4OLWeQIpPfcceA9k9ddvWArQbHkXhJZ5aTZS7mTtLtHOus5gi1g12pxKcuhJ_7DgCL918sqI6pxp9f8LBYBoIMfThBYo7opg8KXxSRSVrbE4l-XUvXBukMC-cINCYXmItLqePns8sIY9bmWchKqSs083eK2O8JaiNWDRmmcDqRO7N90MqpXIqNKBpAtZWtGjpM4cyRbTniYsYqSuHv6YA8u91UF0NKyEVNuQyBH-ThNTr2aPTKTKUmfnxgE3u44s8aSq8920hyKpAU7u0y-SnPTK3rRAkWmj2QMJtJd8P0JLSYr50VwBNpAelB-oaQ2Flo_OsuAQ"
	jwkEStr := "AQAB"
	jwkNStr := "oRRQG-ib30x09eWtDpL0wWahA-hgjc0lWoQU4lwBFjXV2PfPImiAvwxOxNG34Mgnw3K9huBYLsrvOQAbMdBmE8lwz8DFKMWqHqoH3xSqDGhIYFobQDiVRkkecpberH5hqJauSD7PiwDBSQ_RCDIjb0SOmSTpZR97Ws4k1z9158VRf4BUbGjzVt4tUAz_y2cI5JsXQfcgAPB3voP8eunxGwZ_iM8evw3hUOw7-nuiPyts7HSkvV6GMwrXfOymY_w07mYxw_2LnKInfsWBtcRIDG-Nrsj237LgtBhK7TkzuVrguq__-bkDwwF3qTRXGAX9KrwY4huRxDRslMIg30Hqgw"

	jwkEBytes, err := base64.RawURLEncoding.DecodeString(jwkEStr)
	assert.NotError(err)
	jwkE := big.NewInt(0)
	jwkE.SetBytes(jwkEBytes)

	jwkNBytes, err := base64.RawURLEncoding.DecodeString(jwkNStr)
	assert.NotError(err)
	jwkN := big.NewInt(0)
	jwkN.SetBytes(jwkNBytes)

	jwk := &rsa.PublicKey{
		E: int(jwkE.Int64()),
		N: jwkN,
	}

	t.Run("valid signature", func(t *testing.T) {
		assert := &testutil.Assert{T: t}

		_, err = jwtsupport.SanitizeParse(jwt.Parse(jwtStr, func(_ *jwt.Token) (interface{}, error) {
			return jwk, nil
		}))
		assert.Bool(err != nil)
		assert.StrEQ(`Token validation error: token is invalid: errorFlags=0x00000010=(ValidationErrorExpired) wrappedError=(Token is expired)`, err.Error())
	})

	t.Run("invalid signature", func(t *testing.T) {
		assert := &testutil.Assert{T: t}

		_, err = jwtsupport.SanitizeParse(jwt.Parse(jwtStr+"bogus", func(_ *jwt.Token) (interface{}, error) {
			return jwk, nil
		}))
		assert.Bool(err != nil)
		assert.StrEQ(`Token validation error: token is invalid: errorFlags=0x00000014=(ValidationErrorSignatureInvalid|ValidationErrorExpired) wrappedError=(crypto/rsa: verification error)`, err.Error())
	})
}
