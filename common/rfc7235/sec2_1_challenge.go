package rfc7235

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"

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

// ParseChallenges returns a list of parsable challenges (challenge being defined in §2.1), as would
// be used in WWW-Authenticate (§4.1) or Proxy-Authenticate (§4.3) from an HTTP header field.
//
// 'field' should probably be "WWW-Authenticate" or "Proxy-Authenticate" (capitalization is not
// significant).
//
// Returns all parsable challenges, and any errors encountered parsing challenges.  An error later
// in the input does not inhibit successfully parse challenges earlier in the input from being
// returned; it is possible for both a non-empty list of challenges and a non-empty list of errors
// to be returned.
func ParseChallenges(field string, header http.Header) ([]Challenge, []error) {
	field = http.CanonicalHeaderKey(field)
	var retvals []Challenge
	var reterrs []error
	for hkey, hvals := range header {
		if http.CanonicalHeaderKey(hkey) != field {
			continue
		}
		for _, hval := range hvals {
			_ret, _rest, _err := scanChallenges(hval)
			if _err != nil {
				reterrs = append(reterrs, _err)
			} else if _rest != "" {
				reterrs = append(reterrs, errors.Errorf("invalid challenge: unparsable suffix: %q", _rest))
			}
			retvals = append(retvals, _ret...)
		}
	}
	return retvals, reterrs
}

// scanChallenges scans a list of challenges (challenge being defined in §2.1), as would be used in
// WWW-Authenticate (§4.1) or Proxy-Authenticate (§4.3) from the beginning of a string, and returns
// the structured result, as well as the remainder of the input string.
//
// If the input does not have a non-empty list of challenges as a prefix, then (nil, "", err) is
// returned, where err explains why the prefix is not a list of challenges.
func scanChallenges(str string) ([]Challenge, string, error) {
	// ABNF:
	//     WWW-Authenticate    = 1#challenge
	//     Proxy-Authenticate  = 1#challenge
	untypedRet, rest, err := rfc7230.ScanList(str, 1, 0, func(input string) (interface{}, string, error) {
		_el, _rest, _err := scanChallenge(input)
		// Because
		//  1. This is a comma-separated-list inside of a comma-separated list,
		//  2. There can be empty elements in either list
		//  3. The parser is greedy
		// the above scanChallenge() can think that it has a trailing empty element, and
		// steal the comma that separates it from the next challenge.  So we check if that
		// happened, and steal it back.
		if _rest != input && _rest != "" && !strings.HasPrefix(strings.TrimLeft(_rest, " \t"), ",") {
			prefix := strings.TrimSuffix(input, _rest)
			if strings.HasSuffix(prefix, ",") {
				_rest = strings.TrimPrefix(input, strings.TrimSuffix(prefix, ","))
			}
		}
		return _el, _rest, _err
	})
	if err != nil {
		return nil, "", err
	}
	ret := make([]Challenge, 0, len(untypedRet))
	for _, el := range untypedRet {
		ret = append(ret, el.(Challenge))
	}
	return ret, rest, nil
}

// scanChallenge scans a challenge (as defined by §2.1) from the beginning of a string, and returns
// the structured result, as well as the remainder of the input string.
//
// If the input does not have a challenge as a prefix, then (Challenge{}, "", err) is returned,
// where err explains why the prefix is not a challenge.
func scanChallenge(str string) (Challenge, string, error) {
	// ABNF:
	//     challenge      = auth-scheme [ 1*SP ( token68 / #auth-param ) ]
	//     auth-scheme    = token
	//     token68        = 1*( ALPHA / DIGIT / "-" / "." / "_" / "~" / "+" / "/" ) *"="
	//     auth-param     = token BWS "=" BWS ( token / quoted-string )
	//     OWS            = *( SP / HTAB )
	//     BWS            = OWS

	sp := strings.IndexByte(str, ' ')
	if sp < 0 {
		return Challenge{}, "", errors.New("invalid challenge: no ' ' (SP) to separate the auth-scheme from the body")
	}
	authScheme := str[:sp]
	rest := strings.TrimLeft(str[sp:], " ")
	if !rfc7230.IsValidToken(authScheme) {
		return Challenge{}, "", errors.Errorf("invalid challenge: invalid auth-scheme: %q", authScheme)
	}

	// try both, and choose the greedy option
	var body ChallengeBody
	legacyStr, legacyRest, legacyErr := scanToken68(rest)
	paramsRaw, paramsRest, paramsErr := rfc7230.ScanList(rest, 0, 0, func(input string) (interface{}, string, error) { return scanAuthParam(input) })
	switch {
	case legacyErr != nil && paramsErr != nil:
		return Challenge{}, "", errors.Errorf("invalid challenge: body does not appear to be a token68 (%v) or an auth-param list (%v)", legacyErr, paramsErr)
	case legacyErr == nil && (paramsErr != nil || len(legacyRest) < len(paramsRest)):
		body = ChallengeLegacy(legacyStr)
		rest = legacyRest
	case paramsErr == nil && (legacyErr != nil || len(paramsRest) < len(legacyRest)):
		_body := make(ChallengeParameters, 0, len(paramsRaw))
		for _, param := range paramsRaw {
			_body = append(_body, param.(AuthParam))
		}
		body = _body
		rest = paramsRest
	default:
		panic("should not happen")
	}
	return Challenge{AuthScheme: authScheme, Body: body}, rest, nil
}

// ChallengeBody is the body of a Challenge; either ChallengeParameters or ChallengeLegacy, as
// defined by §2.1.
type ChallengeBody interface {
	fmt.Stringer
	isChallengeBody()
}

// ChallengeParameters is a list of authentication parameters making up the body of a Challenge, as
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
