package rfc7235

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/apro/common/rfc7235/internal/rfc5234"
)

// ScanToken68 scans a "token68" (as defined by §2.1) from the beginning of a string, and returns
// the token68, as well as the remainder of the input string.
//
// If the input does not have a token68 as a prefix, then ("", "", err) is returned, where err
// explains why the prefix is not a token68.
func scanToken68(input string) (string, string, error) {
	// ABNF:
	//     token68        = 1*( ALPHA / DIGIT / "-" / "." / "_" / "~" / "+" / "/" ) *"="
	rest := strings.TrimLeft(strings.TrimLeft(input, rfc5234.CharsetALPHA+rfc5234.CharsetDIGIT+"-._~+/"), "=")
	if rest == input {
		if len(input) == 0 {
			return "", "", errors.New("invalid token68: must not be empty")
		} else {
			return "", "", errors.Errorf("invalid token68: illegal character %c", input[0])
		}
	}
	return strings.TrimSuffix(input, rest), rest, nil
}
