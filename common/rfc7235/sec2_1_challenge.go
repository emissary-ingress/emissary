package rfc7235

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/apro/common/rfc7235/internal/rfc5234"
	"github.com/datawire/apro/common/rfc7235/internal/rfc7230"
)

// A Challenge is an authentication challenge, as defined by §2.1
type Challenge struct {
	AuthScheme string
	Body       ChallengeBody
}

func (c Challenge) String() string {
	return c.AuthScheme + " " + c.Body.String()
}

// ParseChallenge parses a string containing a Challenge, as defined by §2.1.
//
// If an auth-scheme is parsed, then the returned Challenge will have AuthScheme set, even if there
// is an error parsing the remainder and an error is returned.
func ParseChallenge(str string) (Challenge, error) {
	// ABNF:
	//     challenge      = auth-scheme [ 1*SP ( token68 / [ ( "," / auth-param ) *( OWS "," [ OWS auth-param ] ) ] ) ]
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
		return Challenge{}, errors.Errorf("invalid challenge: invalid auth-scheme: %q", authScheme)
	}

	ret := Challenge{
		AuthScheme: authScheme,
	}

	if strings.Trim(strings.TrimRight(bodyStr, "="), rfc5234.CharsetALPHA+rfc5234.CharsetDIGIT+"-._~+/") == "" {
		// token68
		ret.Body = ChallengeLegacy(bodyStr)
	} else {
		// auth-param list
		//
		// ABNF:
		//     [ ( "," / auth-param ) *( OWS "," [ OWS auth-param ] ) ]
		bodyList := ChallengeParameters{}
		if len(bodyStr) > 0 {
			var param AuthParam
			var err error
			// leading part
			if bodyStr[0] == ',' {
				bodyStr = bodyStr[1:]
			} else {
				param, bodyStr, err = ScanAuthParam(bodyStr)
				if err != nil {
					return ret, errors.Wrap(err, "invalid challenge")
				}
				bodyList = append(bodyList, param)
			}
			// repeating part
			for len(bodyStr) > 0 {
				// ABNF: OWS ","
				bodyStr = strings.TrimLeft(bodyStr, " \t")
				if len(bodyStr) == 0 {
					return ret, errors.New("invalid challenge: expected a ',' bug got EOF")
				} else if bodyStr[0] != ',' {
					return ret, errors.Errorf("invalid challenge: expected a ',' bug got %#v", bodyStr[0])
				}
				bodyStr = bodyStr[1:]
				// ABNF: [ OWS auth-param ]
				if len(strings.TrimLeft(bodyStr, " \t")) > 0 {
					param, bodyStr, err = ScanAuthParam(strings.TrimLeft(bodyStr, " \t"))
					if err != nil {
						return ret, errors.Wrap(err, "invalid challenge")
					}
					bodyList = append(bodyList, param)
				}
			}
		}
		ret.Body = bodyList
	}

	return ret, nil
}

// ChallengeBody is the body of a Challenge; either ChallengeParameters or ChallengeLegacy, as
// defined by §2.1.
type ChallengeBody interface {
	fmt.Stringer
	isChallengeBody()
}

// ChallengeParameters a list of authentication parameters making up the body of a Challenge, as
// defined by §2.1.
type ChallengeParameters []AuthParam

func (params ChallengeParameters) String() string {
	paramStrs := make([]string, 0, len(params))
	for _, param := range params {
		paramStrs = append(paramStrs, param.String())
	}
	return strings.Join(paramStrs, ", ")
}

func (ChallengeParameters) isChallengeBody() {}

// ChallengeLegacy is a "token68" string making up the body of a Challenge, as defined by §2.1.
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
//  > challenge or credential.  Thus, new schemes ought to use the
//  > auth-param syntax instead, because otherwise future extensions
//  > will be impossible.
type ChallengeLegacy string

func (t68 ChallengeLegacy) String() string {
	return string(t68)
}

func (ChallengeLegacy) isChallengeBody() {}
