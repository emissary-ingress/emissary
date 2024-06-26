//go:build !disable_pgv

// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/extensions/watchdog/profile_action/v3/profile_action.proto

package profile_actionv3

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

// Validate checks the field values on ProfileActionConfig with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ProfileActionConfig) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ProfileActionConfig with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ProfileActionConfigMultiError, or nil if none found.
func (m *ProfileActionConfig) ValidateAll() error {
	return m.validate(true)
}

func (m *ProfileActionConfig) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetProfileDuration()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ProfileActionConfigValidationError{
					field:  "ProfileDuration",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ProfileActionConfigValidationError{
					field:  "ProfileDuration",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetProfileDuration()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ProfileActionConfigValidationError{
				field:  "ProfileDuration",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if utf8.RuneCountInString(m.GetProfilePath()) < 1 {
		err := ProfileActionConfigValidationError{
			field:  "ProfilePath",
			reason: "value length must be at least 1 runes",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	// no validation rules for MaxProfiles

	if len(errors) > 0 {
		return ProfileActionConfigMultiError(errors)
	}

	return nil
}

// ProfileActionConfigMultiError is an error wrapping multiple validation
// errors returned by ProfileActionConfig.ValidateAll() if the designated
// constraints aren't met.
type ProfileActionConfigMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ProfileActionConfigMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ProfileActionConfigMultiError) AllErrors() []error { return m }

// ProfileActionConfigValidationError is the validation error returned by
// ProfileActionConfig.Validate if the designated constraints aren't met.
type ProfileActionConfigValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ProfileActionConfigValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ProfileActionConfigValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ProfileActionConfigValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ProfileActionConfigValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ProfileActionConfigValidationError) ErrorName() string {
	return "ProfileActionConfigValidationError"
}

// Error satisfies the builtin error interface
func (e ProfileActionConfigValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sProfileActionConfig.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ProfileActionConfigValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ProfileActionConfigValidationError{}
