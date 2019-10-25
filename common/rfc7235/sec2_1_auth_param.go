package rfc7235

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/apro/common/rfc7235/internal/rfc7230"
)

// An AuthParam is an authentication parameter, as defined by §2.1.
type AuthParam struct {
	Key   string
	Value string
}

func (p AuthParam) String() string {
	if !rfc7230.IsValidToken(p.Key) {
		// panic; in the same class as asking us to dereference a nil pointer.
		panic(errors.Errorf("invalid auth-param key: %q", p.Key))
	}

	// §2.2 notes that as a special case, for the "realm" authentication parameter, "For
	// historical reasons, a sender MUST only generate the quoted-string syntax."
	if !rfc7230.IsValidToken(p.Value) || p.Key == "realm" {
		return p.Key + "=" + rfc7230.QuoteString(p.Value)
	}
	return p.Key + "=" + p.Value
}

// ScanAuthParam scans a auth-param (as defined by §2.1) from the beginning of a string, and returns
// the structured result, as well as the remainder of the input string.
//
// For example:
//
//     ScanAuthParam(`foo = "bar baz" remainder`) => (AuthParam{Key: `foo`, `Value: `bar baz`}, ` remainder`, nil)
//
// If a syntax error is encountered, (AuthParam{}, "", err) is returned.
func ScanAuthParam(input string) (param AuthParam, rest string, err error) {
	rest = input

	// ABNF:
	//     auth-paramx = token BWS "=" BWS ( token / quoted-string )

	// ABNF: token
	param.Key = rest[:len(rest)-len(strings.TrimLeft(rest, rfc7230.CharsetTChar))]
	if len(param.Key) == 0 {
		switch {
		case len(rest) == 0:
			return AuthParam{}, "", errors.New("invalid auth-param: expected a key token, but got EOF")
		default:
			return AuthParam{}, "", errors.Errorf("invalid auth-param: expected a key token, but got non-token character %#v", rest[0])
		}
	}
	rest = rest[len(param.Key):]

	// ABNF: BWS
	rest = strings.TrimLeft(rest, " \t")

	// ABNF: "="
	switch {
	case len(rest) == 0:
		return AuthParam{}, "", errors.New("invalid auth-param: expected an '=', but got EOF")
	case rest[0] != '=':
		return AuthParam{}, "", errors.Errorf("invalid auth-param: expected an '=', but got %#v", rest[0])
	default:
		rest = rest[1:]
	}

	// ABNF: BWS
	rest = strings.TrimLeft(rest, " \t")

	// ABNF: ( token / quoted-string )
	switch {
	case rest == "":
		return AuthParam{}, "", errors.New("invalid auth-param: expected a value, got EOF")
	case rest[0] == '"':
		param.Value, rest, err = rfc7230.ScanQuotedString(rest)
		if err != nil {
			return AuthParam{}, "", errors.Wrap(err, "invalid auth-param")
		}
	default:
		param.Value = rest[:len(rest)-len(strings.TrimLeft(rest, rfc7230.CharsetTChar))]
		if len(param.Value) == 0 {
			return AuthParam{}, "", errors.Errorf("invalid auth-param: expected a value, but got non-quoted-string, non-token character %#v", rest[0])
		}
		rest = rest[len(param.Value):]
	}

	return param, rest, nil
}
