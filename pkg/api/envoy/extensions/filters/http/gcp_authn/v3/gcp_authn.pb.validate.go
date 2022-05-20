// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/extensions/filters/http/gcp_authn/v3/gcp_authn.proto

package gcp_authnv3

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
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
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on GcpAuthnFilterConfig with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *GcpAuthnFilterConfig) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on GcpAuthnFilterConfig with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// GcpAuthnFilterConfigMultiError, or nil if none found.
func (m *GcpAuthnFilterConfig) ValidateAll() error {
	return m.validate(true)
}

func (m *GcpAuthnFilterConfig) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if m.GetHttpUri() == nil {
		err := GcpAuthnFilterConfigValidationError{
			field:  "HttpUri",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetHttpUri()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, GcpAuthnFilterConfigValidationError{
					field:  "HttpUri",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, GcpAuthnFilterConfigValidationError{
					field:  "HttpUri",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetHttpUri()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return GcpAuthnFilterConfigValidationError{
				field:  "HttpUri",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if all {
		switch v := interface{}(m.GetRetryPolicy()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, GcpAuthnFilterConfigValidationError{
					field:  "RetryPolicy",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, GcpAuthnFilterConfigValidationError{
					field:  "RetryPolicy",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetRetryPolicy()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return GcpAuthnFilterConfigValidationError{
				field:  "RetryPolicy",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return GcpAuthnFilterConfigMultiError(errors)
	}

	return nil
}

// GcpAuthnFilterConfigMultiError is an error wrapping multiple validation
// errors returned by GcpAuthnFilterConfig.ValidateAll() if the designated
// constraints aren't met.
type GcpAuthnFilterConfigMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m GcpAuthnFilterConfigMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m GcpAuthnFilterConfigMultiError) AllErrors() []error { return m }

// GcpAuthnFilterConfigValidationError is the validation error returned by
// GcpAuthnFilterConfig.Validate if the designated constraints aren't met.
type GcpAuthnFilterConfigValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e GcpAuthnFilterConfigValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e GcpAuthnFilterConfigValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e GcpAuthnFilterConfigValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e GcpAuthnFilterConfigValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e GcpAuthnFilterConfigValidationError) ErrorName() string {
	return "GcpAuthnFilterConfigValidationError"
}

// Error satisfies the builtin error interface
func (e GcpAuthnFilterConfigValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sGcpAuthnFilterConfig.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = GcpAuthnFilterConfigValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = GcpAuthnFilterConfigValidationError{}

// Validate checks the field values on Audience with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Audience) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Audience with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in AudienceMultiError, or nil
// if none found.
func (m *Audience) ValidateAll() error {
	return m.validate(true)
}

func (m *Audience) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for AudienceMap

	if len(errors) > 0 {
		return AudienceMultiError(errors)
	}

	return nil
}

// AudienceMultiError is an error wrapping multiple validation errors returned
// by Audience.ValidateAll() if the designated constraints aren't met.
type AudienceMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m AudienceMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m AudienceMultiError) AllErrors() []error { return m }

// AudienceValidationError is the validation error returned by
// Audience.Validate if the designated constraints aren't met.
type AudienceValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e AudienceValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e AudienceValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e AudienceValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e AudienceValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e AudienceValidationError) ErrorName() string { return "AudienceValidationError" }

// Error satisfies the builtin error interface
func (e AudienceValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sAudience.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = AudienceValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = AudienceValidationError{}
