package rfc7235

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/apro/common/rfc7235/internal/rfc5234"
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

// ParseCredentials parses a string containing Credentials, as defined by §2.1.
//
// If an auth-scheme is parsed, then the returned Credentials will have AuthScheme set, even if there
// is an error parsing the remainder and an error is returned.
func ParseCredentials(str string) (Credentials, error) {
	// ABNF:
	//     credentials    = auth-scheme [ 1*SP ( token68 / [ ( "," / auth-param ) *( OWS "," [ OWS auth-param ] ) ] ) ]
	//     token68        = 1*( ALPHA / DIGIT / "-" / "." / "_" / "~" / "+" / "/" ) *"="
	//     auth-param     = token BWS "=" BWS ( token / quoted-string )
	//     OWS            = *( SP / HTAB )
	//     BWS            = OWS
	sp := strings.IndexByte(str, ' ')
	if sp < 0 {
		sp = len(str)
	}

	authScheme := str[:sp]
	bodyStr := strings.TrimLeft(str[sp:], " ")

	if !rfc7230.IsValidToken(authScheme) {
		return Credentials{}, errors.Errorf("invalid credentials: invalid auth-scheme: %q", authScheme)
	}

	ret := Credentials{
		AuthScheme: authScheme,
	}

	if strings.Trim(strings.TrimRight(bodyStr, "="), rfc5234.CharsetALPHA+rfc5234.CharsetDIGIT+"-._~+/") == "" {
		// token68
		ret.Body = CredentialsLegacy(bodyStr)
	} else {
		// auth-param list
		//
		// ABNF:
		//     [ ( "," / auth-param ) *( OWS "," [ OWS auth-param ] ) ]
		bodyList := CredentialsParameters{}
		if len(bodyStr) > 0 {
			var param AuthParam
			var err error
			// leading part
			if bodyStr[0] == ',' {
				bodyStr = bodyStr[1:]
			} else {
				param, bodyStr, err = scanAuthParam(bodyStr)
				if err != nil {
					return ret, errors.Wrap(err, "invalid credentials")
				}
				bodyList = append(bodyList, param)
			}
			// repeating part
			for len(bodyStr) > 0 {
				// ABNF: OWS ","
				bodyStr = strings.TrimLeft(bodyStr, " \t")
				if len(bodyStr) == 0 {
					return ret, errors.New("invalid credentials: expected a ',' bug got EOF")
				} else if bodyStr[0] != ',' {
					return ret, errors.Errorf("invalid credentials: expected a ',' bug got %#v", bodyStr[0])
				}
				bodyStr = bodyStr[1:]
				// ABNF: [ OWS auth-param ]
				if len(strings.TrimLeft(bodyStr, " \t")) > 0 {
					param, bodyStr, err = scanAuthParam(strings.TrimLeft(bodyStr, " \t"))
					if err != nil {
						return ret, errors.Wrap(err, "invalid credentials")
					}
					bodyList = append(bodyList, param)
				}
			}
		}
		ret.Body = bodyList
	}

	return ret, nil
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
