// Package rfc7230 provides utilities for dealing with bits of HTTP structured header field syntax.
package rfc7230

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/liboauth2/syntax/rfc5234"
)

const (
	// CharsetTChar is the list of characters that may appear in a valid "token", as defined by
	// §3.2.6.
	CharsetTChar = "!#$%&'*+-.^_`|~" + rfc5234.CharsetDIGIT + rfc5234.CharsetALPHA
)

// IsValidToken returns whether a string is a valid "token", as defined by §3.2.6.
func IsValidToken(str string) bool {
	return len(str) >= 1 && strings.Trim(str, CharsetTChar) == ""
}

// QuoteString quotes a string in the "quoted-string" syntax, as defined by §3.2.6.
//
// It is a runtime error (panic) to try to quote a string containing the bytes 0x00-0x08, 0x0A-0x1F,
// or 0x7F (the ASCII control characters, and ASCII DEL).
func QuoteString(str string) string {
	// ABNF:
	//     quoted-string  = DQUOTE *( qdtext / quoted-pair ) DQUOTE
	//     qdtext         = HTAB / SP /%x21 / %x23-5B / %x5D-7E / obs-text
	//     obs-text       = %x80-FF
	//     quoted-pair    = "\" ( HTAB / SP / VCHAR / obs-text )

	var ret strings.Builder
	_ = ret.WriteByte('"')

	// RFC 7230 is defined in terms of octets; it is correct for us to iterate over bytes of
	// encoded-utf-8 instead of utf-8 runes.
	for _, b := range []byte(str) {
		switch {
		case b == '\t' || (' ' <= b && b <= '~' && b != '"' && b != '\\') || b >= 0x80:
			// qdtext
			_ = ret.WriteByte(b)
		case b == '"' || b == '\\':
			// quoted-pair
			_ = ret.WriteByte('\\')
			_ = ret.WriteByte(b)
		default: // (b < 0x20 && b != '\t') || b == 0x7F
			// non-printable ASCII characters
			panic(errors.Errorf("non-quotable octet: %#v", b))
		}
	}
	_ = ret.WriteByte('"')
	return ret.String()
}

// ScanQuotedString scans a "quoted-string" (as defined by §3.2.6) from the beginning of a string,
// and returns the un-quoted result, as well as the remainder of the input string.
//
// For example:
//
//     ScanQuotedString(`"quoted\"part" unquoted part`) => (`quoted"part`, ` unquoted part`, nil)
//
// If a syntax error is encountered, ("", "", err) is returned.
func ScanQuotedString(input string) (value, rest string, err error) {
	var ret strings.Builder
	rest = input

	switch {
	case len(rest) == 0:
		return "", "", errors.Errorf("invalid quoted string: expected opening quote, got EOF")
	case rest[0] != '"':
		return "", "", errors.Errorf("invalid quoted string: expected opening quote, got %#v", rest[0])
	default:
		rest = rest[1:]
	}

	var b byte
	for b, rest = rest[0], rest[1:]; len(rest) > 0; b, rest = rest[0], rest[1:] {
		switch {
		case (b < ' ' && b != '\t') || b == 0x7F:
			// invalid byte: ASCII control character or ASCII DEL
			return "", "", errors.Errorf("invalid quoted string: illegal octet: %#v", b)
		case b == '"':
			return ret.String(), rest, nil
		case b == '\\':
			if len(rest) == 0 {
				return "", "", errors.Errorf("invalid quoted string: reached EOF looking for closing quote")
			}
			b = rest[0]
			rest = rest[1:]
			if (b < ' ' && b != '\t') || b == 0x7F {
				// invalid byte: ASCII control character or ASCII DEL
				return "", "", errors.Errorf("invalid quoted string: illegal octet in quoted-pair : %#v", b)
			}
			_ = ret.WriteByte(b)
		default:
			_ = ret.WriteByte(b)
		}
	}
	return "", "", errors.Errorf("invalid quoted string: reached EOF looking for closing quote")
}
