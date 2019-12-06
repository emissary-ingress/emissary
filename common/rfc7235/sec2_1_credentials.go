package rfc7235

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/apro/common/rfc7235/internal/rfc7230"
)

// Credentials is a set of authentication credentials, as defined by §2.1
type Credentials struct {
	AuthScheme string
	Body       CredentialsBody
}

func (c Credentials) String() string {
	return c.AuthScheme + " " + c.Body.String()
}

// ParseCredentials parses a string as a set of credentials as defined by §2.1.
//
// If the entirety of str can't be parsed as a set of credentials, then (Credentials{}, err) is
// returned, where err describes what's wrong with the input sting.
func ParseCredentials(str string) (Credentials, error) {
	ret, rest, err := scanCredentials(str)
	if err != nil {
		return Credentials{}, err
	}
	if rest != "" {
		return Credentials{}, errors.Errorf("invalid credentials: unparsable suffix: %q", rest)
	}
	return ret, nil
}

// scanCredentials scans credentials (as defined by §2.1) from the beginning of a string, and
// returns the structured result, as well as the remainder of the input string.
//
// If the input does not have credentials as a prefix, then (Credentials{}, "", err) is returned,
// where err explains why the prefix is not a set of credentials.
func scanCredentials(str string) (Credentials, string, error) {
	// ABNF:
	//     credentials    = auth-scheme [ 1*SP ( token68 / #auth-param ) ]
	//     auth-scheme    = token
	//     token68        = 1*( ALPHA / DIGIT / "-" / "." / "_" / "~" / "+" / "/" ) *"="
	//     auth-param     = token BWS "=" BWS ( token / quoted-string )
	//     OWS            = *( SP / HTAB )
	//     BWS            = OWS

	sp := strings.IndexByte(str, ' ')
	if sp < 0 {
		return Credentials{}, "", errors.New("invalid credentials: no ' ' (SP) to separate the auth-scheme from the body")
	}
	authScheme := str[:sp]
	rest := strings.TrimLeft(str[sp:], " ")
	if !rfc7230.IsValidToken(authScheme) {
		return Credentials{}, "", errors.Errorf("invalid credentials: invalid auth-scheme: %q", authScheme)
	}

	// try both, and choose the greedy option
	var body CredentialsBody
	legacyStr, legacyRest, legacyErr := scanToken68(rest)
	paramsRaw, paramsRest, paramsErr := rfc7230.ScanList(rest, 0, 0, func(input string) (interface{}, string, error) { return scanAuthParam(input) })
	switch {
	case legacyErr != nil && paramsErr != nil:
		return Credentials{}, "", errors.Errorf("invalid credentials: body does not appear to be a token68 (%v) or an auth-param list (%v)", legacyErr, paramsErr)
	case legacyErr == nil && (paramsErr != nil || len(legacyRest) < len(paramsRest)):
		body = CredentialsLegacy(legacyStr)
		rest = legacyRest
	case paramsErr == nil && (legacyErr != nil || len(paramsRest) < len(legacyRest)):
		_body := make(CredentialsParameters, 0, len(paramsRaw))
		for _, param := range paramsRaw {
			_body = append(_body, param.(AuthParam))
		}
		body = _body
		rest = paramsRest
	default:
		panic("should not happen")
	}
	return Credentials{AuthScheme: authScheme, Body: body}, rest, nil
}

// CredentialsBody is the body of a set of Credentials; either CredentialsParameters or
// CredentialsLegacy, as defined by §2.1.
type CredentialsBody interface {
	fmt.Stringer
	isCredentialsBody()
}

// CredentialsParameters is a list of authentication parameters making up the body of a Credentials,
// as defined by §2.1.
type CredentialsParameters []AuthParam

func (params CredentialsParameters) String() string {
	paramStrs := make([]string, 0, len(params))
	for _, param := range params {
		paramStrs = append(paramStrs, param.String())
	}
	return strings.Join(paramStrs, ", ")
}

func (CredentialsParameters) isCredentialsBody() {}

// CredentialsLegacy is a "token68" string making up the body of a Credentials, as defined by §2.1.
//
// Per §2.1:
//  > The token68 syntax allows the 66 unreserved URI characters
//  > ([RFC3986]), plus a few others, so that it can hold a base64,
//  > base64url (URL and filename safe alphabet), base32, or base16 (hex)
//  > encoding, with or without padding, but excluding whitespace
//  > ([RFC4648]).
//
// Per §5.1.2:
//  > The "token68" notation was introduced for compatibility with
//  > existing authentication schemes and can only be used once per
//  > credentials or credential.  Thus, new schemes ought to use the
//  > auth-param syntax instead, because otherwise future extensions
//  > will be impossible.
type CredentialsLegacy string

func (t68 CredentialsLegacy) String() string {
	return string(t68)
}

func (CredentialsLegacy) isCredentialsBody() {}
