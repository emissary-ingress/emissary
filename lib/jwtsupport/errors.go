// Package jwtsupport provides utilities for working around crap in github.com/dgrijalva/jwt-go
package jwtsupport

import (
	"fmt"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
)

// Keep this list in sync with the list of errors in
// github.com/dgrijalva/jwt-go/errors.go
var validationErrorFlagNames = map[uint32]string{
	jwt.ValidationErrorMalformed:        "ValidationErrorMalformed",
	jwt.ValidationErrorUnverifiable:     "ValidationErrorUnverifiable",
	jwt.ValidationErrorSignatureInvalid: "ValidationErrorSignatureInvalid",

	// Standard Claim validation errors
	jwt.ValidationErrorAudience:      "ValidationErrorAudience",
	jwt.ValidationErrorExpired:       "ValidationErrorExpired",
	jwt.ValidationErrorIssuedAt:      "ValidationErrorIssuedAt",
	jwt.ValidationErrorIssuer:        "ValidationErrorIssuer",
	jwt.ValidationErrorNotValidYet:   "ValidationErrorNotValidYet",
	jwt.ValidationErrorId:            "ValidationErrorId",
	jwt.ValidationErrorClaimsInvalid: "ValidationErrorClaimsInvalid",
}

// JWTGoError wraps a jwt-go jwt.ValidationError, because their
// .Error() method produces depressingly poor error messages.
type JWTGoError struct {
	jwt.ValidationError
}

// SanitizeError sanitizes errors coming from jwt-go, because their
// error messages are bad.  Returns nil if the input error is nil.
func SanitizeError(e error) error {
	switch e := e.(type) {
	case nil:
		return nil
	case jwt.ValidationError:
		return &JWTGoError{e}
	case *jwt.ValidationError:
		return &JWTGoError{*e}
	default:
		return e
	}
}

func (e *JWTGoError) Error() string {
	// There are 3 fields we want to extract from the
	// e.ValidationError:
	fieldErrorFlags := e.ValidationError.Errors
	fieldWrappedError := e.ValidationError.Inner
	// e.ValidationError.text is unexported, but .Error() returns
	// it if .Inner is nil.  This will _always_ be non-empty,
	// because if .text is empty, then .Error() returns "token is
	// invalid".
	veCopy := e.ValidationError
	veCopy.Inner = nil
	fieldMessage := veCopy.Error()

	// Format the fieldErrorFlags bitfield as a
	// "(flagA|flagB|flagC)" string.
	fieldErrorFlagsStr := "0"
	if fieldErrorFlags != 0 {
		flagsFound := []string(nil)
		flagsLeft := fieldErrorFlags
		for i := uint(0); flagsLeft != 0; i++ {
			var flag uint32 = 1 << i
			if flagsLeft&(flag) > 0 {
				flagName := validationErrorFlagNames[flag]
				if flagName == "" {
					flagName = fmt.Sprintf("%#08x", flag)
				}
				flagsFound = append(flagsFound, flagName)
				flagsLeft -= flag
			}
		}
		fieldErrorFlagsStr = strings.Join(flagsFound, "|")
	}

	return fmt.Sprintf("Token validation error: %v: errorFlags=%#08x=(%s) wrappedError=(%v)",
		fieldMessage,
		fieldErrorFlags, fieldErrorFlagsStr,
		fieldWrappedError)
}

// Cause implements github.com/pkg/errors.causer
func (e *JWTGoError) Cause() error {
	return e.ValidationError.Inner
}

// SanitizeParse can wrap a call to Parse() or ParseWithClaims();
// calling SanitizeError() on the error.
//
// This can be used with:
//    - jwt.Parse()
//    - jwt.ParseWithClaims()
//    - jwt.Parser.Parse()
//    - jwt.Parser.ParseWithClaims()
func SanitizeParse(token *jwt.Token, err error) (*jwt.Token, error) {
	return token, SanitizeError(err)
}

// SanitizeParseUnverified can wrap a call to
// jwt.Parser.ParseUnverified(); calling SanitizeError() on the error.
func SanitizeParseUnverified(token *jwt.Token, parts []string, err error) (*jwt.Token, []string, error) {
	return token, parts, SanitizeError(err)
}
