// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/type/matcher/number.proto

package envoy_type_matcher

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gogo/protobuf/types"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = types.DynamicAny{}
)

// define the regex for a UUID once up-front
var _number_uuidPattern = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

// Validate checks the field values on DoubleMatcher with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *DoubleMatcher) Validate() error {
	if m == nil {
		return nil
	}

	switch m.MatchPattern.(type) {

	case *DoubleMatcher_Range:

		{
			tmp := m.GetRange()

			if v, ok := interface{}(tmp).(interface{ Validate() error }); ok {

				if err := v.Validate(); err != nil {
					return DoubleMatcherValidationError{
						field:  "Range",
						reason: "embedded message failed validation",
						cause:  err,
					}
				}
			}
		}

	case *DoubleMatcher_Exact:
		// no validation rules for Exact

	default:
		return DoubleMatcherValidationError{
			field:  "MatchPattern",
			reason: "value is required",
		}

	}

	return nil
}

// DoubleMatcherValidationError is the validation error returned by
// DoubleMatcher.Validate if the designated constraints aren't met.
type DoubleMatcherValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DoubleMatcherValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DoubleMatcherValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DoubleMatcherValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DoubleMatcherValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DoubleMatcherValidationError) ErrorName() string { return "DoubleMatcherValidationError" }

// Error satisfies the builtin error interface
func (e DoubleMatcherValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDoubleMatcher.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DoubleMatcherValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DoubleMatcherValidationError{}
